<p align="center">
  <img src="assets/gotify-mu-logo.png" alt="Gotify MU" width="520">
</p>

<h1 align="center">Gotify MU</h1>

<p align="center"><strong>Gotify-compatible push notifications with real multi-user channels.</strong></p>

Gotify MU is a multi-user fork of [Gotify Server](https://github.com/gotify/server). It keeps the Gotify protocol and client compatibility while extending the server so a notification channel can be shared with multiple users instead of belonging to only one account.

> **Project status:** active development. The multi-user foundation is available for testing, but releases should be treated as pre-production until the compatibility and migration test suite is complete.

## Why Gotify MU?

Upstream Gotify applications are owned by a single user. Gotify MU keeps that model for compatibility, but adds a membership layer so an application can function as a shared **Channel**.

### Current MU features

- Shared notification channels across multiple users
- Channel owner plus user memberships
- Admin-managed global channels
- Automatic assignment to all current and future users
- One stored message with WebSocket fan-out to every entitled user
- Per-user dismissal of shared messages
- Original physical-delete behavior retained for private channels
- Owner/admin member management in the Web UI
- Shared users can read and receive without automatically gaining publish authority
- Existing Gotify application tokens remain valid
- Existing client-token authentication remains supported
- Existing `/application`, `/message`, and `/stream` compatibility is retained

## Client compatibility

Gotify MU is intentionally designed to remain compatible with normal Gotify clients.

The official [Gotify Android app](https://github.com/gotify/android) can continue to connect using its normal client token and WebSocket stream. MU-specific fields are additive, so clients that do not understand them can ignore them.

The existing Gotify CLI and API integrations should continue to work against the compatibility routes.

## Web UI

The MU Web UI presents Gotify applications as **Channels** and adds member management and global auto-assignment controls.

The underlying `/application` API naming remains in place to avoid breaking existing clients and integrations.

## Installation

Gotify MU currently follows the upstream Gotify configuration model. Existing `GOTIFY_*` environment variables are intentionally retained for compatibility.

Until formal Gotify MU releases are published, build and test from this repository:

```bash
git clone https://github.com/gigabytegrove/gotify-mu.git
cd gotify-mu
make build-js
go build -o gotify-mu app.go
./gotify-mu serve
```

A Gotify MU container build is defined in `docker/Dockerfile`. The project container namespace is:

```text
ghcr.io/gigabytegrove/gotify-mu
```

## Upgrading an existing Gotify installation

The MU database migration is additive. Existing users, applications, tokens and messages remain in place, and existing application owners are backfilled as channel members.

**Back up your database before testing an upgrade.** This project is still in active development.

## Development

Server: Go  
Web UI: TypeScript / React  
Database layer: GORM  
Realtime delivery: existing Gotify WebSocket stream

The Go module remains `github.com/gotify/server/v3` for upstream/plugin compatibility. That is intentional and should not be changed as a branding exercise.

## Upstream relationship

Gotify MU is based on Gotify and retains substantial upstream code and compatibility. It is a separate fork maintained at:

https://github.com/gigabytegrove/gotify-mu

For upstream Gotify documentation and ecosystem information, see [gotify.net](https://gotify.net/).

Gotify MU is not presented as an official Gotify project.

## Contributing

Issues and pull requests for Gotify MU should be opened in this repository. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Gotify MU remains licensed under the MIT License inherited from Gotify. See [LICENSE](LICENSE).
