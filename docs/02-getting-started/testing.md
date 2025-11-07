# Testing & Coverage

Hall Monitor ships with a growing test suite to keep core features reliable. This guide explains how to run the tests locally, review coverage output, and understand the CI pipeline.

## Quick Commands

- `make test` – run the standard Go test suite across all packages
- `make test-race` – run tests with the race detector enabled (slower, but catches data races)
- `make test-coverage` – run tests and generate `coverage.out` plus a package coverage summary

All commands operate from the repository root. Artifacts such as `coverage.out` can be removed with `make clean`.

## Coverage Helper Script

`make test-coverage` wraps `scripts/coverage.sh`, which:

1. Runs `go test ./...` with `-covermode=atomic`
2. Emits a per-package coverage summary sorted from lowest to highest coverage
3. Produces `coverage.out` (and optionally `coverage.html` when invoked with `--html`)

You can call the script directly for additional options:

```bash
./scripts/coverage.sh --html                      # adds coverage.html for local inspection
./scripts/coverage.sh --packages ./internal/api   # scope coverage to a subset of packages
```

Per-package numbers in the summary help prioritize low-coverage areas. The final “total” line matches the value reported to Codecov.

## CI & Codecov

- The `ci` workflow runs on every push to `main` and on pull requests
- CI executes `make test-coverage` and uploads `coverage.out` via the official Codecov action
- Releases reuse the same coverage step before building binaries
- Project-wide coverage is required to stay at or above **80%**, with a small 2% leeway for natural variance (configured in `codecov.yml`)
- Patch coverage targets the same 80% goal to discourage regressions in new code

For private repositories you must provide `CODECOV_TOKEN` in the repository secrets so the GitHub Action can authenticate. Public repositories do not require a token, but the secret is respected when present.

## Troubleshooting

- **Ping monitor tests fail locally**: ICMP requires elevated privileges on some systems. Running the suite without raw socket capabilities falls back to an unprivileged mode automatically; the tests simulate this path.
- **Coverage report missing packages**: Ensure `coverage.out` exists in the repository root (created by `make test-coverage`). Other tooling reads that path by default.
- **Codecov upload failures in CI**: Confirm `CODECOV_TOKEN` is configured for private repos and that the Codecov Action is pinned to version `v5`. The CI logs link to detailed error messages.

Keeping `make test-coverage` green locally is the quickest way to verify that the CI check and Codecov gate will pass.

