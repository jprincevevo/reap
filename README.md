# reap

`reap` is a command-line tool for batch cloning Git repositories. It's designed for developers who frequently work with multiple repositories and need a quick way to clone them for tasks like testing, analysis, or running scripts.

## Why use reap?

- **Organize your repositories**: Group your most frequently used repositories for quick access.
- **Interactive TUI**: A terminal user interface for selecting which repositories to clone.
- **Quickly clone repos**: Bypass the configuration and clone repositories directly by providing their URLs.

## Installation

Installation instructions will be added here once the build process is set up.

## Usage

`reap` uses a configuration file at `~/.reap.yaml` to manage your repositories.

### Commands

- `reap`: The main interactive flow. If you have groups configured, you will be prompted to select a group, otherwise, you will be prompted to select the repositories to clone.
- `reap <url1> <url2>`: Direct clone of specific URLs, bypassing the configuration file.
- `reap add <url>`: Add a new repository to the configuration file.
- `reap remove`: Interactive TUI list to delete a repository from the configuration file.
- `reap groups`: List all custom groups.
- `reap groups add <name>`: Prompt with a multi-select list of ALL repositories. The user selects which repos should belong to this new group.
- `reap groups remove <name>`: Delete a group definition from all repos.
- `reap update`: Update reap to the latest version.
