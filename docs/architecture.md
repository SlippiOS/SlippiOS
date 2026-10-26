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
- `/data` is a separate partition and is never written by system updates.
- `MELEE-ROMS` is mounted at `/data/isos`; it is the only PC-visible data drive.
- A first-boot service grows `SLIPPIOS` and its ext4 filesystem to fill the USB.

## Boot

1. UEFI runs `systemd-boot` from the ESP.
2. The newest UKI is selected by version.
3. The initrd mounts the matching squashfs store and the root ext4.
4. Stage 2 starts the kiosk, mounts `/data`, and launches the Godot shell.

Each updated UKI carries a three-attempt boot counter. `systemd-bless-boot`
removes it after a successful boot. If the counter reaches zero, systemd-boot
falls back to the previous good UKI.

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
   partition, updates the ESP, sets the boot counter, and writes
   `/data/system/update/status.json`.
9. The shell offers a restart.

Manifest fields:

```json
{
  "version": "1792937434-05a864f",
  "tag": "v1.0.0",
  "nix_store": {
    "name": "slippios_<version>.nix-store.raw",
    "sha256": "…",
    "size": 1992294400,
    "zsync": "slippios_<version>.nix-store.raw.zsync"
  },
  "uki": {
    "name": "slippios_<version>.efi",
    "sha256": "…",
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

- `slippios-shell` — Godot kiosk UI; reads controllers through SDL3 and exits
  before Dolphin claims the GameCube adapter.
- `slippios net --mode setup` — first-boot browser wizard for game and account.
- `data-isos.mount` — mounts `MELEE-ROMS` FAT32 at `/data/isos`.
- `slippios-recovery` — tty console after an unexpected shell exit.
- `slippios-session` — terminal session used for hands-on diagnostics.
- `slippios` — Go CLI; human output on a TTY, JSON when piped.
- `slippios-os-update.service` — root installer for staged updates.
- `slippios-os-reboot.path` — root reboot request.
- `slippios-data.service` — mounts/creates `/data`, seeds controller profiles.

## Slippi auth

The Go backend performs the same Firebase/GraphQL exchange as the Slippi client.
Passwords stay in memory. The resulting `user.json` token is stored with mode
`0600`, installed into Dolphin, and bound to the machine's DMI UUID. A mismatch
removes the saved login.

## CI and hosting

- **Validate** on pushes: Go tests and headless Godot.
- **Build Image** on tags/manual: full image + update slot + UKI + zsync +
  manifest, published as a GitHub release.
- Release assets are public only when the repository is public.
