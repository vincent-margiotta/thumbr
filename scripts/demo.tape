# Thumbr demo — run from project root with:
#   PATH="/tmp:$PATH" /home/vince/go/bin/vhs scripts/demo.tape
# Requires: vhs, ttyd >= 1.7.2, ffmpeg

Output docs/thumbr.webm

Set Shell "bash"
Set FontSize 14
Set Width 1100
Set Height 680
Set Theme "Dracula"
Set PlaybackSpeed 0.8

# ── Setup ────────────────────────────────────────────────────────────────────
# Copy sample notes to a temp dir so new cards don't modify the repo.
Type "cp -r /home/vince/Documents/src/thumbr/samples/notes /tmp/thumbr_demo"
Enter
Sleep 0.5s

# ── Launch ───────────────────────────────────────────────────────────────────
Type "/home/vince/Documents/src/thumbr/bin/thumbr /tmp/thumbr_demo"
Enter
Sleep 2.5s

# ── Browse: j/k one card at a time ──────────────────────────────────────────
Type "k"
Sleep 0.5s
Type "k"
Sleep 0.5s
Type "k"
Sleep 0.5s
Type "j"
Sleep 0.5s
Type "j"
Sleep 0.8s

# ── Jump to end, bisect back ─────────────────────────────────────────────────
Type "G"
Sleep 0.6s
Type "["
Sleep 0.5s
Type "["
Sleep 0.8s

# ── Random jump ─────────────────────────────────────────────────────────────
Type "r"
Sleep 0.8s
Type "r"
Sleep 1s

# ── Navigate to 1b (has a landscape back): position 4/10, k k k from start ──
Type "g"
Sleep 0.3s
Type "g"
Sleep 0.5s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.4s
Type "k"
Sleep 0.8s

# ── Open overlay and read ────────────────────────────────────────────────────
Enter
Sleep 1.2s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.4s
Type "j"
Sleep 0.8s

# ── Flip to the landscape back ───────────────────────────────────────────────
Type "f"
Sleep 2s
Escape
Sleep 0.8s

# ── Navigate to card 4 (portrait back): G then j ────────────────────────────
Type "G"
Sleep 0.5s
Type "j"
Sleep 0.7s

# ── Overlay + flip to portrait back ─────────────────────────────────────────
Enter
Sleep 1s
Type "f"
Sleep 2s
Escape
Sleep 0.8s

# ── Mark a card ─────────────────────────────────────────────────────────────
Type "m"
Sleep 1.2s

# ── Open editor (stays on card 4) ───────────────────────────────────────────
Type "e"
Sleep 1.5s

# ── Write a continuation note ───────────────────────────────────────────────
Type "G"
Sleep 0.4s
Type "o"
Sleep 0.4s
Type "See also: 4a"
Sleep 0.8s
Escape
Sleep 0.6s

# ── Save and close ───────────────────────────────────────────────────────────
Type ":wq"
Enter
Sleep 1s

# ── Create a linked continuation card (c) ───────────────────────────────────
Type "c"
Sleep 2s

# ── Write in the new card (bottom pane, auto-focused) ───────────────────────
Type "i"
Sleep 0.4s
Type "The filing instinct is the enemy."
Sleep 1s
Escape
Sleep 0.6s

# ── Save and close both panes ───────────────────────────────────────────────
Type ":wqa"
Enter
Sleep 1.2s

# ── Help overlay ────────────────────────────────────────────────────────────
Type "?"
Sleep 3s
Escape
Sleep 0.8s

# ── Quit ────────────────────────────────────────────────────────────────────
Type "q"
Sleep 0.5s
Type "q"
Sleep 1s

# ── Cleanup ──────────────────────────────────────────────────────────────────
Type "rm -rf /tmp/thumbr_demo"
Enter
Sleep 0.3s
