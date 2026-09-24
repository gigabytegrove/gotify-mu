# Contributing to Gotify MU

Thanks for your interest in Gotify MU.

Gotify MU is a compatibility-focused fork of [Gotify Server](https://github.com/gotify/server) that adds real multi-user notification channels.

## Where to contribute

For MU-specific bugs, features, documentation, and Web UI changes, use this repository:

https://github.com/gigabytegrove/gotify-mu

If a problem also exists unchanged in upstream Gotify, please say so in the issue. That helps us decide whether the fix belongs only here or should also be proposed upstream.

## Compatibility rules

Changes should preserve normal Gotify clients and integrations whenever practical.

In particular:

- do not casually change the `github.com/gotify/server/v3` Go module path
- preserve existing Gotify API routes such as `/application`, `/message`, and `/stream`
- keep existing application-token and client-token behavior compatible
- treat new MU response fields as additive
- do not grant publish permission simply because a user can receive/read a channel
- migrations must preserve existing users, applications, tokens, and messages

Visible UI terminology may use **Channels** even where the compatibility API still uses **Application** internally.

## Pull requests

Use a branch and open a pull request against `master`.

Please include:

- what changed
- why it changed
- compatibility impact
- migration impact, if any
- tests performed
- screenshots for meaningful Web UI changes

## Development

Server code is Go.  
The Web UI is TypeScript/React.

Common checks:

```bash
go test ./...
cd ui
yarn
yarn lint
yarn test
yarn build
```

## Upstream credit

Gotify MU is derived from Gotify and remains under the MIT License. Keep applicable upstream copyright and license notices intact.
