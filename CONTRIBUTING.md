<p align="center">
  <img src="assets/monita-banner.svg" alt="Monita" width="720">
</p>

# Contributing to Monita

Thanks for your interest in Monita.

Contributions are welcome for bug fixes, documentation, integrations, accessibility, user experience, and new features.

## Before opening a pull request

Please:

- check existing Issues and pull requests
- keep changes focused
- explain what problem the change solves
- preserve Gotify compatibility where practical
- include screenshots for visible UI changes
- add or update tests when behavior changes

## Development

Monita uses:

- Go for the server
- TypeScript and React for the Web UI
- Docker for the recommended deployment model

Common checks include:

```bash
go test ./...
cd ui
yarn
yarn lint
yarn test
yarn build
```

## Compatibility

Monita is built from the Gotify Server codebase and continues to support Gotify-compatible clients and integrations.

Changes should avoid breaking existing users unless there is a clear migration path.

## Pull requests

Open pull requests against `master`.

A useful pull request description includes:

- what changed
- why it changed
- user-visible impact
- compatibility impact
- testing performed

## Security issues

Do not open a public issue for a suspected unpatched vulnerability.

See [SECURITY.md](SECURITY.md) for private reporting instructions.

## License

By contributing, you agree that your contribution may be distributed under the project's MIT License.
