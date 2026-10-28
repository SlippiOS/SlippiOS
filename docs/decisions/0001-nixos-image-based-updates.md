# ADR-0001: Use NixOS as the image builder, with image-based A/B updates

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-10-27 |
| **Deciders** | Ryan Wendell |

## Context

SlippiOS ships as a fixed-function Melee console. The system image is built
reproducibly, signed for Secure Boot, and updated offline from a release
manifest. The Nix store is a read-only squashfs partition, not a writable
directory. The device has two store partitions and one ESP; `systemd-sysupdate`
writes the inactive partition and installs a matching UKI. `systemd-boot`
selects the newest UKI and rolls back after three failed boots.

NixOS normally tracks updates as generations: each `nixos-rebuild` adds a new
system closure to a writable `/nix/store` and a bootloader entry for it. That
model does not fit a read-only store image or a signed, prebuilt release.

The boot menu currently shows the UKI's `PRETTY_NAME`, which includes the NixOS
release. It does not show a generation number. After an update, the menu can
show two SlippiOS UKIs because the previous slot is kept for rollback.

## Decision

We use NixOS as the build and package substrate, and image-based A/B as the
update and rollback model. We do not use NixOS generations for system updates,
and we do not display a NixOS generation at boot. The user-visible identity is
the SlippiOS version (`/etc/slippios/version`, also the UKI filename and
`IMAGE_VERSION`).

The boot menu title is `SlippiOS <version>`. Stale `nixos-generation-*.conf`
entries from older images are removed during an update.

## Consequences

**Positive:** Updates are atomic, signed, and bounded to two store slots. The
device needs no package manager or writable `/nix/store`. Builds are
reproducible from pinned inputs. Rollback is handled by systemd-boot boot
assessment rather than by a generation menu. The update payload is a single
store image plus a UKI, which is straightforward to publish and verify.

**Negative:** We own the updater, slot selection, boot assessment, and recovery
paths. A kernel, firmware, or package change requires a new store image and
UKI, so updates are image-sized even when zsync only transfers changed blocks.
There is one rollback slot, not an unlimited generation history. NixOS
generation tooling and documentation do not apply, so support must identify
systems by SlippiOS version. The Secure Boot chain (shim, MOK, signed
systemd-boot, signed UKI) has more moving parts than a single signed UKI.

**Neutral:** Store partitions remain labelled `nix-store_<version>` because
that is what `systemd-sysupdate` matches and what the UKI embeds. `/data`
remains the only writable system partition and is never touched by an update.

## Alternatives considered

**NixOS generations with a writable `/nix/store`** - For: many rollback points,
package-level updates, familiar NixOS workflow. Against: requires a writable
store, conflicts with the signed read-only image, grows without a fixed bound,
and needs generation entries in the bootloader. Why not chosen: it gives up the
atomic signed image and makes offline recovery harder.

**Fedora IoT/Silverblue with rpm-ostree** - For: native A/B deployment, delta
updates, mature tooling. Against: a base rewrite, less reproducible closure
construction, and a larger package/tooling surface. Why not chosen: no
concrete blocker has justified replacing the existing NixOS image pipeline.

**Debian/Ubuntu with RAUC or swupdate** - For: mature A/B update frameworks and
broad hardware support. Against: significant integration work, less declarative
system configuration, and a different signing/verification path. Why not
chosen: the current systemd-sysupdate path already works and is smaller.

**Buildroot** - For: small image, fast boot, direct control. Against: high
maintenance for packages, firmware, and graphics; no Nix reproducibility. Why
not chosen: the maintenance cost is not justified for this console.

## References

- [Architecture](../architecture.md)
- [System update service](../../hosts/usb/update.nix)
- [Image layout](../../hosts/usb/image.nix)
- [Boot configuration](../../hosts/usb/boot.nix)
