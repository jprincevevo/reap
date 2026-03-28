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
│   ├── config.go        # `reap config {export,import}` subcommands
│   ├── status.go        # `reap status` subcommand
│   └── update.go        # `reap update` subcommand (self-update via go-selfupdate)
├── config/
│   ├── config.go        # Config struct, Load/Save, path resolution, HasGroups
│   └── mutations.go     # Methods on *Config: AddRepo, RemoveRepo, ApplyGroupToRepos, etc.
├── tui/
│   ├── app.go           # Root appModel — single program covering all screens
│   ├── groups.go        # Group-selection list screen; shared constants, helpers, and styles
│   ├── repos.go         # Repo-selection (multi-select) screen
│   ├── cloning.go       # Parallel cloning progress screen
│   ├── confirm.go       # Yes/No dialog (used before cloning inside a git repo)
│   ├── remove.go        # Repo-removal selection screen
│   ├── groups_add.go    # Repo multi-select screen used when creating a group
│   ├── manage_groups.go # Interactive group management TUI (reap group)
│   ├── manage_repos.go  # Interactive repo management TUI (reap repo)
│   ├── settings.go      # Settings form screen (depth, dir, pull toggle)
│   ├── prompt.go        # Text-input prompt sub-model shared by management screens
│   ├── paste_add.go     # Paste/drop URL confirmation + optional group selection
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

### Config mutation methods (`config/mutations.go`)

Business logic that operates on `*Config` lives here as methods, importable by both `cmd` and `tui` without circular dependencies:

| Method | Behaviour |
|---|---|
| `cfg.AddRepo(url) bool` | Appends a new repo (`Selected: true`); no-op and returns `false` if URL already present |
| `cfg.RemoveRepo(url)` | Removes every repo with the given URL in place |
| `cfg.ApplyGroupToRepos(name, urls) int` | Adds group to each repo whose URL is in `urls` (skips if already present); returns count modified |
| `cfg.RemoveGroupFromAllRepos(name) int` | Removes group from every repo that has it; returns total entries removed |
| `cfg.UniqueGroupNames() []string` | Deduplicated group names in first-seen order; always non-nil |

---

## TUI architecture

Most TUI screens follow the standard Bubble Tea pattern: a model struct with `Init()`, `Update()`, and `View()` methods, plus a `New*Model()` constructor and an `Initial*Model()` function that runs `tea.NewProgram(m).Run()` and extracts the result from the final model state.

The exception is the main `reap` command, which uses a **single persistent program** (`appModel` in `tui/app.go`) covering all screens for the full process lifetime — see below.

### Shared constants, helpers, and styles (defined in `tui/groups.go`)

```go
// Constants
listHeight       = 14   // shared fixed height for all list screens
listDefaultWidth = 20   // initial width before WindowSizeMsg arrives
showAllRepos     = "Show all repos"  // sentinel group name meaning "no filter"

// Color palette (light/dark adaptive)
colorAccent  // amber
colorSuccess // teal
colorError   // red
colorDim     // gray
colorMuted   // peach/dark amber
colorPurple  // purple (confirm dialog border)

// Styles (package-level vars used by every file in package tui)
titleStyle, itemStyle, selectedItemStyle, cursorStyle
paginationStyle, helpStyle, quitTextStyle
dimHelpStyles   // help.Styles override for all list help bars
```

**`newList(items, delegate, title)`** — shared constructor that applies all standard styles and settings to a `list.Model`. Every `build*List()` method and list constructor calls this instead of duplicating the 7-line setup block.

