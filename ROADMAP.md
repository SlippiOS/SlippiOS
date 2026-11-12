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

- [x] Switched to semver versions (`0.3.0`); devices on the old epoch format
      require a one-time reflash.
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

- [x] Add local release signing with `mise run release`; it signs the image and
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

## Window manager

SlippiOS runs a minimal Openbox before the shell. Its config removes
decorations and keybindings and limits the root menu to the shell, terminal,
text editor, Dolphin settings, reboot, and power off. Openbox handles
`_NET_WM_STATE_FULLSCREEN`, focus, and dialogs; the shell and Dolphin still
size themselves explicitly as a fallback.

- [x] Prototype a minimal Openbox session.
- [x] Test on real hardware and measure performance.
- [x] Test the shell at 640x480 and 1280x720.
- [x] Test runtime resolution changes.
- [x] Test Dolphin fullscreen, settings window, and file dialogs.
- [x] Measure input latency and game frame pacing.
- [x] Check focus stealing, accidental close, and fallback behavior.
- [x] Adopt Openbox as the default session.

The no-WM session was the original design. Openbox was adopted after hardware
testing on the EM680; it was not clearly worse, and it provides a usable
fallback desktop.

## Later: hardware performance check

SlippiOS runs on arbitrary PCs, so the best build and settings can vary. A
short, deterministic check can recommend them instead of guessing.

- [ ] Add a Settings and optional onboarding performance check. Run a bundled
      replay through the playback build at the configured resolution, report
      effective FPS and frame consistency, and recommend resolution or settings
      changes when it cannot hold 60 fps.
- [ ] Benchmark stable and beta on the same machine once beta build switching
      and playback are available, and recommend the faster build.
- [ ] Evaluate a bot/save-state scenario for live-match performance after the
      replay-based check ships. Stable's netplay build has no CLI save-state
      option and mainline playback is still WIP, so this is not a first step.
- [ ] Consider extracting the measurement engine into a standalone tool for
      non-SlippiOS users if there is demand.

## Next: release distribution

- [x] Create the SourceForge project (`slippi-os`).
- [x] Upload the current signed/unsigned images plus `LICENSE`, `NOTICE`,
      `THIRD_PARTY_LICENSES.md`, `SOURCE`, and `LICENSES.tar.gz`.
- [x] Delete the full images from R2; keep only the manifest, nix-store raw,
      zsync, UKI, and `SHA256SUMS.update`.
- [ ] Archive superseded full images on the Internet Archive.
- [x] Add the license/source offer to the release page and the USB package.
- [ ] Rebuild the image so `/etc/slippios/licenses` is present in the next
      release.

## Later: commercial distribution

Selling pre-flashed USBs is allowed under GPL-3.0-or-later; the software stays
free. This is a future business/legal track, not a release gate.

- [ ] Decide the business model: USB hardware, testing, support, updates, and
      warranty. The software remains GPL-3.0-or-later.
- [ ] Verify the Slippi playback source repository with the Slippi project;
      stable and beta sources are available.
- [ ] Replace the temporary Slippi mark with an original SlippiOS mark or get
      permission; confirm the "SlippiOS" trademark policy.
- [ ] Review the Slippi online service terms for commercial use.
- [ ] Confirm firmware redistribution terms and include the notices.
- [ ] Define the source-offer process for physical USBs: include `LICENSE`,
      `NOTICE`, `THIRD_PARTY_LICENSES.md`, `SOURCE`, and the license texts;
      keep the exact source available for at least three years.
- [ ] Review consumer warranty, product liability, tax, shipping, returns, and
      export-control requirements with a lawyer.
- [ ] Review GPLv3 installation-information obligations if selling a locked
      console/PC; a USB-only product is simpler.
- [ ] Decide whether to use a CLA/DCO for future dual-licensing of SlippiOS's
      own code; bundled GPL components cannot be relicensed.
- [ ] Keep the Melee ISO out of the product; users supply their own game.
- [ ] Build a repeatable USB production and verification script with a QA
      checklist.

## Not planned yet

- Dual-boot alongside Windows or another Linux install.
- Encrypted internal disks.
- Migrating an existing USB `/data` into an internal install.
- Mod-specific management UI beyond selecting and launching Melee images.
- A separate installer ISO; the USB image is the installer medium.
