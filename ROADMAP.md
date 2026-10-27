# Roadmap

Current priority: make the USB image reliable on real hardware before adding
more installation modes.

## Verified in QEMU

- [x] Cold-boot splash, browser wizard QR, and **Skip for now**.
- [x] Menu, `Melee ROMs`, and default-image selection persisted in `settings.toml`.
- [x] Cold-boot autoplay, Dolphin first-launch download, and reopening the shell
      after Dolphin exits without a second splash.
- [x] `/data` grew from 1 GiB to fill a 16 GB disk and survived a reboot.
- [x] Settings and Update screens render.
- [x] OS update: staged, installed, rebooted, and rolled back after three
      failed boots; the shell reported the rollback and `/data` survived.
- [x] Upgraded v6 to v8 with the decimal version format; `/data` survived and
      the installer rejected a `systemd-sysupdate` no-op instead of recording
      a false install.
- [x] Secure Boot: Fedora 16.1-7 dual-signed shim, SlippiOS MOK key, signed
      systemd-boot and UKI; first-boot enrollment and a re-signed UKI boot
      verified under OVMF.

## Now: USB image

- [ ] Flash a 16 GB USB stick and boot on real UEFI hardware with Secure Boot
      off (a one-time firmware setting).
- [ ] Boot the signed image with Secure Boot on and complete the one-time MOK
      enrollment.
- [ ] Confirm Windows and macOS see and mount `MELEE-ROMS`.
- [ ] Copy vanilla Melee, Training Mode CE, and one modded ISO; confirm all three
      fit and appear with their original filenames.
- [ ] Choose a default image, power-cycle, and confirm it launches automatically.
- [ ] Open `Melee ROMs`, switch images, and confirm the new default persists.
- [ ] Confirm the hold-A menu path on hardware. (Cold-boot Plymouth/Godot
      splash and reopening the shell without a splash are verified in QEMU.)
- [ ] Replace the temporary Slippi mark with a SlippiOS mark for Plymouth and
      the Godot splash.
- [ ] Test the real GameCube adapter in the shell and Dolphin.
- [ ] Test Wi-Fi setup and a real Slippi sign-in.
- [ ] Confirm `/data` grows to fill the stick and survives reboots.
- [ ] Test an OS update end to end: stage, install, reboot, and rollback.

## Next: release confidence

- [ ] Make the repository public (release downloads currently 404 for
      unauthenticated devices), publish a release, and verify unauthenticated
      update downloads. Secure Boot is not a release gate; the signed image is
      a separate asset. See [Secure Boot and Mac boot](#later-secure-boot-and-mac-boot).
- [ ] Verify the local release script: build, sign, generate assets, and
      confirm both images and the signed UKI are published.
- [ ] Test update failure paths: no network, bad checksum, interrupted download.
- [ ] Test power loss during writes and boot-counter rollback.
- [ ] Stop offering a version that just failed boot assessment until a newer
      release appears; the Update screen currently offers it again.
- [ ] Add a backup/restore path for `/data`.
- [ ] Test on a second machine to verify hardware binding and login removal.

## Later: install to internal disk

- [ ] Add an **Install to internal disk** mode to the USB shell.
- [ ] Require explicit target-disk selection and erase confirmation.
- [ ] Copy the A/B OS layout to the internal disk and create fresh `/data` and
      `MELEE-ROMS` partitions.
- [ ] Install the UEFI boot entry and reboot into the internal disk.
- [ ] Keep the USB image as the installer medium.
- [ ] Initial scope: UEFI only, erase the whole target disk, fresh install.
- [ ] Decide later whether dual boot and USB-data migration are in scope.

## Later: Secure Boot and Mac boot

The signed image prototype uses a Fedora 16.1-7 shim, dual-signed by Microsoft
UEFI CA 2011 and 2023, plus a SlippiOS MOK key. The first boot requires a
one-time MOK enrollment per PC; updates use the same key. The unsigned image
still requires Secure Boot off.

- [x] Add local release signing with `scripts/release`; it signs the image and
      extracts the signed UKI for the manifest.
- [x] Document the one-time MOK enrollment in the user guide and keep the
      unsigned image as a fallback.
- [ ] Document per-vendor Secure Boot off steps for the unsigned fallback.
- [ ] Test the signed image on real UEFI hardware with Secure Boot on.
- [ ] Evaluate a custom Microsoft-signed shim with an embedded SlippiOS
      certificate for zero-interaction boot. This needs a legal entity, an EV
      code-signing certificate, hardware key storage, shim-review approval, and
      Microsoft signing.
- [ ] Investigate Mac boot support. Apple Silicon cannot run this x86_64 image.
      Intel T2 Macs require **Allow booting from external or removable media**;
      pre-T2 Intel Macs can boot USB without a firmware change.

## Later: window manager experiment

The normal session runs without a window manager. The shell applies the saved
display mode before starting, opens at the screen resolution, and sizes a
borderless window; Dolphin uses the current X resolution and borderless
fullscreen. Openbox is only the crash/close fallback.

A lightweight kiosk window manager is a proposed general fix for fullscreen,
window sizing, focus, and Dolphin dialog placement. Openbox is the first
candidate because it is already installed and handles
`_NET_WM_STATE_FULLSCREEN`; `matchbox-window-manager` is a lighter alternative.
A prototype needs a minimal config with no decorations, no root menu, no
keybindings, and new-window focus.

- [ ] Prototype a minimal Openbox session in the dev VM.
- [ ] Measure cold boot time and memory/CPU overhead.
- [ ] Test the shell at 640x480 and 1280x720.
- [ ] Test Dolphin fullscreen, settings window, and file dialogs.
- [ ] Measure input latency and game frame pacing.
- [ ] Check focus stealing, accidental close, and fallback behavior.
- [ ] Adopt only if it is clearly better than the targeted no-WM fixes.

Decision: keep no-WM for now.

## Not planned yet

- Dual-boot alongside Windows or another Linux install.
- Encrypted internal disks.
- Migrating an existing USB `/data` into an internal install.
- Mod-specific management UI beyond selecting and launching Melee images.
- A separate installer ISO; the USB image is the installer medium.
