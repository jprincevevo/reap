# Output settings
Output demo.gif
Set FontSize 18
Set Width 1200
Set Height 800
Set Padding 40
Set Theme "Catppuccin Mocha"

# Setup: Build the tool and add the current dir to PATH
Hide
Type "go build -o reap main.go"
Enter
Type "export PATH=$PWD:$PATH"
Enter
Type "mkdir -p vhs-sandbox && cd vhs-sandbox"
Enter
# IMPORTANT: Give the shell a moment to 'settle' before the next command
Sleep 500ms
Type "reap repo add https://github.com/charmbracelet/bubbletea"
Enter
Type "clear"
Enter
Sleep 1s
Show

# Step 1: Launch the tool
Type "reap"
Sleep 500ms
Enter
Sleep 2s

# Step 2: Navigate the TUI
# VHS can simulate arrow keys
Down
Sleep 500ms
Down
Sleep 500ms
Enter

# Step 3: Select and Clone
# (Assuming your TUI uses space to select and enter to clone)
Space
Sleep 1s
Enter

# Wait for the parallel cloning animation to finish
Sleep 5s

# Step 4: Show the result
Type "ls -l"
Enter
Sleep 3s

Hide
Type "cd .."
Enter
Type "rm -rf vhs-sandbox reap"
Enter