**`logSaveErr(err)`** — prints `"Error saving config: <err>"` to stdout when err is non-nil. Used at every `config.Save` call site in the TUI (can't return errors from Update).

### List screens pattern

Each list-based screen (groups, repos, remove, groups_add, management screens) has:
- A custom `ItemDelegate` that renders items using `itemStyle` / `selectedItemStyle`
- A `WindowSizeMsg` handler that calls `m.list.SetSize(msg.Width, listHeight)` — **use `SetSize`, not `SetWidth`**, to ensure the paginator's `PerPage` is recalculated correctly at the new terminal width
- A call to `newList(items, delegate, title)` in the constructor — do not inline the style setup

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
    appScreenPasteAdd                  // paste/drop URL confirmation + group selection
)
```

The home screen is the group-selection list (`groupModel`). From there:
- `enter` → repo selection → clone flow → program exits
- `g` → manage groups (returns on `q`/`esc`)
- `r` → manage repos (returns on `q`/`esc`)
- `s` → settings (returns on `esc` or after last field)
- paste/drop a GitHub URL on any screen → `pasteAddModel` confirmation flow
- `q`/`esc` → exits program

The `Update` method intercepts `g`, `r`, `s` **before** forwarding to `groupModel`, but only when the list is not in filter mode (`list.FilterState() != list.Filtering`). Sub-models signal the parent via `goBack bool` rather than propagating `tea.Quit`; the parent discards the `tea.Quit` and transitions screens.

**`appModel.goHome()`** is a helper method that rebuilds the home `groupModel` from the current config and transitions `screen` to `appScreenHome`. It is called in every "return to home" branch — do not inline the 5-line block again.

`cmd/root.go` calls `tui.InitialAppModel(cfg)` for the main command, which returns the selected repo URLs when the user confirms a clone. The collection loop uses **`fm.repo.list.VisibleItems()`** (not `Items()`) so that an active filter restricts which repos are cloned.

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

### Filter-aware repo selection

When the user applies a filter on the repo-selection screen and presses Enter, **only visible repos are candidates for cloning** — hidden repos are excluded regardless of their selected state. This is enforced in both `InitialRepoModel` and `InitialAppModel` by iterating `m.list.VisibleItems()` instead of `m.list.Items()`.

The Enter help label also changes contextually: `"confirm"` when unfiltered, `"clone visible"` when `FilterState() == FilterApplied`. This is set dynamically in `repoModel.View()` (value receiver, so it only affects the local render copy) via `m.list.AdditionalShortHelpKeys`.

Pressing `esc` while a filter is active clears the filter (restoring all items) rather than going back to the previous screen.

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

Config mutation logic lives in `config/mutations.go` as methods on `*Config` (no I/O, no cobra dependency) and is covered by `config/mutations_test.go`. The cobra `Run` closures in `cmd/` are thin orchestrators (load → call `cfg.Method()` → save → print) and have no dedicated tests; their correctness is implicitly covered by the config package tests.

### Config mutation tests (`config/mutations_test.go`)

Table-driven, fully-parallelised subtests (`t.Parallel()` at both levels) covering every method:

| Test | Covers |
|---|---|
| `TestConfig_AddRepo` | append, duplicate skip, empty config |
| `TestConfig_RemoveRepo` | removal, no-op on unknown URL, empty config |
| `TestConfig_ApplyGroupToRepos` | add group, skip if already present, unselected repos untouched, count |
| `TestConfig_RemoveGroupFromAllRepos` | removal across repos, count, no-op, unrelated groups survive |
| `TestConfig_UniqueGroupNames` | dedup, first-seen order, empty/no-group configs return non-nil slice |

### Running tests

```bash
make test        # go test ./...
make test-v      # go test ./... -v  (verbose, shows each subtest)
make check       # gofmt + go vet + go test ./...  (full pre-push gate)
```

### TUI model tests (`tui/models_test.go`)

TUI model logic is tested headlessly by calling `Update` with hand-crafted `tea.KeyPressMsg` / `tea.WindowSizeMsg` values and asserting on the returned model state — no terminal or `tea.Program` required. Covered areas:

- `repoItem` — `Description()` and `FilterValue()`
- `NewRepoModel` — `showAllRepos` inclusion, group filtering, per-group `Selected` state, unknown group
- `NewGroupModel` — `showAllRepos` always first, group deduplication, empty config
- `repoModel.Update` — quit keys, esc (goBack), enter, space toggle
- `repoModel` filter scope — `SetFilterText` puts model into `FilterApplied`; `VisibleItems()` returns only matching repos while `Items()` retains all
- `groupModel.Update` — quit keys, enter sets choice
- `confirmModel.Update` — default button, left/right, enter on Yes/No, quit keys
- `appModel.Update` — WindowSizeMsg stores width, g/r/s key transitions, enter on group list → repo screen, goBack from management → home screen
- `appModel` filter scope — same `VisibleItems()` contract verified through the appModel path
- `promptModel` constructor — `Focused()` returns true, `ready` is true
- `manageGroupModel` — list screen key routing (enter/a/r/d/q), q sets goBack (not done), detail screen key routing (esc/q/r), `buildDetailList` content and title
- `manageRepoModel` — list screen key routing (enter/a/q), q sets goBack (not done), detail screen key routing (esc/a/r), add-to-group esc, `buildDetailList` content, `buildGroupList` exclusion logic
- `settingsModel` — esc sets goBack, ctrl+c sets quitting, enter on depth advances to dir field, enter on dir advances to pull field, space toggles pull, enter on pull sets goBack

### What is not tested

- The cloning screen (`tui/cloning.go`) is I/O-bound and requires a live terminal
- The cobra `Run` closures in `cmd/` — thin orchestrators covered implicitly by config package tests
- `settingsModel` config persistence (`config.Save` calls) — requires a real filesystem

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
