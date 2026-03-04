# Changelog

All notable changes to Draupnir are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is automatically updated by the release workflow on each tagged release. Do not edit it by hand — use [Conventional Commits](https://www.conventionalcommits.org/) in your commit messages and the changelog will reflect your changes at the next release.

## [1.1.0] — 2026-03-04

### Added

- Add help target to Makefile with categorized target listing ([`3f34fa0`](https://github.com/pacphi/draupnir/commit/3f34fa0eb07bf5822bdd9a295ae1aeec7ad6f930))

- Add LLM traffic interception via two-tier proxy and eBPF ([`df5bd83`](https://github.com/pacphi/draupnir/commit/df5bd83d5fc5448d872449d618b64a2e16561421))


### Dependencies

- **deps:** Bump github.com/shirou/gopsutil/v4 from 4.26.1 to 4.26.2 (#3) ([`5433213`](https://github.com/pacphi/draupnir/commit/5433213bd276bc5d9681812a554a9b88c546ffb8))

- **deps:** Bump actions/upload-artifact from 6 to 7 (#4) ([`9ff3bcb`](https://github.com/pacphi/draupnir/commit/9ff3bcb06f8daafb5c4a6a9645799115847a4765))

- **deps:** Bump actions/setup-go from 5 to 6 (#5) ([`bdfff3f`](https://github.com/pacphi/draupnir/commit/bdfff3fe48d38f89f99bc030fcfe4ca41fb07a7a))

- **deps:** Bump actions/download-artifact from 7 to 8 (#6) ([`bb2f0e7`](https://github.com/pacphi/draupnir/commit/bb2f0e7dacb60bed938c9eeb99432618a39dec9f))


### Fixed

- Use PAT for release to trigger downstream workflows ([`8c9bbc8`](https://github.com/pacphi/draupnir/commit/8c9bbc8c07424fb25dd1bbaadf57494f316a8c8f))

- Handle no-op gracefully in update-extension workflow ([`c3113b1`](https://github.com/pacphi/draupnir/commit/c3113b19856572d7b9e6ad7f43c5a89be846830a))

- Resolve lint errors in eBPF loader and add cross-platform lint to pre-commit hook ([`28ec045`](https://github.com/pacphi/draupnir/commit/28ec04544d04a955788f443979b4892ac526340a))

- Use targeted sed substitutions in update-extension workflow ([`650787c`](https://github.com/pacphi/draupnir/commit/650787c608c77aca66eb54b28af95f203faed7b8))


## [1.0.0] — 2026-02-24

### Added

- Establish draupnir as the Sindri per-instance agent ([`67411e9`](https://github.com/pacphi/draupnir/commit/67411e9aeb165a8c37e7adb1195726ec52b44051))

- Add --version flag and goreleaser config ([`030281a`](https://github.com/pacphi/draupnir/commit/030281adf7da1eceec90abd4da8572c5cd00b6d2))


### Dependencies

- **deps:** Bump golangci/golangci-lint-action from 8 to 9 ([`ca28258`](https://github.com/pacphi/draupnir/commit/ca28258f6f5c9c5a9895480af6942f91f3a57e73))

- **deps:** Bump actions/setup-go from 5 to 6 ([`5846e7d`](https://github.com/pacphi/draupnir/commit/5846e7d3b1995eb91af37e8d533f63716aaccddc))


### Documentation

- Add planning/design.md ([`4143773`](https://github.com/pacphi/draupnir/commit/4143773e3b062789f3ac751d18f8908c122c992c))

- Add status badges to README ([`006b40c`](https://github.com/pacphi/draupnir/commit/006b40cac8b755eace9ea011d6615f4d52196ef0))

- Add comprehensive documentation suite ([`e01562e`](https://github.com/pacphi/draupnir/commit/e01562e1856595cc812f1f7364661f2f1f38a794))

- Add RELEASE.md to docs directory ([`e21a2e8`](https://github.com/pacphi/draupnir/commit/e21a2e85211a6d185877bf41d876ff01efa4f8e2))


[1.1.0]: https://github.com/pacphi/draupnir/releases/tag/v1.1.0
[1.0.0]: https://github.com/pacphi/draupnir/releases/tag/v1.0.0

