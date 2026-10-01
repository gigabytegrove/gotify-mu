#!/usr/bin/env python3
"""Render compatibility PNGs from the canonical Monita SVG artwork.

The SVG files under assets/source are the immutable source artwork supplied by the
project owner. This script only rasterizes those files at their native viewBox
dimensions. It does not crop, recolor, reshape, simplify, or redraw them.
"""

from pathlib import Path
import cairosvg

ROOT = Path(__file__).resolve().parents[1]
FULL = ROOT / "assets/source/Monita_Full-01.svg"
ICON = ROOT / "assets/source/Monita_IconOnly-01.svg"

def render(source: Path, destination: Path, width: int, height: int) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    cairosvg.svg2png(
        url=str(source),
        write_to=str(destination),
        output_width=width,
        output_height=height,
    )

for target in (
    ROOT / "assets/gotify-mu-banner.png",
    ROOT / "assets/gotify-mu-logo.png",
    ROOT / "ui/public/static/gotify-mu-logo.png",
):
    render(FULL, target, 1413, 512)

for target in (
    ROOT / "assets/gotify-mu-icon.png",
    ROOT / "ui/public/static/gotify-mu-icon.png",
):
    render(ICON, target, 512, 512)
