<p align="center">
  <img src="assets/monita-banner.svg" alt="Monita" width="800">
</p>

<h1 align="center">Monita</h1>

<p align="center"><strong>Notifications · Messaging · Automation</strong></p>

Monita (formerly Monita) is a self-hosted notifications, messaging, and automation platform built from the Gotify Server codebase. It preserves Gotify protocol and client compatibility while adding shared multi-user Channels, collaboration, integrations, automation, security controls, and operations tooling.

> **Current release:** **v1.1.0**.

## Branding

The canonical Monita brand system lives in [`assets/`](assets/README.md). During the transition, user-facing surfaces use **Monita (formerly Monita)** where historical context is useful. Technical Gotify compatibility identifiers remain unchanged when renaming them would break existing clients or deployments.

Approved brand palette: `#2563EB` Primary, `#3B82F6` Blue, `#06B6D4` Teal, `#0F172A` Slate, and `#94A3B8` Gray.

## Why Monita?

Upstream Gotify applications are owned by a single user. Monita keeps that model for compatibility, but adds a membership layer so an application can function as a shared **Channel**.

### Current MU features

- Shared notification channels across multiple users
- Channel owner plus user memberships
- Admin-managed global channels
- Automatic assignment to all current and future users
- One stored message with WebSocket fan-out to every entitled user
- Per-user dismissal of shared messages
- Reversible per-user message archive and restore
- Per-user mute/unmute of realtime delivery without losing channel history
- Admin-controlled Chat Channels where assigned members can post
- Sender identity on member-posted Chat Channel messages
- Channel ownership transfer between users
- Safe user deletion guard when shared channels still need an owner
- Owner/admin action to permanently clear a channel's history for everyone
- Original physical-delete behavior retained for private channels
- Owner/admin member management in the Web UI
- Shared users can read and receive without automatically gaining publish authority
- Existing Gotify application tokens remain valid
- Existing client-token authentication remains supported
- Existing `/application`, `/message`, and `/stream` compatibility is retained

## Client compatibility

Monita is intentionally designed to remain compatible with normal Gotify clients.

