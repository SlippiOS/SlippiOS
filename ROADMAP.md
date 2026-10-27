# Roadmap

Current priority: make the USB appliance reliable on real hardware before adding
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
- [x] Installer rejects a `systemd-sysupdate` no-op instead of recording a
      false install.

## Now — USB image

- [ ] Flash a 16 GB USB stick and boot on real UEFI hardware with Secure Boot off.
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

## Next — release confidence

- [ ] Make the repository public (release downloads currently 404 for
      unauthenticated devices), publish a release, and verify unauthenticated
      update downloads.
- [ ] Test update failure paths: no network, bad checksum, interrupted download.
- [ ] Test power loss during writes and boot-counter rollback.
- [ ] Stop offering a version that just failed boot assessment until a newer
      release appears; the Update screen currently offers it again.
- [ ] Add a backup/restore path for `/data`.
- [ ] Test on a second machine to verify hardware binding and login removal.

## Later — install to internal disk

- [ ] Add an **Install to internal disk** mode to the USB shell.
- [ ] Require explicit target-disk selection and erase confirmation.
- [ ] Copy the A/B OS layout to the internal disk and create fresh `/data` and
      `MELEE-ROMS` partitions.
- [ ] Install the UEFI boot entry and reboot into the internal disk.
- [ ] Keep the USB image as the installer medium.
- [ ] Initial scope: UEFI only, erase the whole target disk, fresh install.
- [ ] Decide later whether dual boot and USB-data migration are in scope.

## Not planned yet

- Dual-boot alongside Windows or another Linux install.
- Encrypted internal disks.
- Migrating an existing USB `/data` into an internal install.
- Mod-specific management UI beyond selecting and launching Melee images.
- A separate installer ISO; the USB image is the installer medium.
