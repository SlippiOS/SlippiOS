# SlippiOS developer guide

Disk layout and update protocol: [architecture](architecture.md). Player behavior: [users](users.md).

## Quick start

```sh
mise trust
mise install
mise run build:image
mise run run:image
mise run test
```

No Docker or system Nix required.

## Tasks

```sh
mise run build:image    # USB image
mise run build:iso      # development ISO (not published)
mise run run:image      # boot the local USB image
mise run run:iso        # boot the local ISO
mise run fetch:image    # download the latest published USB image
mise run test           # Python byte-compile + headless Godot parse
mise run test:python
mise run test:shell
mise run nix            # shell with Godot, QEMU, zsync, xz, jq
```

The image is written to `result-local-image/` (git-ignored).

## Build notes

- With system Nix: `scripts/build-image` calls `nix build`.
- Without Nix: it downloads nix-portable to
  `~/.cache/slippios/nix-portable` and runs it under bubblewrap.
- nix-portable uses `--option sandbox false`; `systemd-repart` needs nested
  user namespaces.
- Build artifacts need about 12 GB free. `TMPDIR` defaults to
  `~/.cache/slippios/build-tmp`, not `/tmp`.
- First build downloads the closure. Later builds reuse `~/.nix-portable`.
- Full image and update slot must stay below GitHub's 2 GiB per-asset limit.

`test:shell` starts the shell headlessly. JSON parse errors from the missing
`slippios` CLI are expected; GDScript parse errors are not.

## VM

```sh
mise run run:image
```

`run-image` accepts `.raw` or `.raw.xz`, creates a 16 GB qcow2 overlay in
`~/.cache/slippios`, and boots it as a USB stick under OVMF.

```sh
SLIPPI_DISPLAY=none SLIPPI_SERIAL=/tmp/slippios-serial.log mise run run:image
SLIPPI_DISK_SIZE=32G mise run run:image
```

Host port 8080 forwards to the guest. Use
`http://localhost:8080/<token>/` from the host. QEMU user networking does not
reach phones.

Delete the qcow2 named after the image hash to reset the guest.

## Development ISO

```sh
mise run build:iso
mise run run:iso
```

The ISO is local/recovery only. `scripts/flash-usb` can add a `SLIPPIOS`
partition to a physical stick; it cannot update in place.

## CI

- **Validate** — pushes to `main`; Python byte-compile and headless Godot.
- **Build Image** — `v*`/`os-*` tags and manual dispatch only.

Release assets:

```text
slippios-usb-<version>.img.xz
slippios_<version>.nix-store.raw
slippios_<version>.nix-store.raw.zsync
slippios_<version>.efi
manifest.json
SHA256SUMS
```

Tag a release:

```sh
git tag -a v1.0.0 -m "SlippiOS 1.0.0"
git push origin v1.0.0
```

The ISO is not built or published by CI.

## Repository map

```text
flake.nix                  NixOS configs and app packages
hosts/live.nix             development ISO
hosts/usb.nix              A/B USB image, sysupdate, data growth
modules/slippios.nix       kiosk, user, data mount, boot hooks
apps/session/              settings, auth, launch, Dolphin/OS updates
apps/shell/                Godot UI
apps/cli/                  slippios CLI and LAN pages
pkgs/appimage-runner.nix   Dolphin AppImage sandbox
scripts/build-image        local image build
scripts/run-image          QEMU/OVMF USB runner
scripts/build-iso          local ISO build
scripts/fetch-image        release downloader
scripts/flash-usb          legacy ISO + data partition flasher
```

## System updates

```sh
slippios os check
slippios os status
slippios os update
slippios os reboot
```

`osupdate.py` fetches `manifest.json`, zsyncs the store image against the active
`/nix/store` device, verifies SHA-256, downloads the UKI, and writes a request
file. A root oneshot runs `systemd-sysupdate`. The shell polls
`/data/system/update/status.json` and offers a restart.

Local update source:

```json
{ "base": "http://192.168.1.10:8000/" }
```

Write this to `/data/system/update-source.json`. The HTTP server must support
Range requests for zsync.

## Network pages

The shell reads:

- `/data/system/net-status.json` — server URL or startup error.
- `/data/system/net-upload.json` — written after a complete validated ISO.

Port 8080 is retried for five seconds.

## Debugging

```sh
SLIPPI_SERIAL=/tmp/slippios-serial.log mise run run:image
```

- Boot entries: `bootctl list`.
- Data: `SLIPPIOS` → `/data`; system store `nix-store_<version>`.
- Update units: `slippios-os-update.service`, `slippios-os-update.path`.
- Reboot unit: `slippios-os-reboot.path`.
- Private repository release downloads return 404 unless authenticated; use a
  local `update-source.json` or publish the repository.

## Known work

- GameCube-adapter virtual pad.
- Wi-Fi setup in the shell.
- Backup/restore for larger USB migration.
- End-to-end update validation on real hardware.
