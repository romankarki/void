# Void Packaging Strategy

## Goals

- Ship a single executable (`void.exe`) for users.
- Keep installation user-level (no admin required by default).
- Support in-place updates from the CLI.
- Ask before making shell profile changes.

## User Flow

1. User downloads `void.exe` from GitHub Releases.
2. User runs:
   - `void install`
3. Installer performs:
   - Copy binary to `%LOCALAPPDATA%\Void\bin\void.exe`
   - Create `~/.void/config.toml` if absent
   - Prompt to add install directory to user `PATH`
   - Prompt to append prompt integration to shell profile
4. User updates later with:
   - `void update`

## Commands

- `void install`
- `void install --yes`
- `void install --shell powershell`
- `void install --no-profile`
- `void update`
- `void update --repo owner/repo`

## Release Artifact Convention

- `void-windows-amd64.exe`
- `void-windows-arm64.exe`
- `void-linux-amd64`
- `void-linux-arm64`
- `void-darwin-amd64`
- `void-darwin-arm64`

`void update` resolves the expected filename from `GOOS/GOARCH` and downloads:

`https://github.com/<owner>/<repo>/releases/latest/download/<asset>`

## Maintainer Strategy

1. Build cross-platform binaries on release tags.
2. Upload assets to a GitHub Release using the artifact names above.
3. Keep install path stable to avoid repeated profile/PATH churn.
4. Treat `void.exe` as both runtime and installer/updater entrypoint.

Use the included build script for artifact generation:

`powershell -ExecutionPolicy Bypass -File scripts/build-release.ps1 -OutDir dist`

## Notes

- On Windows, replacing the running executable is handled via a small temporary `.cmd` update script.
- Profile modification is additive and marker-based to avoid duplicate blocks.
- PATH updates are user-scoped.
