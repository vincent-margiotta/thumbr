#!/usr/bin/env bash
# Convert docs/thumbr.webm -> docs/thumbr.gif with section label overlays.
# Run from project root after: vhs scripts/demo.tape
# Requires: ffmpeg

set -euo pipefail

SRC="docs/thumbr.webm"
OUT="docs/thumbr.gif"
TMP="docs/thumbr_labeled.mp4"

# Section label timestamps (seconds): derived from tape Sleep sums.
# Format: "text:start:end"
LABELS=(
  "browse the stack:5:16"
  "open and read   -   enter:17:20"
  "edit in place   -   e:21:29"
  "continue a thread   -   c:30:38"
  "random jump   -   r:39:42"
  "branch   -   C:43:53"
)

# Build drawtext filter chain
FONT_COLOR="white"
BOX_COLOR="black@0.55"
FONT_SIZE=16
PAD=6
Y_POS=12

drawtext_chain=""
for label in "${LABELS[@]}"; do
  IFS=':' read -r text t_start t_end <<< "$label"
  # Escape special characters for ffmpeg
  escaped=$(printf '%s' "$text" | sed "s/'/'\\\\''/g; s/:/\\\\:/g")
  filter="drawtext=text='${escaped}':x=${PAD}:y=${Y_POS}:fontcolor=${FONT_COLOR}:fontsize=${FONT_SIZE}:box=1:boxcolor=${BOX_COLOR}:boxborderw=${PAD}:enable='between(t,${t_start},${t_end})'"
  if [[ -z "$drawtext_chain" ]]; then
    drawtext_chain="$filter"
  else
    drawtext_chain="${drawtext_chain},${filter}"
  fi
done

echo "Step 1: applying labels -> $TMP"
ffmpeg -y -i "$SRC" \
  -vf "fps=12,scale=960:trunc(ow/a/2)*2:flags=lanczos,${drawtext_chain}" \
  -c:v libx264 -crf 18 "$TMP"

echo "Step 2: converting to gif with clean palette -> $OUT"
ffmpeg -y -i "$TMP" \
  -filter_complex "split[a][b];[a]palettegen=stats_mode=full:max_colors=256[p];[b][p]paletteuse=dither=none" \
  "$OUT"

rm -f "$TMP"
echo "Done: $OUT ($(du -sh "$OUT" | cut -f1))"
