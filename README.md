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
├── nix-store-A    1.8G   read-only squashfs system store
├── nix-store-B    1.8G   empty until the first update
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
scripts/      host tooling (fetch, flash, run)
docs/         notes
```

Runtime pieces:

- `slippios-shell` — fullscreen Godot menu (the console UI)
- `slippios-session` — headless fallback and boot flow (Python)
- `slippios` — state/actions consumed by the shell, human-first with
  automatic JSON when piped

## Build

Pushes to `main` run two GitHub Actions workflows:

- **Build Image** builds `nixosConfigurations.slippios-usb`, publishes the
  full `.img.xz`, the A/B update slot, its `.zsync`, the UKI, and
  `manifest.json` as a GitHub release.
- **Build ISO** builds the development/recovery ISO and uploads it to the
  transfer.sh mirror.

Local builds need Nix once:

```sh
sh <(curl -L https://nixos.org/nix/install) --daemon
nix build .#nixosConfigurations.slippios-usb.config.system.build.image
```

`boot.loader.timeout` is 1 second, so the image boots automatically; GRUB treats
`0` as “wait forever”.

## Test in a VM

```sh
./scripts/fetch-image
```

Downloads the newest successful USB image, verifies it, creates a 16 GB qcow2
overlay, and boots it under OVMF (UEFI). The overlay leaves room for the data
partition to grow, exactly as it would on a larger USB stick. `--offline`
reuses the cache and `--no-run` only downloads.

For the development ISO:

```sh
./scripts/fetch-iso --restart
```

Downloads the newest successful ISO into `~/.cache/slippios/`, verifies its
SHA-256, installs it to `/var/lib/libvirt/images/slippios.iso`, then cold-boots
the domain. Use `--offline` to reuse a verified cached ISO, `--dry-run` to
preview, and `--yes` to skip the prompt when the newest run has not succeeded.

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

The image is set up for in-place updates: `systemd-sysupdate` is configured to
install a staged store image and UKI from `/data/system/update`, and the CI
release contains the store image, its `.zsync`, the UKI, and `manifest.json`.
The remaining on-device piece is the updater that fetches the manifest, zsyncs
the new store image against the active slot, verifies it, and triggers
`updatectl`. That is the next shell/CLI milestone.

System updates never write to `/data`. Dolphin updates remain a separate,
smaller runtime download under `/data/system/dolphin`.

## slippios

```sh
slippios status
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
2. Wire the shell's update button to the zsync/sysupdate flow and add a
   progress/reboot screen.
3. Adapter helper: expose the GameCube adapter as a virtual SDL pad, then
   release it before Dolphin starts.
4. Wi-Fi setup in the shell.
5. Move the flake from NixOS 25.05 to a supported release; 25.05 is past its
   end-of-support date.
