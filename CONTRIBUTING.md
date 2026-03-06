# Contributing

## Development

```sh
go test ./...
go build .
```

CI runs tests, linting, and a security scan on every PR against `main`.

## Releasing

Releases are intentional — no merge auto-tags. After merging a PR:

1. Go to **Actions → Release → Run workflow** on GitHub
2. Choose a bump type and run against `main`

| Bump | When to use |
|------|-------------|
| `patch` | Bug fixes, docs, internal refactors — no API change |
| `minor` | New commands or flags added in a backward-compatible way |
| `major` | Breaking changes to the CLI interface or flag behaviour |

> **Note:** while the project is `v0.x`, a `minor` bump may still contain breaking changes per semver convention. Prefer `minor` for any meaningful change and document it in the PR description so the release notes are informative.

The workflow tags `HEAD` of `main` and creates a GitHub Release with auto-generated notes from merged PR titles.
