<p align="center">
  <img src="assets/monita-banner.svg" alt="Monita" width="800">
</p>

<h1 align="center">Monita</h1>

<p align="center"><strong>Notifications · Messaging · Automation</strong></p>

Monita is a self-hosted notification and messaging platform for people, teams, home automation, and connected systems. It expands on Gotify compatibility with shared Channels, chat, automation, integrations, and multi-user administration.

> Formerly known as Gotify MU. Gotify-compatible protocol names remain where required for upgrade and client compatibility.

## Current release

**Monita 1.3.4** is the current server release documented by this repository.

Companion projects:

- [Monita for Android](https://github.com/gigabytegrove/monita-android) — native Android client; current testing release: **0.3.16**
- [Monita for Home Assistant](https://github.com/gigabytegrove/monita-ha) — HACS-compatible Home Assistant integration; current release: **1.8.3**

Release-specific changes are tracked in [CHANGELOG.md](CHANGELOG.md). Installation and update instructions are in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## What Monita does

Monita gives you one place to receive, organize, share, and automate notifications.

You can use it to:

- create private, shared, and Global Channels
- deliver notifications to multiple users
- use Chat Channels for two-way conversation
- connect Home Assistant, MQTT, Webhooks, email, RSS/Atom, Syslog, and calendars
- schedule notifications and escalation workflows
- use Quiet Hours, Digests, acknowledgements, mentions, replies, reactions, and assignments
- manage users, Groups, permissions, integrations, and security settings from the Web UI
- continue using compatible Gotify clients and integrations

## Why Monita

Traditional push-notification servers are often designed around one user or one application owner. Monita adds shared access and collaboration so the same Channel can be useful to a household, team, support group, or automation environment.

Monita is designed to remain familiar to Gotify users while adding the features needed for larger and more collaborative installations.

## Getting started

### Docker Compose

Docker Compose is the recommended installation method. **Normal installs use a prebuilt Monita image; nothing is compiled on your server.**

```bash
cd /opt
git clone https://github.com/gigabytegrove/monita.git
cd monita
cp .env.example .env
nano .env
docker compose pull
docker compose up -d
```

At minimum, set a secure administrator password in `.env`:

```env
GOTIFY_DEFAULTUSER_NAME=admin
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
```

By default, the Web UI is available on:

```text
http://SERVER-IP:8080
```

Persistent application data is stored in the configured Monita data directory and remains in place when the container is updated. Normal installs use the prebuilt Monita image, so users do not need to compile the application locally.

## Prebuilt release image archives

If your environment cannot pull directly from GitHub Container Registry, Monita also publishes prebuilt Docker image archives with each supported release.

Download the archive that matches your server architecture, verify it with the published checksum file, load it with Docker, and start Monita with Docker Compose. This path does not compile Monita locally.

Supported release archives:

- `linux-amd64` for standard 64-bit Intel/AMD systems
- `linux-arm64` for 64-bit ARM systems

## Updating

Monita can check for published releases from **Settings → Software Update**. Starting with 1.1.2, updates are handled entirely inside the single Monita container: Monita downloads and verifies the published runtime, restarts itself in place, and keeps the same persistent data. No updater container or local compilation is used.

For a manual update:

```bash
cd /opt/monita
git pull --ff-only origin master
docker compose pull
docker compose up -d
```

Back up your persistent data before major upgrades.

See [Deployment](docs/DEPLOYMENT.md) for installation, update, backup, and troubleshooting guidance.

## Main features

### Channels and collaboration

- shared Channels for multiple users
- Global Channels managed by administrators
- Chat Channels
- user and Group assignments
- per-user notification preferences
- archive and restore
- replies and threads
- reactions
- mentions
- acknowledgements
- per-message assignment, resolve/reopen, and attachment controls
- image/file attachments on supported messages
- `@username` Chat mentions with autocomplete and visible mention highlighting
- message templates
- saved searches

### Automation

- scheduled notifications
- cron scheduling
- Quiet Hours
- Digests
- escalation workflows
- acknowledgement-aware escalation cancellation

### Integrations

- Home Assistant
- Webhooks
- MQTT
- email delivery and inbound email
- RSS / Atom
- Syslog
- Calendar / iCal

### Administration and security

- local accounts
- OIDC
- LDAP / Active Directory
- TOTP MFA
- passkeys
- recovery codes
- service accounts
- role-based Channel permissions
- active-session management
- audit logging
- backup and restore tools

## Home Assistant

Monita supports both direct Home Assistant connections and native pairing with **Monita for Home Assistant**.

See [Home Assistant integration](docs/HOME_ASSISTANT_NATIVE_PAIRING.md) for setup and usage.

## Gotify compatibility

Monita is built from the Gotify Server codebase and keeps compatibility with common Gotify clients and integrations wherever practical.

Existing Gotify-style application and client tokens continue to work with supported compatibility routes. Monita-specific features are added on top of that compatibility layer.

Monita is an independent project and is not an official Gotify product.

## Documentation

- [Deployment](docs/DEPLOYMENT.md)
- [Home Assistant](docs/HOME_ASSISTANT_NATIVE_PAIRING.md)
- [Security](SECURITY.md)
- [Security overview](docs/SECURITY_ROADMAP.md)
- [Roadmap](docs/ROADMAP.md)
- [Branding](docs/BRANDING.md)
- [Contributing](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)

## Project status

Monita is under active development. The latest stable release and release notes are available from the GitHub Releases page.

## License

Monita is licensed under the MIT License inherited from Gotify. See [LICENSE](LICENSE).
