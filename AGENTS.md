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
│   ├── groups.go        # Group-selection list screen
│   ├── repos.go         # Repo-selection (multi-select) screen
│   ├── cloning.go       # Parallel cloning progress screen
│   ├── confirm.go       # Yes/No dialog (used before cloning inside a git repo)
│   ├── remove.go        # Repo-removal selection screen
│   └── groups_add.go    # Repo multi-select screen used when creating a group
├── version/
│   └── version.go       # `var Version = "dev"` — injected at build time via ldflags
└── .github/workflows/
    ├── tag.yml           # Auto-bumps semver tag on every push to main
    └── release.yml       # Runs GoReleaser on every v* tag push
```

---

## Config system

- **Location:** `~/.config/reap/config.yaml` (Unix) or `%AppData%\reap\config.yaml` (Windows)
- **Auto-created** as an empty file on first run if missing
- **Structure:**

```yaml
repos:
  - url: https://github.com/owner/repo.git
    selected: true
    groups:
      - name: my-group
        selected: true
```

- `config.Load()` returns `(*Config, bool, error)` — the bool is `true` when the file was just created
- `config.Save(cfg)` marshals back to YAML and writes atomically
- `cfg.HasGroups()` returns true if any repo has at least one group; used by the root command to decide whether to show the group-selection screen

---

## TUI architecture

All TUI screens follow the standard Bubble Tea pattern: a model struct with `Init()`, `Update()`, and `View()` methods, plus a `New*Model()` constructor and an `Initial*Model()` function that creates the program, runs it, and extracts the result from the final model state.

### Shared styles (defined in `tui/groups.go`)

```go
itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
titleStyle        = lipgloss.NewStyle().MarginLeft(2)
paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
listHeight        = 14   // shared fixed height for all list screens
```

These are package-level vars used by every file in `package tui`.

### List screens pattern

Each list-based screen (groups, repos, remove, groups_add) has:
- A custom `ItemDelegate` that renders items using `itemStyle` / `selectedItemStyle`
- A `WindowSizeMsg` handler that calls `m.list.SetSize(msg.Width, listHeight)` — **use `SetSize`, not `SetWidth`**, to ensure the paginator's `PerPage` is recalculated correctly at the new terminal width
- `tea.WithAltScreen()` on the programs that are part of the main selection flow (`InitialGroupModel`, `InitialRepoModel`) to avoid Bubbletea inline-rendering ghost artifacts

### Critical TUI rendering rule

**Always use `tea.WithAltScreen()` for list-based selection screens.** Bubbletea's inline renderer (no alt screen) only appends `EraseLineRight` to lines when `r.width > 0`, which is set by the first `WindowSizeMsg`. Any renders that happen before that message arrives leave old terminal content in place, causing visible ghost lines that grow with each keypress.

The two screens that have NOT yet been updated to use alt screen are:
- `tui/remove.go` (`InitialRemoveModel`)
- `tui/groups_add.go` (`InitialGroupAddModel`)

These have the same latent bug and should be updated consistently if they are modified.

### Cloning screen (`tui/cloning.go`)

This screen runs in inline mode (no alt screen) intentionally — the spinner output and final result summary are meant to persist in the terminal after the program exits. It uses a package-level `var p *tea.Program` so that goroutines performing clones can call `p.Send(...)` to send progress messages back to the model.

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

To exercise the group-selection TUI you need repos with groups in the config. The quickest way to set that up:

```bash
go run . repo add https://github.com/some/repo.git
go run . group add my-group   # launches TUI to assign repos to the group
go run .                      # now shows group-selection screen first
```

---

## Key dependencies

| Package | Role |
|---|---|
| `github.com/charmbracelet/bubbletea` | TUI event loop and renderer |
| `github.com/charmbracelet/bubbles` | `list`, `spinner`, `help` components |
| `github.com/charmbracelet/lipgloss` | Styling |
| `github.com/spf13/cobra` | CLI command routing |
| `gopkg.in/yaml.v3` | Config file marshaling |
| `github.com/creativeprojects/go-selfupdate` | Self-update via GitHub Releases |
