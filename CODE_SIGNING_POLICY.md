# Code Signing Policy

Free code signing provided by [SignPath.io](https://signpath.io/), certificate by
[SignPath Foundation](https://signpath.org/).

This policy covers the Windows release artifacts of the **WoW Backup Tool**
project (`github.com/panle80/wow-backup-tool-go`).

## Team roles

| Role | Person |
|---|---|
| Author / Committer | [@panle80](https://github.com/panle80) |
| Reviewer | [@panle80](https://github.com/panle80) |
| Approver | [@panle80](https://github.com/panle80) |

Everyone listed above uses multi-factor authentication for both GitHub and
SignPath. Additional maintainers, if any, are added to this table before they
are granted a role.

## What is signed

- `WoWBackupTool.exe`, built by the trusted GitHub Actions build defined in
  [`.github/workflows/build.yml`](.github/workflows/build.yml) from the source
  code in this public repository.
- Only artifacts produced from a version tag (`v*`) by that CI workflow are
  submitted for signing. Pull-request and branch builds are never signed.
- Every signing request is verified against the public source revision and is
  manually approved in SignPath by a designated approver.
- No third-party binaries or bundled components are submitted for signing.

## Privacy

This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or operating
it. The application does not include telemetry or analytics and does not send
file paths, backup contents, or any other user data anywhere. All backup and
restore operations run locally.

The only persisted settings are the last-used output directory (Windows:
`HKCU\SOFTWARE\pan.wowbackup`, macOS: `UserDefaults`) and an optional
`config.json` created manually by the user.
