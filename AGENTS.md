# Agent Onboarding: reap

`reap` is a Go CLI tool for batch-cloning Git repositories from a YAML config file. It provides a Bubble Tea TUI for interactive repo/group selection followed by parallel cloning.

---

## Project layout

```
reap/
├── main.go              # Entry point — calls cmd.Execute()
├── cmd/
│   ├── root.go          # Root cobra command, main run loop, self-update check
│   ├── repo.go          # `reap repo {add,remove,list}` subcommands
│   ├── group.go         # `reap group {add,remove,list}` subcommands
│   ├── logic.go         # Pure business-logic helpers (no I/O, no cobra) — testable in isolation
│   ├── logic_test.go    # Table-driven unit tests for every helper in logic.go
│   └── update.go        # `reap update` subcommand (self-update via go-selfupdate)
├── config/
│   └── config.go        # Config struct, Load/Save, path resolution
├── tui/
│   ├── app.go           # Root appModel — single program covering all screens
│   ├── groups.go        # Group-selection list screen (home screen / clone entry point)
│   ├── repos.go         # Repo-selection (multi-select) screen
│   ├── cloning.go       # Parallel cloning progress screen
│   ├── confirm.go       # Yes/No dialog (used before cloning inside a git repo)
│   ├── remove.go        # Repo-removal selection screen
│   ├── groups_add.go    # Repo multi-select screen used when creating a group
│   ├── manage_groups.go # Interactive group management TUI (reap group)
│   ├── manage_repos.go  # Interactive repo management TUI (reap repo)
│   ├── settings.go      # Settings form screen (depth, dir, pull toggle)
│   ├── prompt.go        # Text-input prompt sub-model shared by management screens
│   └── models_test.go   # Headless unit tests for TUI model constructors and Update logic
├── version/
│   └── version.go       # `var Version = "dev"` — injected at build time via ldflags
├── .workspace/          # Ephemeral agent workspace — see below
└── .github/workflows/
    ├── tag.yml           # Auto-bumps semver tag on every push to main
    └── release.yml       # Runs GoReleaser on every v* tag push
```

---

## Ephemeral workspace (`.workspace/`)

`.workspace/` is a scratch directory for agents. It is git-ignored. Use it freely to:

