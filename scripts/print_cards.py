#!/usr/bin/env python3
"""
print_cards.py — lay out Luhmann .txt cards on 8.5"×11" letter paper for printing.

Two 4"×6" landscape cards are printed per sheet, centered with corner tick marks
as cutting guides. Cards are sorted in natural Luhmann order (1, 1a, 1a1, 2, …).

Cards that exceed the printable area are refused with an error message; they must
be split before printing.

Cards with a "---back---" delimiter are printed with a back face. By default,
backs are printed as separate cards labeled "1a (back)". With --duplex, each
sheet carries the front on the top slot and the back on the bottom slot so the
sheet can be cut and folded to produce a two-sided card.

Usage:
    python3 scripts/print_cards.py <directory> [output.pdf] [--duplex] [--back-offset dx,dy]
    <tool> | python3 scripts/print_cards.py - [output.pdf] [--duplex] [--back-offset dx,dy]

    Pass "-" instead of a directory to read a newline-separated list of .txt
    file paths from stdin. Useful for printing only cards changed since a date:

        scripts/changed-since "2024-01-01" ~/notes | python3 scripts/print_cards.py - out.pdf

    --back-offset dx,dy   Shift the back page by dx mm (right) and dy mm (up) to
                          correct duplex registration. Negative values go left/down.
                          Example: --back-offset 1,-0.5

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

def _reflow(content: str, max_chars: int) -> list:
    """Convert card content to visual lines, re-flowing paragraphs.

    Paragraph text is re-joined and re-wrapped to max_chars so that lines
    authored for a narrow terminal card face fill the wider PDF card. Blank
    lines (paragraph separators) and link lines (starting with -->) are kept
    as-is and never merged into surrounding text.
    """
    visual = []
    paragraph = []

    def flush():
        if paragraph:
            for line in textwrap.wrap(" ".join(paragraph), width=max_chars) or [""]:
                visual.append(line)
            paragraph.clear()

    for raw in content.splitlines():
        stripped = raw.strip()
        if not stripped:
            flush()
            visual.append("")
        elif stripped.startswith("-->"):
            flush()
            visual.append(stripped)
        else:
            paragraph.append(stripped)

    flush()
    return visual

def _visual_line_count(content: str, max_chars: int) -> int:
    return len(_reflow(content, max_chars))

# ── natural sort ──────────────────────────────────────────────────────────────

def _natural_key(path: Path):
    parts = re.split(r'(\d+)', path.stem)
    return [int(p) if p.isdigit() else p.lower() for p in parts]

# ── card back splitting ───────────────────────────────────────────────────────

BACK_DELIMITER = "---back---"

def _split_sides(content: str) -> tuple[str, str | None]:
    """Return (front, back) where back is None if no ---back--- line exists."""
    lines = content.splitlines()
    for i, line in enumerate(lines):
        if line.strip() == BACK_DELIMITER:
            front = "\n".join(lines[:i])
            back  = "\n".join(lines[i + 1:])
            return front, back
    return content, None

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
    for line in _reflow(content, max_chars):
        if current_y < min_body_y:
            break
        if line:
            c.drawString(x + PAD, current_y, line)
        current_y -= LEADING


# ── main ──────────────────────────────────────────────────────────────────────

def main():
    MM = 72 / 25.4  # points per millimetre

    args = sys.argv[1:]
    duplex = "--duplex" in args
    args = [a for a in args if a != "--duplex"]

    back_dx = back_dy = 0.0
    for i, a in enumerate(args):
        if a == "--back-offset" and i + 1 < len(args):
            try:
                parts = args[i + 1].split(",")
                back_dx = float(parts[0]) * MM
                back_dy = float(parts[1]) * MM if len(parts) > 1 else 0.0
            except (ValueError, IndexError):
                print("error: --back-offset expects dx,dy in mm (e.g. 1,-0.5)", file=sys.stderr)
                sys.exit(1)
    args = [a for i, a in enumerate(args)
            if a != "--back-offset" and (i == 0 or args[i - 1] != "--back-offset")]

    if not args:
        print(__doc__)
        sys.exit(1)

    output = args[1] if len(args) > 1 else "cards.pdf"

    if args[0] == "-":
        lines = sys.stdin.read().splitlines()
        all_paths = sorted(
            (Path(l.strip()) for l in lines if l.strip()),
            key=_natural_key,
        )
        if not all_paths:
            print("no .txt files on stdin", file=sys.stderr)
            sys.exit(1)
    else:
        box_dir = Path(args[0]).expanduser().resolve()
        if not box_dir.is_dir():
            print(f"error: {box_dir} is not a directory", file=sys.stderr)
            sys.exit(1)
        all_paths = sorted(box_dir.glob("*.txt"), key=_natural_key)
        if not all_paths:
            print(f"no .txt files found in {box_dir}", file=sys.stderr)
            sys.exit(1)

    max_lines = _content_max_lines()
    max_chars = _content_max_chars()

    # Each entry: (stem_label, content_to_print)
    cards   = []
    refused = []
    for path in all_paths:
        raw     = path.read_text(encoding="utf-8", errors="replace")
        front, back = _split_sides(raw)

        lines = _visual_line_count(front, max_chars)
        if lines > max_lines:
            refused.append((path.stem, lines, "front"))
        else:
            cards.append((path.stem, front, back))

        if back is not None:
            blines = _visual_line_count(back, max_chars)
            if blines > max_lines:
                refused.append((path.stem + " (back)", blines, "back"))
            # back is always paired with its front; refusal is warned but still printed
            # if it fits, or skipped in --duplex if it was refused.

    if refused:
        print("refused (exceeds card face — split before printing):", file=sys.stderr)
        for stem, lines, side in refused:
            print(f"  {stem}  ({lines} lines, limit {max_lines})", file=sys.stderr)

    if not cards:
        print("no printable cards.", file=sys.stderr)
        sys.exit(1)

    c = canvas.Canvas(output, pagesize=letter)

    if duplex:
        # Duplex mode (long-edge binding):
        #   PDF page N   — fronts for this sheet (top + bottom slots, up to 2 cards)
        #   PDF page N+1 — backs in the SAME slots, so they land physically behind
        #                  the fronts when the printer flips on the long edge.
        # After printing, cut the sheet horizontally to get two-sided cards.
        slots = [CARD_TOP_Y, CARD_BOT_Y]
        num_sheets = (len(cards) + 1) // 2
        pdf_pages = 0
        for sheet in range(num_sheets):
            sheet_cards = cards[sheet * 2 : sheet * 2 + 2]
            # Front page
            if pdf_pages > 0:
                c.showPage()
            for i, (stem, front, back) in enumerate(sheet_cards):
                draw_card(c, CARD_X, slots[i], stem, front)
            pdf_pages += 1
            # Back page — only if at least one card in this sheet has a back
            backs = [(i, stem, back) for i, (stem, front, back) in enumerate(sheet_cards)
                     if back is not None and _visual_line_count(back, max_chars) <= max_lines]
            if backs:
                c.showPage()
                pdf_pages += 1
                for i, stem, back in backs:
                    draw_card(c, CARD_X + back_dx, slots[i] + back_dy, stem + " (back)", back)
        print(f"{len(cards)} card(s) → {num_sheets} sheet(s) → {pdf_pages} page(s) duplex (long-edge) → {output}")
    else:
        # Default mode: collect all faces (fronts + backs labeled separately).
        faces = []
        for stem, front, back in cards:
            faces.append((stem, front))
            if back is not None:
                back_lines = _visual_line_count(back, max_chars)
                if back_lines <= max_lines:
                    faces.append((stem + " (back)", back))

        pages = (len(faces) + 1) // 2
        print(f"{len(faces)} face(s) → {pages} page(s) → {output}")

        slots = [CARD_TOP_Y, CARD_BOT_Y]
        for i, (label, content) in enumerate(faces):
            slot = i % 2
            if slot == 0 and i > 0:
                c.showPage()
            draw_card(c, CARD_X, slots[slot], label, content)

    c.save()
    print(f"saved {output}")


if __name__ == "__main__":
    main()
