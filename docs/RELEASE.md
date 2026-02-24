# Release Process

Draupnir releases are fully automated via GitHub Actions. Pushing a version tag is the only manual step required — the workflows handle testing, cross-compilation, changelog generation, GitHub Release creation, and propagation to the Sindri extension registry.

## Quick release steps

```bash
# 1. Ensure main is clean and all CI checks pass
git checkout main
git pull origin main

# 2. Create and push a version tag
git tag v1.2.3
git push origin v1.2.3

# 3. Monitor progress
# https://github.com/pacphi/draupnir/actions
```

That is the complete process. Everything else is automated.

---

## Version numbering

Draupnir follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

| Change type | Version bump | Example |
|-------------|-------------|---------|
| Breaking protocol or API change | MAJOR | `1.x.x` → `2.0.0` |
| New feature, backward-compatible | MINOR | `1.2.x` → `1.3.0` |
| Bug fix, security patch | PATCH | `1.2.3` → `1.2.4` |

### Tag format

Tags must match `v[MAJOR].[MINOR].[PATCH](-[PRERELEASE])?`. The release workflow validates this and fails early on an invalid format.

| Tag | Type | Marked as pre-release? |
|-----|------|----------------------|
| `v1.2.3` | Stable release | No |
| `v1.2.3-alpha.1` | Alpha | Yes |
| `v1.2.3-beta.1` | Beta | Yes |
| `v1.2.3-rc.1` | Release candidate | Yes |

Pre-releases are not promoted as the "latest" release on GitHub and are excluded from the CHANGELOG.

---

## Commit message convention

The changelog is generated automatically from commit messages. Use [Conventional Commits](https://www.conventionalcommits.org/) to ensure your work appears correctly:

| Prefix | Changelog section | Example |
|--------|------------------|---------|
| `feat` | Added | `feat(terminal): support PTY resize mid-session` |
| `fix` | Fixed | `fix(websocket): handle reconnect after TLS failure` |
| `perf` | Performance | `perf(metrics): reduce gopsutil call overhead` |
| `refactor` | Changed | `refactor(config): consolidate validation logic` |
| `docs` | Documentation | `docs: add DEPLOYMENT.md` |
| `test` | Testing | `test(heartbeat): add interval boundary tests` |
| `chore(deps)` | Dependencies | `chore(deps): update gopsutil to v4.26.1` |
| `ci`, `chore` | *(skipped)* | Internal housekeeping, not user-facing |

Breaking changes must include a `!` after the type or a `BREAKING CHANGE:` footer:

```
feat(protocol)!: add required session_token field to registration payload

BREAKING CHANGE: All agents must now include session_token in the registration
payload. Mimir versions before 2.1.0 will reject registrations from this agent.
```

---

## What happens when you push a tag

Four GitHub Actions workflows run in sequence:

### 1. `release.yml` — Cross-compile and publish

Triggered by: `push` to tags matching `v*`

| Job | What it does |
|-----|-------------|
| `validate-tag` | Parses tag, validates semver format, detects pre-release |
| `test` | Runs full test suite (`go test ./...`) |
| `changelog` | Runs [git-cliff](https://git-cliff.org/) to regenerate `CHANGELOG.md` from all conventional commits, commits the updated file to `main`, and outputs the new release section body |
| `build` (×4) | Cross-compiles `draupnir-{os}-{arch}` for all four targets with `CGO_ENABLED=0`; generates a SHA256 checksum file per binary |
| `release` | Creates a GitHub Release named `Draupnir v{version}` with the changelog body as release notes and all four binaries plus `checksums.txt` as assets |

### 2. `update-extension.yml` — Propagate to Sindri registry

Triggered by: GitHub Release published (fires after `release.yml` completes)

1. Checks out the [pacphi/sindri](https://github.com/pacphi/sindri) repository using `SINDRI_PAT`
2. Updates `version:` in `v3/extensions/draupnir/extension.yaml` to the new version
3. Opens a PR against `sindri/main` titled `chore(extensions): bump draupnir to v{version}`

This makes every Draupnir release self-propagating into the Sindri extension registry with no manual steps.

### 3. `ci.yml` — Continuous integration gate

Triggered by: push or PR to `main`

Runs on every code change (not release-specific):

| Job | Checks |
|-----|--------|
| `lint` | `go vet`, `go mod tidy` idempotency, `gofmt` formatting |
| `deadcode` | Dead code scan (`continue-on-error`) |
| `test` | Unit tests with race detector on Ubuntu and macOS |
| `build` | Cross-compile all 4 targets (sanity check) |

### 4. `test.yml` — Extended test & lint

Triggered by: push or PR to `main` or `develop`; also manually dispatchable

| Job | Checks |
|-----|--------|
| `lint` | `golangci-lint` (full linter suite, 5-minute timeout) |
| `test` | Unit tests, unit tests with race detector, coverage report upload |
| `cross-compile` | All 4 targets with binary size check (20 MB limit enforced) |

---

## Required repository secret

The `update-extension.yml` workflow requires one secret to be configured in the repository settings:

| Secret | Purpose |
|--------|---------|
| `SINDRI_PAT` | Personal access token with `repo` scope on `pacphi/sindri`, used to check out the sindri repo and open the cross-repo PR |

Without `SINDRI_PAT`, the release itself succeeds but the Sindri extension registry is not updated automatically.

---

## Release assets

Each stable release attaches these files:

| File | Description |
|------|-------------|
| `draupnir-linux-amd64` | Linux x86-64 static binary |
| `draupnir-linux-arm64` | Linux ARM64 static binary |
| `draupnir-darwin-amd64` | macOS Intel static binary |
| `draupnir-darwin-arm64` | macOS Apple Silicon static binary |
| `checksums.txt` | SHA256 checksums for all four binaries |

Binaries are built with `-ldflags "-s -w"` (stripped debug info) and `-trimpath` for reproducible builds.

---

## Post-release verification checklist

After the workflows complete, verify:

- [ ] GitHub Release page shows the correct version name and changelog body
- [ ] All four binaries and `checksums.txt` are attached to the release
- [ ] `CHANGELOG.md` on `main` is updated with the new version section
- [ ] A PR is open against `pacphi/sindri` bumping the draupnir extension version
- [ ] `sindri extension install draupnir` resolves the new version (after the sindri PR is merged)

---

## Changelog management

The `CHANGELOG.md` at the repository root is the authoritative release history. It is:

- Regenerated in full on every release by [git-cliff](https://git-cliff.org/) using `cliff.toml`
- Committed to `main` by the release workflow as `chore(release): update CHANGELOG.md for vX.Y.Z`
- Formatted to [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions

Do not edit `CHANGELOG.md` by hand. All changes flow from conventional commit messages.

To preview what the next release changelog will look like locally:

```bash
# Install git-cliff
cargo install git-cliff
# or: brew install git-cliff

# Preview unreleased changes
git-cliff --config cliff.toml --unreleased

# Preview full changelog for a specific tag
git-cliff --config cliff.toml --tag v1.2.3
```
