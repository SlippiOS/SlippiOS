# Roadmap

Current priority: make the USB appliance reliable on real hardware before adding
more installation modes.

## Now — USB image

- [ ] Flash a 16 GB USB stick and boot on real UEFI hardware with Secure Boot off.
- [ ] Confirm Windows and macOS see and mount `MELEE-ROMS`.
- [ ] Copy vanilla Melee, Training Mode CE, and one modded ISO; confirm all three
      fit and appear with their original filenames.
- [ ] Choose a default image, power-cycle, and confirm it launches automatically.
- [ ] Open `Melee ROMs`, switch images, and confirm the new default persists.
- [ ] Confirm the hold-A menu path on hardware. (Cold-boot splash and reopening
      the shell without a splash are verified in QEMU.)
- [ ] Test the real GameCube adapter in the shell and Dolphin.
- [ ] Test Wi-Fi setup and a real Slippi sign-in.
- [ ] Confirm `/data` grows to fill the stick and survives reboots.
- [ ] Test an OS update end to end: stage, install, reboot, and rollback.

## Next — release confidence

- [ ] Publish a public release and verify unauthenticated update downloads.
- [ ] Test update failure paths: no network, bad checksum, interrupted download.
- [ ] Test power loss during writes and boot-counter rollback.
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
