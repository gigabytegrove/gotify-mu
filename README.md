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

## Deployment

Gotify MU currently follows the upstream Gotify configuration model. Existing `GOTIFY_*` environment variables are intentionally retained for compatibility.

> **Testing status:** there is not yet a published Gotify MU container image. For now, deploy by building directly from this repository.

### Recommended: Docker Compose

On a Linux system with Git and Docker Compose installed:

```bash
cd /opt
git clone https://github.com/gigabytegrove/gotify-mu.git
cd gotify-mu

cp .env.example .env
nano .env
```

At minimum, change:

```text
GOTIFY_DEFAULTUSER_PASS=CHANGE-THIS-PASSWORD
```

Then build and start Gotify MU:

```bash
docker compose up -d --build
```

The default deployment publishes Gotify MU on port `8080`.

Open:

```text
http://SERVER-IP:8080
```

Default username:

```text
admin
```

The password is whatever you set in `.env`.

Persistent data is stored in:

```text
./data
```

That directory contains the SQLite database and other persistent Gotify data. Rebuilding or replacing the container does not remove it.

### Verify the deployment

Watch startup logs:

```bash
docker logs -f gotify-mu
```

Check the health endpoint:

```bash
curl http://127.0.0.1:8080/health
```

Inspect the running container:

```bash
docker ps --filter name=gotify-mu
```

### Updating a test installation

After new changes are merged into `master`:

```bash
cd /opt/gotify-mu
git pull origin master
docker compose up -d --build
```

Your `./data` directory remains in place.

### Manual Docker deployment

If you do not want to use Compose:

```bash
cd /opt

git clone https://github.com/gigabytegrove/gotify-mu.git
cd gotify-mu

docker build \
  --build-arg BUILD_JS=1 \
  --build-arg GO_VERSION=1.26.0 \
  -f docker/Dockerfile \
  -t gotify-mu:master \
  .

mkdir -p /opt/gotify-mu-data

docker rm -f gotify-mu 2>/dev/null || true

docker run -d \
  --name gotify-mu \
  --restart unless-stopped \
  -p 8080:80 \
  -e GOTIFY_DEFAULTUSER_NAME=admin \
  -e GOTIFY_DEFAULTUSER_PASS='CHANGE-THIS-PASSWORD' \
  -v /opt/gotify-mu-data:/app/data \
  gotify-mu:master
```

The `BUILD_JS=1` build argument is required for the Docker build to include the Web UI.

### Native development/test build

You can also run Gotify MU without Docker:

```bash
git clone https://github.com/gigabytegrove/gotify-mu.git
cd gotify-mu
make build-js
go build -o gotify-mu .
./gotify-mu serve
```

### First-test checklist

For the current development build, verify these behaviors before treating an installation as production-ready:

1. The Gotify MU Web UI loads.
2. The administrator can sign in.
3. Multiple users can be created.
4. A Channel can be created.
5. Multiple users can be assigned to the Channel.
6. All assigned users receive the same notification.
7. The official Gotify Android app receives notifications normally.
8. Deleting a shared message for one user does not remove it for other members.
9. Private channels retain normal Gotify delete behavior.

### Future container namespace

The project container namespace is reserved as:

```text
ghcr.io/gigabytegrove/gotify-mu
```

Once automated builds/releases are active, deployment will be able to use published images instead of compiling locally.

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
