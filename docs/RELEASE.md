# Release Guide

## How Releases Work

Releases are automated via GitHub Actions and GoReleaser. When you push a tag starting with `v`, the release workflow runs automatically and builds binaries for Windows, macOS, and Linux.

## How To Create A Release

Step by step:

```bash
git checkout main
git pull origin main
git tag v1.2.0
git push origin v1.2.0
```

The GitHub Actions `release.yml` workflow then:

1. Builds binaries for all platforms using GoReleaser
2. Creates a GitHub Release automatically
3. Uploads the binaries as downloadable `.zip` and `.tar.gz` files
4. Generates a checksum file

## Naming Your Version

- `v1.0.0` -> first stable release
- `v1.1.0` -> new features added
- `v1.1.1` -> bug fix, nothing new added
