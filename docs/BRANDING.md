# Monita Branding

**Monita** is the current product identity.

The canonical artwork was supplied on **2026-10-01** and is preserved in `assets/source/`.

## Canonical artwork

- `Monita_Full-01.svg` — full Monita logo with the **Notifications · Messaging · Automation** tagline
- `Monita_IconOnly-01.svg` — standalone Monita/application icon

The supplied SVGs are the source of truth. They must not be redrawn, recolored, flattened into substitute artwork, cropped, stretched, re-proportioned, or recreated from individual shapes.

Active files such as `assets/monita-logo.svg`, `assets/monita-banner.svg`, `assets/monita-icon.svg`, and the Web/PWA copies are direct uses of those supplied vectors.

Where a legacy or platform path requires PNG, `scripts/render_branding.py` produces a faithful raster render at the source artwork's native dimensions. The raster files are compatibility derivatives, not separate designs.

## Naming

Use **Monita** for current product-facing text.

Historical Gotify/Gotify MU wording may remain only where it documents project lineage or a compatibility identifier/protocol that cannot be renamed without breaking existing clients.

## CI integrity

The build verifies that the active SVG aliases resolve to the locked canonical Git blobs:

- full logo source: `fc0dd563e198031de3844dd5b9d94a100dba025e`
- icon source: `20ac209f831efc4a983d7e6c762a9163e7074aad`
