# Thumbr demo — run with: vhs scripts/demo.tape
# Requires: vhs (go install github.com/charmbracelet/vhs@latest) + ffmpeg

Output docs/thumbr.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 0.75

# Launch thumbr on the sample notes
Type "./thumbr samples/notes"
Enter
Sleep 1.5s

# Browse the stack — move forward through a few cards
Sleep 0.4s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.6s

# Jump back
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.6s

# Random jump
Type "r"
Sleep 0.8s

# Open the overlay
Enter
Sleep 1s

# Scroll down
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.3s
Type "j"
Sleep 0.6s

# Flip to the back (card 1b has a landscape back; card 4 has a portrait back)
Type "f"
Sleep 1s

# Close overlay
Escape
Sleep 0.5s

# Navigate to card 4 (portrait back demo) — jump to last, then bisect back
Type "G"
Sleep 0.4s
Type "["
Sleep 0.4s
Type "["
Sleep 0.4s
Type "["
Sleep 0.6s

# Open overlay, flip to portrait back
Enter
Sleep 0.8s
Type "f"
Sleep 1.2s

# Close
Escape
Sleep 0.5s

# Open a card in the editor
Type "e"
Sleep 1s

# Insert a line
Type "o"
Sleep 0.4s
Type "This is the in-app editor."
Sleep 0.6s
Escape
Sleep 0.5s

# Save and quit
Type ":q!"
Enter
Sleep 0.8s

# Show help
Type "?"
Sleep 1.5s
Escape
Sleep 0.5s

# Quit
Type "q"
Sleep 0.4s
Type "q"
Sleep 0.8s
