# Release and Distribution

This project can be distributed to users in 3 ways.

## 1) Source-first (fastest)
Tell users to run:
```bash
go install github.com/Polqt/gitflowtui/cmd/gitflow-tui@latest
```

## 2) Prebuilt binaries via GitHub Releases
1. Tag a release:
```bash
git tag v0.1.0
git push origin v0.1.0
```
2. Build artifacts with GoReleaser (config in `.goreleaser.yaml`):
```bash
goreleaser release --clean
```
3. Upload archives to GitHub Release.

## 3) Package-manager wrappers
- Homebrew tap (macOS)
- Scoop manifest (Windows)
- AUR package (Arch)

These wrappers install the same release binaries.

## Recommended
Start with option 1 now, then move to option 2 for broader adoption.
