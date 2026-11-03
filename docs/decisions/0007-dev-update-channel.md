# ADR-0007: Add an expert-only dev update channel

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-11-03 |
| **Deciders** | Ryan Wendell |

## Context

Testing a shell or backend change currently means rebuilding the USB image and
flashing the drive. The update system already stages a store image and UKI and
installs them into the inactive slot, so a running console can update without
reflashing. The release channel is hard-wired to the R2 root, and there is no
way to point a test console at a different build.

Cloudflare R2's free tier is 10 GB-month. The stable update assets already use
part of that budget. A dev channel has to replace its own assets rather than
accumulate every test build.

## Decision

Add `[update] channel` to `settings.toml`, with `stable` as the default and
`dev` as the test channel. The shell shows the switch in
**Settings → System** only while Expert mode is on. `osupdate.source` selects
the R2 root for stable and the `dev/` prefix for dev, unless
`/data/system/update-source.json` overrides it.

Dev images use `<release>.<checkout-timestamp>` as their version. The timestamp
makes each committed build sort above the release and above earlier dev builds,
so `systemd`'s version comparison always offers the new build. Dev builds
require a clean tree; uncommitted changes would reuse the last commit's version.

`scripts/release-dev` builds the uncompressed dev image, signs it, extracts the
store image and UKI, writes a manifest with only the update fields, and
publishes through `scripts/publish-r2 --dev`. The first build is kept as a
signed full image for the one-time flash. Later builds only produce the update
assets.

`scripts/publish-r2 --dev` uploads the new assets (large files first, manifest
last), then deletes old objects under `dev/`. Stable objects are untouched. The
bucket therefore holds the latest stable plus the latest dev assets, which
keeps the 10 GB free tier from filling with old test builds.

## Consequences

- A test console can switch to Dev in Settings → System and install new builds
  from the normal update screen. The initial dev image is flashed once.
- Dev builds are test builds. They use the same signing key and update path as
  releases, so a failed boot still rolls back after three attempts.
- The dev channel keeps only the newest build. A console that is several dev
  builds behind cannot fetch the intermediate build; it installs the newest
  one, which is the intended workflow.
- The dev version uses the flake's last-modified timestamp. Two builds from the
  same commit share a version, so a new test build needs a new commit.
- Switching back to Stable does not downgrade a dev build; the console waits
  for a newer stable release or is reflashed.
- The expert-only gate keeps the test channel out of the normal user path.
- Stable cleanup remains manual. If stable assets plus the latest dev assets
  exceed 10 GB, the publish script refuses the upload and reports the projected
  usage.
