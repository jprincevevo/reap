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
│   └── update.go        # `reap update` subcommand (self-update via go-selfupdate)
├── config/
│   └── config.go        # Config struct, Load/Save, path resolution
├── tui/
│   ├── flow.go          # Combined group→repo flow model (single program, no flash)
│   ├── groups.go        # Group-selection list screen
│   ├── repos.go         # Repo-selection (multi-select) screen
│   ├── cloning.go       # Parallel cloning progress screen
│   ├── confirm.go       # Yes/No dialog (used before cloning inside a git repo)
│   ├── remove.go        # Repo-removal selection screen
│   └── groups_add.go    # Repo multi-select screen used when creating a group
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
default_depth: 1           # optional; used as --depth fallback (see below)
repos:
  - url: https://github.com/owner/repo.git
    selected: true
    groups:
      - name: my-group
        selected: true
```

- `config.Load()` returns `(*Config, bool, error)` — the bool is `true` when the file was just created
- `config.Save(cfg)` marshals back to YAML and writes atomically via a temp file + `os.Rename`
- `cfg.HasGroups()` returns true if any repo has at least one group; used by the root command to decide whether to show the group-selection screen
- `cfg.DefaultDepth` is applied as a fallback for the `--depth` flag: if `--depth` is `0` (not set by the user) and `default_depth` is nonzero in the config, `default_depth` is used for all clones in that run

---

## TUI architecture

Most TUI screens follow the standard Bubble Tea pattern: a model struct with `Init()`, `Update()`, and `View()` methods, plus a `New*Model()` constructor and an `Initial*Model()` function that runs `tea.NewProgram(m).Run()` and extracts the result from the final model state.

The exception is the group→repo selection flow, which uses a **parent model** (`flowModel` in `tui/flow.go`) — see below.

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

Each list-based screen (groups, repos, remove, groups_add) has:
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

### `flowModel` — flash-free multi-screen parent (`tui/flow.go`)

Running two `tea.Program` instances back-to-back (group screen then repo screen) causes a visible flash: the first program's `close()` unconditionally exits the alt screen, briefly exposing the normal terminal before the second program enters alt screen.

`flowModel` solves this by hosting both screens inside a **single** `tea.Program`. The key trick is in `Update`: when the group model sets `m.choice` (user pressed enter), `flowModel` intercepts the `tea.Quit` the sub-model would have propagated, transitions `screen` to `flowScreenRepo`, and returns `nil` — keeping the program alive. The repo model's `goBack` case is handled symmetrically. Only an actual quit (`q`, `ctrl+c`) or a confirmed repo selection propagates `tea.Quit` to end the program.

When transitioning, `flowModel` immediately sizes the new sub-model using the stored `m.width` and sets `ready = true`, so the next `View()` renders the list directly with no blank frame.

`cmd/root.go` calls `tui.InitialFlowModel(cfg)` when `cfg.HasGroups()`, and falls back to the standalone `tui.InitialRepoModel(cfg, "Show All")` when there are no groups (no transition, no flash risk).

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

### Cloning screen (`tui/cloning.go`)

This screen runs in inline mode (no alt screen) intentionally — the spinner output and final result summary are meant to persist in the terminal after the program exits. It uses a package-level `var p *tea.Program` so that goroutines performing clones can call `p.Send(...)` to send progress messages back to the model.

---

## `cmd/root.go` — root command behaviour

The root `reap` command (no subcommand) runs the following sequence:

1. **Self-update check** — `checkForUpdates()` is launched as a goroutine immediately. A `close(updateDone)` channel signals when it finishes (including its 2-second sleep). The check is a no-op when `version.Version == "dev"` so local `go run .` builds stay quiet.
2. **Config load** — `config.Load()` is called; prints a message if the file was just created.
3. **Early exits** — returns immediately if no repos are configured, or if bare URL args were provided (direct clone, no TUI).
4. **`<-updateDone`** — blocks until the update goroutine finishes before opening any TUI screen. This prevents `fmt.Printf` from the banner racing with Bubble Tea's terminal ownership.
5. **TUI flow** — runs `InitialFlowModel` (groups) or `InitialRepoModel` (no groups), then `cloneRepos`.

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
