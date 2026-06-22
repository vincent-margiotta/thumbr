# Thumbr demo — run from project root:
#   PATH="/tmp:$PATH" /home/vince/go/bin/vhs scripts/demo.tape
#   then: bash scripts/demo_gif.sh
# Requires: vhs, ttyd >= 1.7.2, ffmpeg

Output docs/thumbr.webm

Set Shell "bash"
Set FontSize 14
Set Width 1100
Set Height 680
Set Theme "Dracula"
Set PlaybackSpeed 0.8

# Setup: copy sample notes so new cards don't touch the repo
Type "cp -r /home/vince/Documents/src/thumbr/samples/notes /tmp/thumbr_demo"
Enter
Sleep 0.5s

# t=0 (launch): thumbr starts
Type "/home/vince/Documents/src/thumbr/bin/thumbr /tmp/thumbr_demo"
Enter
Sleep 2.5s

# t≈3s — SECTION: browse the stack (k/j, one card at a time)
Type "k"
Sleep 0.5s
Type "k"
Sleep 0.5s
Type "k"
Sleep 0.5s
Type "k"
Sleep 0.7s
Type "j"
Sleep 0.5s
Type "j"
Sleep 0.5s
Type "j"
Sleep 0.9s

# t≈8s — SECTION: open and read (cursor on 1b which has content)
# navigate back to 1b: gg then k k k
Type "g"
Sleep 0.3s
Type "g"
Sleep 0.4s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.6s
Enter
Sleep 1.5s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.6s
Escape
Sleep 0.8s

# t≈15s — SECTION: edit in place
Type "e"
Sleep 1.5s
Type "G"
Sleep 0.3s
Type "o"
Sleep 0.4s
Type "Cards that compress cleanly are notes."
Sleep 0.8s
Escape
Sleep 0.5s
Type ":wq"
Enter
Sleep 1.2s

# t≈21s — SECTION: continue a thread (c → split pane)
Type "c"
Sleep 2.5s
Type "i"
Sleep 0.4s
Type "A new card continues the thread."
Sleep 1.0s
Escape
Sleep 0.5s
Type ":wqa"
Enter
Sleep 1.5s

# t≈28s — SECTION: random jump
Type "r"
Sleep 1.5s

# t≈30s — SECTION: branch (C → split pane sibling)
Type "C"
Sleep 2.5s
Type "i"
Sleep 0.4s
Type "A sibling card branches here."
Sleep 1.0s
Escape
Sleep 0.5s
Type ":wqa"
Enter
Sleep 1.2s

# Quit
Type "q"
Sleep 0.5s
Type "q"
Sleep 1.0s

# Cleanup
Type "rm -rf /tmp/thumbr_demo"
Enter
Sleep 0.3s
