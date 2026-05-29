#!/usr/bin/env python3
"""
print_cards.py — lay out Luhmann .txt cards on 8.5"×11" letter paper for printing.

Two 4"×6" landscape cards are printed per sheet, centered with corner tick marks
as cutting guides. Cards are sorted in natural Luhmann order (1, 1a, 1a1, 2, …).

Cards that exceed the printable area are refused with an error message; they must
be split before printing.

Usage:
    python3 scripts/print_cards.py <directory> [output.pdf]

Requires:
    pip install reportlab
"""

import re
import sys
import textwrap
from pathlib import Path

try:
    from reportlab.lib import colors
    from reportlab.lib.pagesizes import letter
    from reportlab.lib.units import inch
    from reportlab.pdfgen import canvas
except ImportError:
    print("reportlab not found.  Install with:  pip install reportlab", file=sys.stderr)
    sys.exit(1)

# ── dimensions ────────────────────────────────────────────────────────────────

CARD_W = 6.0 * inch   # 4"×6" card in landscape orientation
CARD_H = 4.0 * inch

PAGE_W, PAGE_H = letter  # 8.5"×11" = 612pt × 792pt

# Two cards per page, vertically centered with equal margins.
_gap       = 0.5 * inch
_vertical  = PAGE_H - 2 * CARD_H - _gap
CARD_TOP_Y = PAGE_H - (_vertical / 2) - CARD_H
CARD_BOT_Y = CARD_TOP_Y - _gap - CARD_H
CARD_X     = (PAGE_W - CARD_W) / 2

PAD        = 0.2 * inch

# ── typography ────────────────────────────────────────────────────────────────

HEADER_FONT = "Courier-Bold"
BODY_FONT   = "Courier"
HEADER_PT   = 11
BODY_PT     = 10
LEADING     = BODY_PT * 1.4

# Courier is a fixed-pitch font; each character is ~0.6× the point size wide.
_CHAR_W_RATIO = 0.6

# ── size limits (derived from the typography constants above) ─────────────────

def _content_max_chars() -> int:
    content_w = CARD_W - 2 * PAD
    return max(1, int(content_w / (BODY_PT * _CHAR_W_RATIO)))

def _content_max_lines() -> int:
    header_baseline = CARD_H - PAD - HEADER_PT
    rule_y          = header_baseline - HEADER_PT * 0.35
    first_line_y    = rule_y - LEADING
    available       = first_line_y - PAD
    return max(1, int(available / LEADING))

def _visual_line_count(content: str, max_chars: int) -> int:
    """Count the visual lines a card will occupy, accounting for wrapping."""
    count = 0
    for raw in content.splitlines():
        if not raw.strip():
            count += 1
        else:
            count += max(1, len(textwrap.wrap(raw, width=max_chars)))
    return count

# ── natural sort ──────────────────────────────────────────────────────────────

def _natural_key(path: Path):
    parts = re.split(r'(\d+)', path.stem)
    return [int(p) if p.isdigit() else p.lower() for p in parts]

# ── drawing ───────────────────────────────────────────────────────────────────

def _tick(c: canvas.Canvas, cx: float, cy: float, size: float = 0.08 * inch):
    c.line(cx - size, cy, cx + size, cy)
    c.line(cx, cy - size, cx, cy + size)


def draw_card(c: canvas.Canvas, x: float, y: float, stem: str, content: str):
    # Border
    c.setStrokeColor(colors.black)
    c.setLineWidth(0.5)
    c.rect(x, y, CARD_W, CARD_H)

    # Corner cutting guides
    c.setLineWidth(0.3)
    for cx, cy in [(x, y), (x + CARD_W, y), (x, y + CARD_H), (x + CARD_W, y + CARD_H)]:
        _tick(c, cx, cy)

    # Header
    header_baseline = y + CARD_H - PAD - HEADER_PT
    c.setFont(HEADER_FONT, HEADER_PT)
    c.drawString(x + PAD, header_baseline, stem)

    # Rule below header
    rule_y = header_baseline - HEADER_PT * 0.35
    c.setLineWidth(0.3)
    c.line(x + PAD, rule_y, x + CARD_W - PAD, rule_y)

    # Body text
    max_chars  = _content_max_chars()
    min_body_y = y + PAD
    current_y  = rule_y - LEADING

    c.setFont(BODY_FONT, BODY_PT)
    for raw_line in content.splitlines():
        if not raw_line.strip():
            current_y -= LEADING
            if current_y < min_body_y:
                break
            continue
        for wrapped in textwrap.wrap(raw_line, width=max_chars) or [""]:
            if current_y < min_body_y:
                break
            c.drawString(x + PAD, current_y, wrapped)
            current_y -= LEADING
        if current_y < min_body_y:
            break


# ── main ──────────────────────────────────────────────────────────────────────

def main():
    if len(sys.argv) < 2:
        print(__doc__)
        sys.exit(1)

    box_dir = Path(sys.argv[1]).expanduser().resolve()
    if not box_dir.is_dir():
        print(f"error: {box_dir} is not a directory", file=sys.stderr)
        sys.exit(1)

    output = sys.argv[2] if len(sys.argv) > 2 else "cards.pdf"

    all_cards = sorted(box_dir.glob("*.txt"), key=_natural_key)
    if not all_cards:
        print(f"no .txt files found in {box_dir}", file=sys.stderr)
        sys.exit(1)

    max_lines = _content_max_lines()
    max_chars = _content_max_chars()

    cards   = []
    refused = []
    for path in all_cards:
        content = path.read_text(encoding="utf-8", errors="replace")
        lines   = _visual_line_count(content, max_chars)
        if lines > max_lines:
            refused.append((path.stem, lines))
        else:
            cards.append((path, content))

    if refused:
        print("refused (exceeds card face — split before printing):", file=sys.stderr)
        for stem, lines in refused:
            print(f"  {stem}  ({lines} lines, limit {max_lines})", file=sys.stderr)

    if not cards:
        print("no printable cards.", file=sys.stderr)
        sys.exit(1)

    pages = (len(cards) + 1) // 2
    print(f"{len(cards)} card(s) → {pages} page(s) → {output}")

    c = canvas.Canvas(output, pagesize=letter)
    slots = [CARD_TOP_Y, CARD_BOT_Y]

    for i, (path, content) in enumerate(cards):
        slot = i % 2
        if slot == 0 and i > 0:
            c.showPage()
        draw_card(c, CARD_X, slots[slot], path.stem, content)

    c.save()
    print(f"saved {output}")


if __name__ == "__main__":
    main()
