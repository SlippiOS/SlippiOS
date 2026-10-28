# ADR-0002: Use semver for SlippiOS versions

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-10-28 |
| **Deciders** | Ryan Wendell |

## Context

The first image versions were `<commit-epoch>.<decimal revision>`, for example
`1793162733.261165027`. The format was chosen so systemd-boot and
systemd-sysupdate could order builds with `strverscmp_improved` even when a
commit hash began with a hexadecimal letter. It is unique and monotonic, but it
is unreadable in the boot menu, the shell, release filenames, and support
conversations.

A straight replacement with `0.3.0` would sort below the existing epoch
versions. The updater would treat it as older and would not offer it, and
systemd-sysupdate would not select it as a newer slot. A migration release with
a version above the old epoch would preserve automatic updates, but it would
keep a large numeric version indefinitely.

The device owner accepted a one-time reflash, so the version scheme can change
without a compatibility release.

## Decision

We use semantic versions (`MAJOR.MINOR.PATCH`) as the SlippiOS version. The
repository `VERSION` file is the source of truth. Release images use that
version directly in `system.image.version`, the UKI filename, the partition
label, the manifest, and the boot menu.

Dev builds use the release version plus the source hash, for example
`0.3.0.<rev>`, which sorts above `0.3.0` and below the next release. The
full commit is recorded in `/etc/slippios/build` and in the release manifest,
not in the version string.

Pre-releases use `~rc` (for example `0.3.1~rc.1`). systemd sorts a trailing
`-rc` above the release, so the semver `-rc` spelling is not used for update
versions.

Existing devices on the epoch format must reflash once. After reflash,
`0.3.1`, `0.4.0`, and later updates use normal semver ordering.

## Consequences

**Positive:** The boot menu, shell, release names, and support conversations
show a readable version. Tags are conventional (`v0.3.0`). Numeric segments
order correctly (`0.10.0 > 0.9.0`), so the updater keeps working without a
custom comparison scheme. The exact build remains traceable through the commit
in the manifest and `/etc/slippios/build`.

**Negative:** Every device on an epoch version needs a one-time reflash; it
will not receive `0.3.0` as an automatic update. Releases require a manual
`VERSION` bump, and dev builds must use the release version plus the source
hash so they sort above the release and below the next release. Pre-releases must use
`~rc` rather than the semver `-rc` spelling.

**Neutral:** The build commit and release date move out of the version string
and into metadata. Partition labels and UKI filenames become semver-based.
`systemd-sysupdate` continues to compare versions, but it no longer sees epoch
numbers.

## Alternatives considered

**Date-based versions (`2026.10.28`)** - For: automatic, chronological, and
readable. Against: no feature-level meaning, and multiple releases per day need
another suffix. Why not chosen: the project already has feature releases and
semver communicates them better.

**Hybrid versions (`0.3.0.20261028`)** - For: readable and sortable, with the
build date visible. Against: longer than necessary, and two sources of truth
for the release identity. Why not chosen: semver plus manifest metadata is
cleaner.

**Keep epoch versions internally and show semver only in the UI** - For: no
reflash and no migration risk. Against: two version schemes to maintain, and
the bootloader/partition identity stays opaque. Why not chosen: the owner
accepted a one-time reflash, so the simpler single scheme is preferable.

## References

- [Architecture](../architecture.md)
- [Developer guide](../developers.md)
- [`VERSION`](../../VERSION)
- [`flake.nix`](../../flake.nix)
- [`scripts/release`](../../scripts/release)
