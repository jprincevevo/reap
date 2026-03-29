Output demo.gif
Set FontSize 16
Set Width 900
Set Height 500
Set Padding 30
Set Theme "Catppuccin Mocha"

# ── Setup (hidden — not recorded) ────────────────────────────────────────────
Hide

# VHS starts a bare bash session — ensure Go is findable before building.
Type "export PATH=$PATH:/usr/local/go/bin:/opt/homebrew/bin:$HOME/go/bin"
Enter
# Build a fresh binary and put the project dir first on PATH so
# the demo binary shadows any globally-installed reap.
Type "go build -o reap ."
Enter
Sleep 20s
Type "export PATH=$PWD:$PATH"
Enter

# Preserve the user's real config; write a fake one using Python to avoid
# heredoc issues in VHS's interactive TTY environment.
Type "cp ~/.config/reap/config.yaml ~/.config/reap/config.yaml.bak 2>/dev/null; true"
Enter
Type `python3 -c "import pathlib; cfg = pathlib.Path.home() / '.config/reap/config.yaml'; cfg.parent.mkdir(parents=True, exist_ok=True); cfg.write_text('default_depth: 1\nrepos:\n  - url: https://github.com/charmbracelet/bubbletea.git\n    selected: false\n    groups:\n      - name: charmbracelet\n        selected: false\n  - url: https://github.com/charmbracelet/lipgloss.git\n    selected: false\n    groups:\n      - name: charmbracelet\n        selected: false\n  - url: https://github.com/charmbracelet/bubbles.git\n    selected: false\n    groups:\n      - name: charmbracelet\n        selected: false\n  - url: https://github.com/spf13/cobra.git\n    selected: false\n    groups:\n      - name: tools\n        selected: false\n  - url: https://github.com/spf13/viper.git\n    selected: false\n    groups:\n      - name: tools\n        selected: false\n')"`
Enter
Sleep 500ms

# Nuke and recreate the sandbox so any leftover clone dirs don't cause failures.
Type "rm -rf /tmp/reap-demo && mkdir /tmp/reap-demo && cd /tmp/reap-demo"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Show

# ── Recording ─────────────────────────────────────────────────────────────────

Type "reap"
Sleep 500ms
Enter
Sleep 2s

# Group selection screen:
# Items: [Show all repos] [charmbracelet] [tools]
# Cursor starts on "Show all repos" — one Down lands on "charmbracelet".
Down
Sleep 700ms
Enter
Sleep 1s

# Repo selection screen (charmbracelet group — 3 repos, all unchecked).
# ctrl+a selects everything in one shot.
Ctrl+A
Sleep 1s
Enter

# Cloning screen runs inline (no alt screen).
# Wait until the summary header appears, then hold for readability.
Wait+Screen@60s /Cloning complete/
Sleep 3s

# ── Cleanup (hidden) ──────────────────────────────────────────────────────────
Hide
Type "cd ~"
Enter
Type "rm -rf /tmp/reap-demo"
Enter
Type "cp ~/.config/reap/config.yaml.bak ~/.config/reap/config.yaml 2>/dev/null || rm -f ~/.config/reap/config.yaml"
Enter
Type "rm -f $(which reap)"
Enter
