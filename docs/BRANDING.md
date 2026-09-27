# Monita brand transition

Monita is the new product identity for Gotify MU.

## User-facing identity

- Product: **Monita**
- Transition reference: **Monita (formerly Gotify MU)**
- Brand essence: **Notifications · Messaging · Automation**
- Brand statement: **Monita is a self-hosted notifications, messaging, and automation communication platform for teams and systems.**

## Color system

| Role | Color |
| --- | --- |
| Primary | `#2563EB` |
| Blue | `#3B82F6` |
| Teal | `#06B6D4` |
| Slate | `#0F172A` |
| Gray | `#94A3B8` |

The approved Monita mark, app icon, horizontal logo, palette, and brand essence are derived from the supplied canonical brand sheet.

## Compatibility policy

The rebrand must not break existing Gotify-compatible clients or existing Monita/Gotify MU installations.

The following are compatibility identifiers and are **not** renamed merely for visual branding:

- Gotify REST compatibility routes such as `/application`, `/message`, and `/stream`
- the Go module path `github.com/gotify/server/v3`
- existing upstream-compatible `GOTIFY_*` configuration variables
- persisted database structures and existing tokens
- existing repository/container identifiers until a separately planned migration provides compatibility aliases

Existing Gotify-MU-specific `GOTIFY_MU_*` configuration may gain `MONITA_*` aliases in a later migration, but existing deployments must continue to work without mandatory configuration rewrites.

## Transition rules

1. New user-facing product surfaces say **Monita**.
2. During the transition, documentation and release metadata may use **Monita (formerly Gotify MU)**.
3. Historical release notes retain their original Gotify MU terminology.
4. New release branding uses Monita assets and the Monita palette.
5. Compatibility identifiers are changed only with an explicit migration and backward-compatible aliases.
