# Release and Distribution

This project can be distributed to users in 3 ways.

## 1) Source-first (fastest)
Tell users to run:
```bash
go install github.com/Polqt/gitflowtui/cmd/gitflow-tui@latest
```

## 2) Prebuilt binaries via GitHub Releases
1. Push the release workflow and GoReleaser config to `main`.
2. Tag a release:
```bash
git tag v0.1.0
git push origin v0.1.0
```
3. GitHub Actions runs `.github/workflows/release.yml` automatically and publishes archives to the GitHub Release using GoReleaser.
4. Optional local release build (config in `.goreleaser.yaml`):
```bash
goreleaser release --clean
```
The workflow requires `contents: write` permission, which is already set in the workflow file.

## 3) Package-manager wrappers
- Homebrew tap (macOS)
- Scoop manifest (Windows)
- AUR package (Arch)

These wrappers install the same release binaries.

## Recommended
Start with option 1 now, then move to option 2 for broader adoption.