- Write planning notes, task breakdowns, or design drafts before implementing
- Store intermediate artifacts (e.g. a diff you want to review, a snippet you're iterating on)
- Leave behind a `MANIFEST.md` or similar log if tracking multi-step work across sessions

Nothing in `.workspace/` is part of the production codebase and it is never committed. Feel free to create, modify, or delete files there without ceremony.

---

## Config system

- **Location:** `~/.config/reap/config.yaml` (Unix) or `%AppData%\reap\config.yaml` (Windows)
- **Auto-created** as an empty file on first run if missing
- **Structure:**

```yaml
default_depth: 1           # optional; used as --depth fallback
default_dir: ~/projects    # optional; used as --dir fallback
default_pull: false        # optional; used as --pull fallback
repos:
  - url: https://github.com/owner/repo.git
    selected: true
    groups:
      - name: my-group
        selected: true
```

- `config.Load()` returns `(*Config, bool, error)` — the bool is `true` when the file was just created
- `config.Save(cfg)` marshals back to YAML and writes atomically via a temp file + `os.Rename`
- `cfg.HasGroups()` returns true if any repo has at least one group
- `cfg.DefaultDepth` is applied as a fallback for `--depth`; `cfg.DefaultDir` for `--dir`; `cfg.DefaultPull` for `--pull` — all applied only when the flag was not explicitly set by the user

---

## TUI architecture

Most TUI screens follow the standard Bubble Tea pattern: a model struct with `Init()`, `Update()`, and `View()` methods, plus a `New*Model()` constructor and an `Initial*Model()` function that runs `tea.NewProgram(m).Run()` and extracts the result from the final model state.

The exception is the main `reap` command, which uses a **single persistent program** (`appModel` in `tui/app.go`) covering all screens for the full process lifetime — see below.

### Shared styles (defined in `tui/groups.go`)

```go
itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
titleStyle        = lipgloss.NewStyle().MarginLeft(2)
paginationStyle   = list.DefaultStyles(false).PaginationStyle.PaddingLeft(4)
helpStyle         = list.DefaultStyles(false).HelpStyle.PaddingLeft(4).PaddingBottom(1)
quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
listHeight        = 14   // shared fixed height for all list screens
```

These are package-level vars used by every file in `package tui`.

### List screens pattern

Each list-based screen (groups, repos, remove, groups_add, management screens) has:
- A custom `ItemDelegate` that renders items using `itemStyle` / `selectedItemStyle`
- A `WindowSizeMsg` handler that calls `m.list.SetSize(msg.Width, listHeight)` — **use `SetSize`, not `SetWidth`**, to ensure the paginator's `PerPage` is recalculated correctly at the new terminal width

### Critical TUI rendering rule (v2)

**Always set `v.AltScreen = true` on the `tea.View` returned from `View()` while a list screen is actively displayed.** This is the Bubble Tea v2 idiom — `tea.WithAltScreen()` was a v1 program option that no longer exists.

```go
func (m groupModel) View() tea.View {
    // ...
    v := tea.NewView("\n" + m.list.View())
    v.AltScreen = true  // v2: declarative, set per-frame
    return v
}
```

The alt screen should only be set while the list is actually visible — do not set it on the "quitting" or "choice confirmed" frames, so the terminal is restored cleanly on exit. Also set it on the initial `!m.ready` frame (before `WindowSizeMsg` arrives) so the program enters alt screen immediately and avoids a one-frame flash of the underlying terminal.

### `appModel` — single persistent root model (`tui/app.go`)

Running multiple `tea.Program` instances back-to-back causes a visible flash: each program's `close()` unconditionally exits the alt screen, briefly exposing the normal terminal before the next program enters.

`appModel` solves this by hosting **every screen** inside a single `tea.Program` for the full lifetime of the `reap` process. The screen enum is:

```go
type appScreen int
const (
    appScreenHome     appScreen = iota // group selection — always the anchor
    appScreenRepo                      // repo selection (after group chosen)
    appScreenGroups                    // manageGroupModel
    appScreenRepos                     // manageRepoModel
    appScreenSettings                  // settingsModel
)
```

The home screen is the group-selection list (`groupModel`). From there:
- `enter` → repo selection → clone flow → program exits
- `g` → manage groups (returns on `q`/`esc`)
- `r` → manage repos (returns on `q`/`esc`)
- `s` → settings (returns on `esc` or after last field)
- `q`/`esc` → exits program

The `Update` method intercepts `g`, `r`, `s` **before** forwarding to `groupModel`, but only when the list is not in filter mode (`list.FilterState() != list.Filtering`). Sub-models signal the parent via `goBack bool` rather than propagating `tea.Quit`; the parent discards the `tea.Quit` and transitions screens.

`cmd/root.go` calls `tui.InitialAppModel(cfg)` for the main command, which returns the selected repo URLs when the user confirms a clone.

### Onboarding screens

Three onboarding views replace empty lists when there is nothing to show:

- **Home screen (`groupModel.View()`)**: if `m.repoCount == 0`, shows a welcome message with instructions to press `r` or run `reap repo add <url>`.
- **Group management (`manageGroupModel.View()`)**: if `len(m.list.Items()) == 0`, shows an explanation of groups with `a` to create the first one.
- **Repo management (`manageRepoModel.View()`)**: if `len(m.list.Items()) == 0`, shows instructions to press `a` or run `reap repo add <url>`.

### Key naming gotcha

`KeyPressMsg.String()` (via the ultraviolet library) does **not** return the raw character for every key:

| Key | `.String()` returns |
|---|---|
| Spacebar | `"space"` — **not** `" "` |
| Escape | `"esc"` — **not** `"\x1b"` |
| Regular letters/symbols | the character itself, e.g. `"q"`, `"/"` |

Always use the named form in switch cases. Using `" "` for space means the handler silently never fires.

### `ErrGoBack` sentinel

`tui.ErrGoBack` is a package-level `errors.New` value exported from `repos.go`. `InitialRepoModel` returns it when the user presses `esc` in standalone mode (no-groups path). `flowModel` handles go-back internally and never surfaces `ErrGoBack` to callers.

### Management screens (`tui/manage_groups.go`, `tui/manage_repos.go`)

`reap group` and `reap repo` (with no subcommand) launch interactive management TUIs backed by `manageGroupModel` and `manageRepoModel` respectively. Both follow a multi-screen parent pattern (similar to `appModel`) where all screens live inside a single `tea.Program`.

When embedded in `appModel` (reached via `g`/`r` from the home screen), these models signal return via `goBack bool` instead of quitting the program. When the user presses `q` or `esc` at the top-level list screen (`mgScreenList` / `mrScreenList`), `goBack = true` and `tea.Quit` is returned. `appModel` intercepts this, rebuilds the home screen from the updated config, and transitions back without the program exiting. Pressing `ctrl+c` from any screen is a hard quit (no `goBack` flag) that propagates `tea.Quit` all the way to the program.

Both models expose a package-level `newManageGroupModel(cfg)` / `newManageRepoModel(cfg)` constructor for use by `appModel` during transitions.

**`reap group` key map:**

| Key | Screen | Action |
|-----|--------|--------|
| `enter` | Group list | Open group detail (repos in that group) |
| `a` | Group list | Prompt for new name → repo multi-select → save |
| `r` | Group list | Prompt to rename selected group → save |
| `d` | Group list | Type-yes confirmation → delete group from all repos |
| `r` | Group detail | Type-yes confirmation → remove highlighted repo from group |
| `esc` / `q` | Group detail | Back to group list |
| `q` / `ctrl+c` | Any | Exit |

**`reap repo` key map:**

| Key | Screen | Action |
|-----|--------|--------|
| `enter` | Repo list | Open repo detail (groups the repo belongs to) |
| `a` | Repo list | Prompt for new URL → save |
| `d` | Repo list | Repo-removal selection screen → save |
| `a` | Repo detail | List of groups the repo is NOT yet in → `enter` adds |
| `r` | Repo detail | Type-yes confirmation → remove from highlighted group |
| `esc` / `q` | Repo detail / add-to-group | Back to previous screen |
| `q` / `ctrl+c` | Any | Exit |

### `promptModel` — embedded text-input (`tui/prompt.go`)

`promptModel` wraps `textinput.Model` and is used as a sub-model by the management screens. Two critical properties:

1. **Focus must be set in the constructor, not in `Init()`.** `textinput.Model.Focus()` has a pointer receiver. When called inside `Init()` (value receiver), Go operates on a copy — the focus state is set on a throwaway copy and the real model stays unfocused. `textinput.Update` silently drops all input when `!m.focus`. Fix: call `ti.Focus()` directly on the local variable inside `NewPromptModel` before copying it into the struct.

2. **`ready` is initialised to `true` in the constructor.** When the prompt is embedded in a running program (transitions between screens), no new `WindowSizeMsg` arrives, so the `ready` guard would keep the prompt blank forever. Starting `ready: true` makes it render immediately.

When a prompt sub-model quits (user pressed `enter` or `esc`), the parent's `update*` method checks `m.prompt.quitting` and **discards the `tea.Quit` cmd** returned by the prompt, returning `nil` instead. This keeps the parent program alive for the screen transition.

### Cloning screen (`tui/cloning.go`)

This screen runs in inline mode (no alt screen) intentionally — the spinner output and final result summary are meant to persist in the terminal after the program exits. It uses a package-level `var p *tea.Program` so that goroutines performing clones can call `p.Send(...)` to send progress messages back to the model.

---

## `cmd/root.go` — root command behaviour

The root `reap` command (no subcommand) runs the following sequence:

1. **Self-update check** — `checkForUpdates()` is launched as a goroutine immediately. A `close(updateDone)` channel signals when it finishes (including its 2-second sleep). The check is a no-op when `version.Version == "dev"` so local `go run .` builds stay quiet.
2. **Config load** — `config.Load()` is called; prints a message if the file was just created.
3. **Config defaults** — `DefaultDepth`, `DefaultDir`, and `DefaultPull` from the config are applied when the corresponding CLI flags were not explicitly set.
4. **Early exit for direct args** — if bare URL args were provided, clones them directly without a TUI.
5. **`<-updateDone`** — blocks until the update goroutine finishes before opening any TUI screen.
6. **TUI** — runs `InitialAppModel(cfg)`, then `cloneRepos` if repos were selected.

Note: the empty-config guard (`len(cfg.Repos) == 0`) has been removed. The TUI now handles the empty state with an onboarding screen.

**Flags:**

| Flag | Default | Behaviour |
|---|---|---|
| `--depth N` | `0` | Clone depth passed to `git clone --depth`. `0` = full clone, unless overridden by `default_depth` in config. |
| `--version` / `-v` | `false` | Prints version and exits immediately (no TUI, no update check). |

**Duplicate and empty-state guards (cmd subcommands):**

- `reap repo add <url>` — silently skips if the URL already exists in the config
- `reap repo remove` / `reap repo list` — prints "No repositories configured." and exits if the config is empty
- `reap group add <name>` — skips appending a group that already exists on a given repo
- `reap group remove <name>` — prints "Group not found" and exits without saving if the name matched nothing
- `reap group list` — prints "No groups configured." if no groups are defined

---

## Release pipeline

Pushing to `main` triggers two chained workflows:

1. **`tag.yml`** — uses `anothrNick/github-tag-action` with `DEFAULT_BUMP: patch`. It reads `REAP_TAG_TOKEN` (a PAT with repo write scope — different from `GITHUB_TOKEN` because GitHub won't trigger downstream workflows from events caused by `GITHUB_TOKEN`). Bumps the version tag (e.g. `v0.0.7` → `v0.0.8`) and pushes it. Control the bump type with `#major` or `#minor` in the commit message.

2. **`release.yml`** — triggers on `v*` tags. Runs GoReleaser, which:
   - Builds binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`, `windows/arm64`
   - Injects the version via `-X github.com/jprincevevo/reap/version.Version={{.Version}}`
   - Creates a GitHub Release with `.tar.gz` / `.zip` archives and `checksums.txt`

**To release:** just push to `main`. Verify the `REAP_TAG_TOKEN` secret is valid in repo Settings → Secrets and variables → Actions if releases stop appearing.

---

## Testing

### Architecture

Business logic in `cmd/repo.go` and `cmd/group.go` is extracted into **pure helper functions** in `cmd/logic.go`. Each helper:

- Takes a `*config.Config` and plain value arguments
- Returns a plain result (bool, slice, or int)
- Performs **no I/O** — no `config.Load`/`config.Save`, no `fmt.Print`, no cobra dependency

The cobra `Run` closures are thin orchestrators: load config → call helper → save → print. This split makes the logic trivially unit-testable without a real filesystem or command invocation.

### Helpers (`cmd/logic.go`)

| Function | Behaviour |
|---|---|
| `addRepo(cfg, url) bool` | Appends a new repo (`Selected: true`); returns `false` if URL already present |
| `removeRepo(cfg, url) []Repo` | Returns a new slice with the given URL removed |
| `listRepos(cfg) []string` | Returns all repo URLs in order |
| `applyGroupToRepos(cfg, name, urls) int` | Adds a group to each selected repo (skips if already present); returns count modified |
| `removeGroupFromAllRepos(cfg, name) int` | Removes a group from every repo; returns total entries removed |
| `uniqueGroupNames(cfg) []string` | Deduplicates group names across all repos, first-seen order; always non-nil |

### Running tests

```bash
make test        # go test ./...
make test-v      # go test ./... -v  (verbose, shows each subtest)
make check       # gofmt + go vet + go test ./...  (full pre-push gate)
```

`cmd/logic_test.go` contains 21 table-driven, fully-parallelised subtests (`t.Parallel()` at both the top level and inside each `t.Run`). All helpers are covered.

### TUI model tests (`tui/models_test.go`)

TUI model logic is tested headlessly by calling `Update` with hand-crafted `tea.KeyPressMsg` / `tea.WindowSizeMsg` values and asserting on the returned model state — no terminal or `tea.Program` required. Covered areas:

- `repoItem` — `Description()` and `FilterValue()`
- `NewRepoModel` — Show All inclusion, group filtering, per-group `Selected` state, unknown group
- `NewGroupModel` — "Show All" always first, group deduplication, empty config
- `repoModel.Update` — quit keys, esc (goBack), enter, space toggle
- `groupModel.Update` — quit keys, enter sets choice
- `confirmModel.Update` — default button, left/right, enter on Yes/No, quit keys
- `appModel.Update` — WindowSizeMsg stores width, g/r/s key transitions, enter on group list → repo screen, goBack from management → home screen
- `promptModel` constructor — `Focused()` returns true, `ready` is true
- `manageGroupModel` — list screen key routing (enter/a/r/d/q), q sets goBack (not done), detail screen key routing (esc/q/r), `buildDetailList` content and title
- `manageRepoModel` — list screen key routing (enter/a/q), q sets goBack (not done), detail screen key routing (esc/a/r), add-to-group esc, `buildDetailList` content, `buildGroupList` exclusion logic
- `settingsModel` — esc sets goBack, ctrl+c sets quitting, enter on depth advances to dir field, enter on dir advances to pull field, space toggles pull, enter on pull sets goBack

### What is not tested

- The cloning screen (`tui/cloning.go`) is I/O-bound and requires a live terminal
- The cobra `Run` closures in `cmd/repo.go` and `cmd/group.go` are thin orchestrators (load → call helper → save → print); their correctness is implicitly covered by the logic helper tests
- `settingsModel` config persistence (`config.Save` calls) — these require a real filesystem and are covered by integration testing

---

## Building and testing locally

```bash
# Run without building
go run .

# Build a test binary (doesn't conflict with any installed reap)
go build -o reap_test . && ./reap_test

# Build with version injected
go build -ldflags="-X github.com/jprincevevo/reap/version.Version=v0.0.99" -o reap_test .

# Clean up test binary
rm reap_test
```

Useful flag combinations:

```bash
go run . --version            # print version ("dev" for local builds)
go run . --depth 1            # shallow clone (depth 1)
```

Makefile targets (prefer these over raw `go` commands):

```bash
make test        # run all tests
make test-v      # run all tests, verbose
make check       # fmt + vet + test (run before pushing)
make lint        # go vet only
make fmt         # gofmt -w .
make tidy        # go mod tidy
make build       # produce ./reap_test binary
make clean       # remove ./reap_test
```

To exercise the group-selection TUI you need repos with groups in the config. The quickest way to set that up:

```bash
go run . repo add https://github.com/some/repo.git
go run . group add my-group   # launches TUI to assign repos to the group
go run .                      # now shows group-selection screen first (flowModel path)
```

The no-groups path (single repo-selection screen) is the default when no `group add` has been run.

---

## Key dependencies

| Package | Role |
|---|---|
| `charm.land/bubbletea/v2` | TUI event loop and renderer |
| `charm.land/bubbles/v2` | `list`, `spinner`, `help` components |
| `charm.land/lipgloss/v2` | Styling |
| `github.com/spf13/cobra` | CLI command routing |
| `gopkg.in/yaml.v3` | Config file marshaling |
| `github.com/creativeprojects/go-selfupdate` | Self-update via GitHub Releases |
