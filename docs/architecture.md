# SlippiOS architecture

## Disk layout

```text
GPT
├── ESP            512M   systemd-boot, UKIs
├── nix-store-A    1.9G   read-only squashfs store
├── nix-store-B    1.9G   inactive slot
├── root            1G    ext4: /etc, /var, /home
├── MELEE-ROMS      5G    FAT32: user-managed Melee images
└── SLIPPIOS        grow  ext4: /data
```

- `/nix/store` is mounted from the active `nix-store_<version>` partition.
- The matching UKI embeds that label and is stored in the ESP.
- Versions are `<commit-epoch>.<revision>`, where `<revision>` is the first
  seven hexadecimal digits of the commit converted to decimal. systemd-boot
  and systemd-sysupdate order versions with `strverscmp_improved`, which ranks
  a numeric segment above an alphabetic one. A hexadecimal hash starting with
  a letter sorts below one starting with a digit, so the old `<epoch>-<hash>`
  format could make a newer build look older. Legacy hyphen versions sort
  below the decimal form.
- `/data` is a separate partition and is never written by system updates.
- `MELEE-ROMS` is mounted at `/data/isos`; it is the only PC-visible data drive.
- A first-boot service grows `SLIPPIOS` and its ext4 filesystem to fill the USB.

## Boot

1. UEFI runs `systemd-boot` from the ESP.
2. The newest UKI is selected by version.
3. The initrd mounts the matching squashfs store and the root ext4.
4. Stage 2 starts the kiosk, mounts `/data`, and launches the Godot shell.

The unsigned image has no Secure Boot chain and requires Secure Boot off. The
signed image uses a Fedora shim and a one-time MOK enrollment; see
[the roadmap](../ROADMAP.md#later-secure-boot-and-mac-boot).

Each updated UKI carries a three-attempt boot counter. `systemd-bless-boot`
removes it after a successful boot. If the counter reaches zero, systemd-boot
falls back to the previous good UKI.

### Secure Boot

The signed image installs the Fedora 16.1-7 shim as `EFI/BOOT/BOOTX64.EFI`. The
shim is dual-signed by Microsoft UEFI CA 2011 and 2023. It loads `grubx64.efi`,
a SlippiOS-signed systemd-boot, which loads the SlippiOS-signed UKI. The
SlippiOS certificate is enrolled in MOK on first boot; updates use the same key
and do not require re-enrollment. The shim and MokManager are part of the image
and are not updated by systemd-sysupdate.

## System update

1. `slippios os check` fetches `manifest.json` from the latest release.
2. `slippios os update` downloads the store image with zsync, seeded from the
   active `nix-store` partition.
3. If zsync fails, it downloads the whole store image.
4. SHA-256 is checked against the manifest.
5. The matching UKI is downloaded and checked.
6. Both are written to `/data/system/update`.
7. A request file starts `slippios-os-update.service`.
8. The root service runs `systemd-sysupdate`, which writes the inactive store
   partition, updates the ESP, and sets the boot counter. Before recording
   success, the service requires both the `nix-store_<version>` partition and
   a fresh `slippios_<version>+3-0.efi` boot image. A successful no-op, which
   writes nothing, is recorded as a failure instead of a false install.
9. The service writes `/data/system/update/status.json` with the target
   version and the current boot ID, and the shell offers a restart.
10. If the new system boots, `slippios os status` sees the target version
    running and clears the result. If boot assessment rolls back to the
    previous system, the changed boot ID and old version turn the result into
    a failed update instead of a stale "installed" prompt.

Manifest fields:

```json
{
  "version": "1792937434.113649853",
  "tag": "v1.0.0",
  "full_image": {
    "name": "slippios-usb-<version>.img.xz",
    "sha256": "...",
    "size": 2000000000
  },
  "unsigned_image": {
    "name": "slippios-usb-<version>-unsigned.img.xz",
    "sha256": "...",
    "size": 2000000000
  },
  "nix_store": {
    "name": "slippios_<version>.nix-store.raw",
    "sha256": "...",
    "size": 1992294400,
    "zsync": "slippios_<version>.nix-store.raw.zsync"
  },
  "uki": {
    "name": "slippios_<version>.efi",
    "sha256": "...",
    "size": 40100000
  }
}
```

The updater uses the tag to build release URLs. A local base can replace GitHub
for testing:

```json
{ "base": "http://192.168.1.10:8000/" }
```

The zsync index is small; the raw store image is downloaded only when changed
blocks require it. Hosting is free GitHub Releases, limited to 2 GiB per asset.

## Data

```text
/data
├── isos/                    MELEE-ROMS FAT32 mount
├── system/
│   ├── dolphin/            runtime Dolphin builds
│   ├── hardware-id
│   ├── setup.json          first-boot browser wizard state
│   └── update/             staging, status, request files
└── users/default/
    ├── settings.toml
    ├── slippi-user.json
    ├── dolphin/
    ├── replays/
    └── logs/
```

The profile layout is multi-user ready; the shell currently uses `default`.
Game images live on the FAT32 `MELEE-ROMS` partition, not in the ext4 profile.

## Components

- `slippios-shell`: Godot kiosk UI; reads controllers through SDL3 and exits
  before Dolphin claims the GameCube adapter.
- `slippios net --mode setup`: first-boot browser wizard for game and account.
- `slippios-isos.service`: mounts the `MELEE-ROMS` FAT32 partition at
  `/data/isos` and records the result in `/data/system/isos-mount`.
- `slippios-recovery`: tty console if the shell and the Openbox fallback both exit.
- `slippios-session`: terminal session used for hands-on diagnostics.
- `slippios`: Go CLI; human output on a TTY, JSON when piped.
- `slippios-os-update.service`: root installer for staged updates.
- `slippios-os-reboot.path`: root reboot request.
- `slippios-data.service`: mounts or creates `/data` and seeds controller profiles.

The normal session runs the shell without a window manager. The session applies
the saved display mode before starting the shell; the shell opens at the screen
resolution with a borderless window. Dolphin is launched with the same
resolution and borderless fullscreen. A lightweight kiosk window manager
(Openbox or matchbox) is a proposed alternative for fullscreen, focus, and
dialog handling; it needs performance testing before adoption. See the
[developer guide](developers.md#window-manager-experiment).

## Slippi auth

The Go backend performs the same Firebase/GraphQL exchange as the Slippi client.
Passwords stay in memory. The resulting `user.json` token is stored with mode
`0600`, installed into Dolphin, and bound to the machine's DMI UUID. A mismatch
removes the saved login.

## CI and hosting

- **Validate:** pushes to `main`; Go tests, headless Godot, and NixOS
  evaluation.
- Releases are built locally with `scripts/release`: full image + update slot +
  UKI + zsync + manifest, published as a GitHub release.
- The release script signs the default image, extracts the signed UKI from the
  image, and publishes the unsigned image as a fallback.
- Release assets are public only when the repository is public.
