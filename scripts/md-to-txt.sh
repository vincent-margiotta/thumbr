#!/usr/bin/env bash
# Usage: ./md2txt.sh [directory]
# Renames *.md -> *.txt in the given directory (non-recursive).
# Skips any file where the .txt name already exists.

set -euo pipefail

dir="${1:-.}"

if [[ ! -d "$dir" ]]; then
  echo "Error: '$dir' is not a directory" >&2
  exit 1
fi

count=0
skipped=0

for src in "$dir"/*.md; do
  [[ -e "$src" ]] || { echo "No .md files found in '$dir'"; exit 0; }
  dst="${src%.md}.txt"
  if [[ -e "$dst" ]]; then
    echo "SKIP: $(basename "$dst") already exists"
    skipped=$((skipped + 1))
    continue
  fi
  mv -- "$src" "$dst"
  echo "  $(basename "$src") -> $(basename "$dst")"
  count=$((count + 1))
done

echo ""
echo "Done: $count renamed, $skipped skipped."

