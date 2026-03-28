# reap

[![Latest Release](https://img.shields.io/github/v/release/jprincevevo/reap?style=flat-square&color=7D56F4)](https://github.com/jprincevevo/reap/releases/latest)

A high-performance, visually polished CLI tool for batch-cloning Git repositories based on a YAML config. Optimized for "clone-audit-delete" workflows.

## ✨ Features

- 🚀 **Parallel Cloning**: Uses Go Goroutines to clone multiple repositories simultaneously.
- 🎨 **Interactive TUI**: A modern terminal interface powered by Charmbracelet's Bubble Tea.
- 🛡️ **Git-Safety**: Automatically detects if you are already inside a Git worktree to prevent accidental sub-repository nesting.
- 🔄 **Self-Updating**: Keep the tool up to date with a single command (`reap update`).
- 📂 **Group Management**: Organize your repositories into custom groups for easier batching.

## 📦 Installation

### Via GitHub Releases (Recommended)
Download the pre-compiled binary for your OS from the [Releases Page](https://github.com/jprincevevo/reap/releases/latest).

### Via Go
```bash
go install [github.com/jprincevevo/reap@latest](https://github.com/jprincevevo/reap@latest)
```

## 🚀 Quick Start

1.  **Add a repository:**

    ```bash
    reap repo add https://github.com/charmbracelet/bubbletea.git
    ```

2.  **Launch the TUI:**

    ```bash
    reap
    ```

## Configuration

`reap` uses a configuration file located at `~/.config/reap/config.yaml` (on Unix-like systems) or `%AppData%/reap/config.yaml` (on Windows).

### Example `config.yaml`

```yaml
default_depth: 1          # optional; git clone --depth fallback (0 = full clone)
default_dir: ~/projects   # optional; clone destination fallback
default_pull: false       # optional; pull instead of clone when the repo already exists
repos:
  - url: https://github.com/charmbracelet/bubbletea.git
    groups:
      - name: charm
        selected: true
  - url: https://github.com/charmbracelet/lipgloss.git
    groups:
      - name: charm
        selected: true
  - url: https://github.com/spf13/cobra.git
```

## Commands Reference

| Command                    | Description                                                                 |
| -------------------------- | --------------------------------------------------------------------------- |
| `reap`                     | Launch the interactive TUI to select repositories or groups for cloning.    |
| `reap repo add <url>`      | Add a new repository to the configuration.                                  |
| `reap repo remove`         | Launch a TUI to select and remove repositories from the configuration.      |
| `reap repo list`           | Print all configured repository URLs.                                       |
| `reap group add <name>`    | Create a new group and select repositories to add to it.                    |
| `reap group remove <name>` | Remove a group from all repositories.                                       |
| `reap group list`          | List all custom groups.                                                     |
| `reap status`              | Show clone status of all configured repositories.                           |
| `reap update`              | Update `reap` to the latest version.                                        |

## Development

### Running Locally

1.  Clone the repository:

    ```bash
    git clone https://github.com/jprincevevo/reap.git
    cd reap
    ```

2.  Run the application:

    ```bash
    go run .
    ```

### Common Makefile Targets

```bash
make test        # run all tests (go test ./...)
make test-v      # verbose test output
make check       # gofmt + go vet + tests — run before pushing
make build       # produce ./reap_test binary
make clean       # remove ./reap_test
make fmt         # gofmt -w .
make tidy        # go mod tidy
```

### Testing Versioning

```bash
go build -ldflags="-X github.com/jprincevevo/reap/version.Version=v1.0.0" -o reap_test .
```

## Safety

`reap` includes a safety check that prevents you from accidentally cloning repositories inside an existing Git tree. If it detects that you are in a Git repository, it will prompt for confirmation before proceeding.
