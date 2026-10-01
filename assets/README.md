# Monita Brand Assets

The artwork in this directory is the approved Monita identity supplied on 2026-10-01.

## Canonical source files

The source artwork is preserved under `assets/source/`:

- `Monita_Full-01.svg` — supplied full Monita logo, including the **Notifications · Messaging · Automation** tagline
- `Monita_IconOnly-01.svg` — supplied Monita standalone/application icon

These source SVGs are authoritative. Do not redraw, simplify, recolor, crop, distort, or recreate them.

## Active aliases

- `monita-logo.svg` — exact vector artwork from `Monita_Full-01.svg`
- `monita-banner.svg` — exact vector artwork from `Monita_Full-01.svg`
- `monita-icon.svg` — exact vector artwork from `Monita_IconOnly-01.svg`

The Web/PWA copies under `ui/public/static/` use the same SVG artwork.

Legacy PNG filenames are retained only so older/static paths do not break. They are rasterized directly from the supplied vectors by `scripts/render_branding.py` at the vectors' native dimensions; they are not alternate artwork.

## Preservation rule

When a destination supports SVG, use the SVG directly. When a platform requires raster artwork, render from these SVGs without changing composition, geometry, colors, proportions, or background.
