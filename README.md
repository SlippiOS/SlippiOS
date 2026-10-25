# SlippiOS

A console-style NixOS appliance for Slippi Melee. It boots straight into a
fullscreen Godot shell and keeps games, settings, the Slippi login, Dolphin,
and replays on a persistent data partition.

## Download and flash

The product artifact is `slippios-usb-<version>.img.xz` from the latest
[GitHub release](https://github.com/SlippiOS/SlippiOS/releases/latest). Free
GitHub Releases hosting is deliberate: no paid server, no expiring transfer
link.

```sh
# Etcher: select the .img.xz and flash it.
# Rufus: select the .img.xz and use DD mode.
sudo dd if=slippios-usb-<version>.img.xz of=/dev/sdX bs=4M status=progress conv=fsync
```

Boot with UEFI. Secure Boot must be off. The image is self-contained; there is
no installer, no partitioning step, and no scripts to run. The data partition
grows to fill the stick on first boot.

The ISO build remains available for VM testing and recovery. A plain ISO flash
is ephemeral unless a `SLIPPIOS` partition is attached.

## Disk layout

```text
GPT
├── ESP            512M   systemd-boot + per-slot UKIs
├── nix-store-A    1.9G   read-only squashfs system store
├── nix-store-B    1.9G   empty until the first update
├── root            1G    ext4: /etc, /var, /home
└── SLIPPIOS        grow  ext4: /data
```

The system store is immutable and A/B. An update writes the inactive store and
its matching UKI, then reboots; if the new system fails to boot three times,
systemd-boot falls back to the previous good UKI. User data on `/data` is never
touched by a system update.

Updates use [`zsync`](https://zsync.moria.org.uk/) against the running store
partition, so only changed blocks are downloaded. `systemd-sysupdate` installs
the staged image and manages the slot labels and boot counting. The update
manifest and assets live on GitHub Releases; no chunk server or paid object
storage is required.

From the console: **Update → Install system update**. The downloader runs as
`player`, writes the verified raw store image and UKI to
`/data/system/update`, then asks a root oneshot to install them. Reboot when
prompted. The previous slot remains available if the new system fails.

The ISO is a local development/recovery artifact only. CI does not publish it.

## Layout

```text
hosts/        NixOS configurations (live ISO and USB image)
modules/      shared NixOS modules
pkgs/         Nix package definitions
apps/
  session/    Python boot/session logic (settings, auth, updates, launch)
  shell/      Godot UI
  cli/        slippios, the JSON control CLI
assets/       controller profiles and branding
scripts/      host tooling (build, fetch, flash, run)
docs/         notes
mise.toml     local development tasks
```

Runtime pieces:

- `slippios-shell` — fullscreen Godot menu (the console UI)
- `slippios-session` — headless fallback and boot flow (Python)
- `slippios` — state/actions consumed by the shell, human-first with
  automatic JSON when piped

## Development environment

You do not need Docker or a system-wide Nix install.

```sh
mise trust            # one-time, if mise asks
mise install          # Python 3.13 and jq
mise run build:image  # system Nix, or nix-portable downloaded automatically
mise run run:image    # boot the local image under OVMF/QEMU
mise run test         # Python byte-compile + headless Godot parse
```

`scripts/build-image` uses system Nix when available and otherwise downloads
[nix-portable](https://github.com/DavHau/nix-portable). nix-portable runs under
bubblewrap and keeps the Nix store in `~/.nix-portable`. The build needs about
12 GB of free space for the raw image and a real-disk `TMPDIR`; the script uses
`~/.cache/slippios/build-tmp`.

The first build downloads the closure and takes a while; later builds reuse the
local Nix store. Local builds are much faster than the 2-vCPU private CI runner.

`mise run build:iso` builds the development ISO locally. It is never published.

## CI and releases

- **Validate** runs on pushes: Python byte-compile and a headless Godot parse.
- **Build Image** runs on `v*`/`os-*` tags and manual dispatch only. It builds
  `nixosConfigurations.slippios-usb`, publishes the full `.img.xz`, the A/B
  update slot, its `.zsync`, the UKI, and `manifest.json` as a GitHub release.
- The ISO is not built or published by CI.

## Test in a VM

```sh
mise run fetch:image    # download the newest published release
mise run run:image      # boot the newest local build
```

`run-image` accepts a `.raw` or `.raw.xz`, creates a writable 16 GB qcow2
overlay, and boots under OVMF with the image attached as a USB stick. This lets
the `SLIPPIOS` partition grow exactly as it would on a larger USB stick.

For the development ISO:

```sh
mise run build:iso
mise run run:iso
```

## Flashing a USB

Use the `.img.xz` from the latest release as described above. The image already
contains the ESP, A/B system store, root, and growable `SLIPPIOS` data
partition.

`scripts/flash-usb` remains for the development ISO: it writes the ISO and adds
an ext4 partition labelled `SLIPPIOS` in the remaining space. That makes a live
ISO persistent, but it is not the product image and cannot update in place.

## Data and settings

Everything user-owned lives on a partition labelled `SLIPPIOS`, mounted at
`/data`. Without it the live ISO uses a temporary `/data`, so nothing persists.

```text
/data
├── isos/                    shared game images
├── system/                  Dolphin builds, hardware binding, updates
└── users/default/           per-user profile
    ├── settings.toml
    ├── slippi-user.json
    ├── dolphin/
    ├── replays/
    └── logs/
```

```toml
schema_version = 1

[game]
iso = "melee.iso"          # relative to /data/isos

[display]
refresh_hz = 120
resolution = "auto"

[audio]
sink = "auto"

[network]
ssid = "Home"
psk = "plaintext-for-now"

[slippi]
user_json = "slippi-user.json"
```

The session opens the configured game, prompts for Slippi credentials on first
run, downloads the newest Dolphin release, and verifies it. Passwords are held
in memory only; the saved `user.json` is bound to the machine's DMI UUID and is
removed if the cartridge moves to different hardware.

## Controllers

The shell reads standard SDL gamepads (Glyph, Xbox, DualSense) and the raw
GameCube adapter in parallel. Linux controller profiles are seeded into
`/data/users/default/dolphin/Config/Profiles/GCPad/` on first boot. See
[`docs/controllers.md`](docs/controllers.md).

## Updates

`slippios os check`, `slippios os update`, and the shell's Update screen fetch
`manifest.json` from the latest GitHub release. The updater zsyncs the new
store image against the active `nix-store` partition, falls back to a full
download if needed, verifies SHA-256, downloads the matching UKI, and stages
both in `/data/system/update`. A root oneshot then runs `systemd-sysupdate`;
the shell offers a restart. A failed boot is rolled back by boot counting.

System updates never write to `/data`. Dolphin updates remain a separate,
smaller runtime download under `/data/system/dolphin`.

For development, a private release can be served from a local HTTP server by
writing `/data/system/update-source.json`:

```json
{ "base": "http://192.168.1.10:8000/" }
```

## slippios

```sh
slippios status
slippios os check
slippios os update
slippios update --check
slippios login --stdin
slippios play
slippios adapter
```

Human-readable on a TTY, JSON when piped; `--json`/`--human` override. The
shell calls it and parses stdout.

## Roadmap

1. Shell polish: boot animation and richer settings screens (onboarding and
   account sign-in are implemented).
2. Measure real zsync deltas and add update progress/resume polish.
3. Adapter helper: expose the GameCube adapter as a virtual SDL pad, then
   release it before Dolphin starts.
4. Wi-Fi setup in the shell.
5. Optional backup/restore for migrating to a larger USB or recovering data.
6. Move the image build to release-only CI and keep local builds fast.
