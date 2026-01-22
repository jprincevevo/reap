![Latest Release](https://img.shields.io/github/v/release/jprincevevo/reap?style=flat-square&color=7D56F4)

# reap

A high-performance, visually polished CLI tool for batch-cloning Git repositories based on a YAML config. Optimized for "clone-audit-delete" workflows.

## Installation

### From GitHub Releases

Download the latest binary for your operating system from the [GitHub Releases](https://github.com/jprincevevo/reap/releases) page.

### For Go Developers

```bash
go install github.com/jprincevevo/reap@latest
```

## Quick Start

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
repos:
  - url: https://github.com/charmbracelet/bubbletea.git
    groups:
      - charm
  - url: https://github.com/charmbracelet/lipgloss.git
    groups:
      - charm
  - url: https://github.com/spf13/cobra.git
```

## Commands Reference

| Command                  | Description                                                                 |
| ------------------------ | --------------------------------------------------------------------------- |
| `reap`                   | Launch the interactive TUI to select repositories or groups for cloning.    |
| `reap repo add <url>`    | Add a new repository to the configuration.                                  |
| `reap repo remove`       | Launch a TUI to select and remove repositories from the configuration.      |
| `reap group list`        | List all custom groups.                                                     |
| `reap group add <name>`  | Create a new group and select repositories to add to it.                    |
| `reap group remove`      | Launch a TUI to select and remove a group.                                  |
| `reap update`            | Update `reap` to the latest version.                                        |

## Development

### Running Locally

1.  Clone the repository:

    ```bash
    git clone https://github.com/jprincevevo/reap.git
    cd reap
    ```

2.  Run the application:

    ```bash
    go run main.go
    ```

### Building a Local Binary

```bash
go build -o reap
```

### Testing Versioning

To test the version injection, use the following `ldflags`:

```bash
go build -ldflags="-X github.com/jprincevevo/reap/version.Version=v1.0.0" -o reap
```

## Safety

`reap` includes a safety check that prevents you from accidentally cloning repositories inside an existing Git tree. If it detects that you are in a Git repository, it will prompt for confirmation before proceeding.