The official [Gotify Android app](https://github.com/gotify/android) can continue to connect using its normal client token and WebSocket stream. MU-specific fields are additive, so clients that do not understand them can ignore them.

The existing Gotify CLI and API integrations should continue to work against the compatibility routes.

## Web UI

Monita includes a redesigned Web UI built around a consistent multi-user administration model instead of the inherited upstream Gotify screen layout.

The Web UI uses **Channels** as the user-facing term and provides:

- Dashboard overview
- Unified Messages and per-user Archive views
- Channel status cards and membership management
- Global Channel administration
- User, Client, and Plugin administration
- User display names and administrative Groups
- Administrative/security Audit Log
- Elevated administrator plugin installation directly from the Web UI
- Standardized light, dark, and system themes
- Responsive desktop and mobile-web navigation
- Consistent dialogs, tables, cards, status indicators, and destructive-action language
- Administrator update discovery and one-click managed installation from Settings
- Native Integrations administration for Webhooks, MQTT, and Home Assistant
- Native Automation administration for Scheduled Notifications and Escalations
- Per-user Quiet Hours and Digest preferences
- Per-user message acknowledgement in Message History
- TOTP MFA, recovery codes, and WebAuthn/passkeys
- LDAP / Active Directory authentication
- Scoped service accounts/API credentials
- Owner / Manager / Publisher / Member / Read Only Channel roles
- Group-to-Channel assignment
- Replies/threads, reactions, mentions, assignment, resolve/reopen, attachments, templates, and saved searches
- Cron/custom scheduling, schedule history, deferred Quiet Hours, stored Digests, and richer Escalations
- Hardened Webhooks with signing, replay defense, CIDR rules, rate limiting, templates, conditions, and history
- MQTT 5 / 3.1.1 with QoS 0/1/2, custom CA and mutual TLS
- Home Assistant event/entity/data filtering with connection diagnostics
- First-party Email Delivery, SMTP Receiver, RSS/Atom, Syslog, and Calendar/iCal connectors
- Signed/checksummed Plugin Catalog installs, updates, and uninstall
- Operations, backup/restore, diagnostics, active-session administration, audit export/retention, and hardened updates

The underlying `/application` API naming remains in place to avoid breaking existing clients and integrations.

The official Gotify Android app is not modified by the Web UI rewrite.

Administrators can install compatible Linux Go plugin binaries from **Plugins → Install Plugin**. Uploaded plugins are stored under the configured `GOTIFY_PLUGINSDIR` (the default Docker data volume resolves to `/app/data/plugins`) and are loaded immediately. Plugin binaries execute native code inside the Monita process, so only trusted plugins built for the matching Monita/Go ABI and server architecture should be installed.

Authentication and security controls including MFA, passkeys, LDAP/Active Directory, session policy, service accounts, encrypted stored secrets, and audit/security administration are implemented in the v0.5 preview. See [docs/SECURITY_ROADMAP.md](docs/SECURITY_ROADMAP.md) for the current security status and trust boundaries.

### Monita channel-management API

The compatibility API remains unchanged, and MU adds these management endpoints:

```text
GET    /application/:id/members
GET    /application/:id/assignable-users
POST   /application/:id/members
DELETE /application/:id/members/:userId
PUT    /application/:id/auto-assign
PUT    /application/:id/notifications
PUT    /application/:id/owner
PUT    /application/:id/member-posting
POST   /application/:id/message/archive
DELETE /application/:id/message/archive
POST   /message/:id/archive
DELETE /message/:id/archive
POST   /message/archive
DELETE /message/archive
DELETE /application/:id/message/all
```

`PUT /application/:id/notifications` changes only the current user's realtime delivery preference. The user remains a channel member and can still read history.

`PUT /application/:id/owner` transfers canonical ownership while preserving channel membership and the existing application token.

`DELETE /application/:id/message/all` is an owner/admin action that physically removes that channel's messages for all members. Normal Gotify-compatible delete actions on shared channels remain per-user dismissals.

A user who still owns a shared channel cannot be deleted until ownership of that channel has been transferred.

### Global Channel deletion and archive behavior

For a **Global** Channel, only an administrator may physically delete individual messages, clear the Channel's history, or delete the Channel itself.

Non-admin users can archive messages instead. Archive is per-user and reversible, so archiving a message does not remove it for anyone else.

### Chat Channels

An administrator can enable **Allow channel members to post (Chat Channel)** on a Channel. When enabled, any assigned member may post using normal user/client authentication.

Monita records the sender's user ID and username. If a member leaves the title blank, the sender's username is used as the title so existing Gotify clients can still show who sent the message.

The official Gotify Android app continues to receive Chat Channel messages as normal Gotify messages. The stock Android app does not gain a compose/chat interface from this server feature; sending is available in the Monita Web UI or through compatible API clients.

## Releases

The current published release is **Monita v1.1.0**. Release history is tracked in [CHANGELOG.md](CHANGELOG.md), with detailed notes in [docs/releases/v1.1.0.md](docs/releases/v1.1.0.md).

Deployment, validation, updater, backup, and rollback procedures are maintained in [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

For a release checkout:

```bash
git clone https://github.com/gigabytegrove/gotify-mu.git monita
cd monita
git checkout v1.1.0
```

Release builds inject the release version, commit, and build date into the server binary. Development builds continue to use `master-<commit>`, `master-local`, or `dev-<commit>` identities as appropriate.

## Deployment

Monita currently follows the upstream Gotify configuration model. Existing `GOTIFY_*` environment variables are intentionally retained for compatibility.

> **Release automation:** the repository workflow publishes versioned release ZIPs and GHCR images from the version in `VERSION`. If GitHub Actions is disabled for the repository, source/Docker builds remain available as a fallback.

### Recommended: Docker Compose

On a Linux system with Git and Docker Compose installed:

```bash
cd /opt
git clone https://github.com/gigabytegrove/gotify-mu.git monita
cd monita

cp .env.example .env
nano .env
```

At minimum, change:

```text
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
```

The included `docker-compose.yml` starts exactly one persistent service: `monita`.

Example `.env`:

```env
GOTIFY_MU_VERSION=master-local
GOTIFY_MU_COMMIT=local
GOTIFY_MU_PORT=8080
GOTIFY_DEFAULTUSER_NAME=admin
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
```

Then build and start Monita:

```bash
docker compose up -d --build
```

The default deployment publishes the Web UI/API on port `8080`. Native SMTP and Syslog receivers are mapped to ports `2525/tcp` and `5514/udp` but bind to `127.0.0.1` by default. Set `GOTIFY_MU_RECEIVER_BIND` to a trusted LAN/host address only when remote devices must reach those listeners.

Open:

```text
http://SERVER-IP:8080
```

Default username:

```text
admin
```

The password is whatever you set in `.env`.

Local Compose builds display `@master-local` in the Web UI instead of `@unknown`. Native source builds identify themselves as `dev` or `dev-<commit>` when Go can read VCS metadata. GitHub-built master images use `master-<commit>`.

Persistent data is stored in:

```text
./data
```

That directory contains the SQLite database and other persistent Gotify data. Rebuilding or replacing the container does not remove it.

### Verify the deployment

Watch startup logs:

```bash
docker logs -f monita
```

Check the health endpoint:

```bash
curl http://127.0.0.1:8080/health
```

Inspect the running container:

```bash
docker ps --filter name=monita
```

### Managed in-app updates

Docker Compose runs one persistent `monita` container. There is no always-running updater service.

From **Settings → Software Update**, Monita checks GitHub for the newest published release. An administrator can click **Update** to start the upgrade. The UI shows progress, the replacement container is health-checked, and the previous container is restored automatically if verification fails.

During an update only, Monita starts a short-lived `monita-update-worker` container from the currently installed image. It exists only long enough to perform the replacement and removes itself when finished. Persistent application data remains mounted at `/app/data`.

Managed self-updating requires the Docker socket mount included in `docker-compose.yml`. Docker socket access is host-privileged. If managed updates are not wanted, remove that mount and use the manual update procedure instead.

### Updating a development installation

Development and preview builds should be validated with the full Web UI build and Go test suite before replacing a running container. See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for the validated upgrade and rollback procedure.

For ordinary Compose development after changes are merged into `master`:

```bash
cd /opt/monita
git pull --ff-only origin master
docker compose build --build-arg RUN_TESTS=1
docker compose up -d
```

Your `./data` directory remains in place. Keep a verified pre-upgrade data backup whenever a build introduces database migrations.

### Building the current published release manually

After checking out the release tag, build with explicit release identity:

```bash
cd /opt/monita

COMMIT="$(git rev-parse --short HEAD)"

docker build --no-cache \
  --build-arg BUILD_JS=1 \
  --build-arg GO_VERSION=1.26.0 \
  --build-arg MONITA_VERSION="1.1.0" \
  --build-arg MONITA_COMMIT="${COMMIT}" \
  -f docker/Dockerfile \
  -t monita:1.1.0 \
  .
```

Use the same persistent `/app/data` mount when replacing an existing container.

### Manual Docker deployment

If you do not want to use Compose:

```bash
cd /opt

git clone https://github.com/gigabytegrove/gotify-mu.git monita
cd monita

docker build \
  --build-arg BUILD_JS=1 \
  --build-arg GO_VERSION=1.26.0 \
  -f docker/Dockerfile \
  -t monita:master \
  .

mkdir -p /opt/monita-data

docker rm -f monita 2>/dev/null || true

docker run -d \
  --name monita \
  --restart unless-stopped \
  -p 8080:80 \
  -e GOTIFY_DEFAULTUSER_NAME=admin \
  -e GOTIFY_DEFAULTUSER_PASS='CHANGE-THIS-PASSWORD' \
  -v /opt/monita-data:/app/data \
  monita:master
```

The `BUILD_JS=1` build argument is required for the Docker build to include the Web UI.

### Native development/test build

You can also run Monita without Docker:

```bash
git clone https://github.com/gigabytegrove/gotify-mu.git monita
cd monita
make build-js
go build -o monita .
./monita serve
```

### First-test checklist

For the current development or preview build, verify these behaviors before treating an installation as production-ready:

1. The Monita Web UI loads.
2. The administrator can sign in.
3. Multiple users can be created.
4. A Channel can be created.
5. Multiple users can be assigned to the Channel.
6. All assigned users receive the same notification.
7. A user can mute and re-enable realtime delivery for a Channel without losing history.
8. Channel ownership can be transferred to another member.
9. A user who still owns a shared Channel cannot be deleted until ownership is transferred.
10. A non-admin can archive and restore messages without affecting other members.
11. Only an administrator can delete or clear messages in a Global Channel.
12. An administrator can enable Chat Channel posting and an assigned member can send a message.
13. Chat Channel messages show the sender identity.
14. The official Gotify Android app receives notifications normally.
15. Deleting a shared message for one user does not remove it for other members.
16. Private channels retain normal Gotify delete behavior.
17. Webhook routes can deliver JSON/plain-text payloads to the selected Channel.
18. MQTT connections can subscribe and route messages to the selected Channel.
19. Home Assistant can receive events and send a test event back successfully.
20. Scheduled Notifications run at the expected local/timezone-aware time.
21. Acknowledging a message prevents a pending Escalation from firing.
22. Quiet Hours save correctly and suppress only lower-priority realtime delivery.
23. Digest settings save correctly and collect lower-priority notifications.
24. Settings remains stable and does not enter a refresh loop after updates.

### Future container namespace

The project container namespace is reserved as:

```text
ghcr.io/gigabytegrove/monita
```

Once automated builds/releases are active, deployment will be able to use published images instead of compiling locally.

## Upgrading an existing Gotify installation

The MU database migration is additive. Existing users, applications, tokens and messages remain in place, and existing application owners are backfilled as channel members.

**Back up your database before testing an upgrade.** This project is still in active development.

## Security

The 1.0 release includes MFA/TOTP, recovery codes, passkeys, LDAP/Active Directory, service accounts, login throttling, encrypted stored secrets, session policy, audit/security administration, hardened Webhooks, plugin signature verification, and a hardened managed updater.

See `docs/SECURITY_ROADMAP.md` for implementation status and trust-boundary details.

## Development

Server: Go  
Web UI: TypeScript / React  
Database layer: GORM  
Realtime delivery: existing Gotify WebSocket stream

The Go module remains `github.com/gotify/server/v3` for upstream/plugin compatibility. That is intentional and should not be changed as a branding exercise.

## Upstream relationship

Monita is based on Gotify and retains substantial upstream code and compatibility. The transition repository is maintained at:

https://github.com/gigabytegrove/gotify-mu

For upstream Gotify documentation and ecosystem information, see [gotify.net](https://gotify.net/).

Monita is not presented as an official Gotify project.

## Contributing

Issues and pull requests for Monita should be opened in this repository. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Monita remains licensed under the MIT License inherited from Gotify. See [LICENSE](LICENSE).
