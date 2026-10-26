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
mise run build:image    # USB image, compressed for release
mise run build:image-dev  # uncompressed image, faster local boots
mise run run:image      # boot the local USB image
mise run fetch:image    # download the latest published USB image
mise run test           # Go tests + headless Godot parse
mise run test:shell
mise run go:test
mise run go:build
mise run nix            # shell with Godot, QEMU, zsync, xz, jq
```

The image is written to `result-local-image/` (git-ignored).

## Settings format

`/data/users/default/settings.toml` stays TOML. It is small, human-editable for
recovery from another machine, and has a maintained Go decoder. JSON remains the
shell/CLI protocol; changing the config format would add migration work without
improving the appliance.

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
  Compression only matters for published images, so `build:image-dev` skips it
  (`SLIPPI_DEV=1`). Local boots then use the `.raw` directly.

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
reach other devices.

Delete the qcow2 named after the image hash to reset the guest.

## CI

- **Validate** — pushes to `main`; Go tests and headless Godot.
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

## Repository map

```text
flake.nix                  NixOS configs and app packages
hosts/usb.nix              A/B USB image, sysupdate, data growth
modules/slippios.nix       kiosk, user, data mount, boot hooks
backend/                   Go CLI, session fallback, auth, updates, LAN pages
apps/shell/                Godot UI
pkgs/slippios.nix          Go binaries
pkgs/appimage-runner.nix   Dolphin AppImage sandbox
scripts/build-image        local image build
scripts/run-image          QEMU/OVMF USB runner
scripts/fetch-image        release downloader
```

## Shell binary

`apps/shell` is exported at image-build time to a compiled Godot binary with an
embedded PCK. The USB image contains that binary, not the `.gd`/`.tscn` source
project. `mise run test:shell` still parses the source project.

Export with the template nixpkgs builds (`godot_4.export-template`), not
`godot_4.export-templates-bin`. The prebuilt upstream template expects a generic
desktop graphics stack, cannot initialise OpenGL on NixOS, falls back to OpenGL
ES, and exits. The symptom is the shell dying immediately and Openbox falling
back to the terminal session.

## Dolphin AppImage

`pkgs/appimage-runner.nix` is an FHS environment built from
`appimageTools.defaultFhsEnvArgs`. Slippi's AppImage downloads at runtime and
expects a normal desktop set of system libraries (`harfbuzz`, `gmp`,
`libgpg-error`, `libcom_err` and friends). Keep the upstream environment rather
than a hand-picked package list; the upstream AppImage changes which libraries
it leaves to the system between releases.

## Melee ROMs

The USB image creates a 5 GiB FAT32 partition labelled `MELEE-ROMS`. It is
mounted at `/data/isos` after `slippi-data.service` creates the directory. The
partition is fixed-size; the ext4 `/data` partition after it grows to fill the
stick. Images keep their original filenames. SlippiOS validates only the
GameCube game ID, not a hash.

## Boot and game handoff

The shell shows a short boot splash on cold boot and waits for A/Enter. It then
exits with code 42. The `slippios-shell` wrapper runs `slippios play` only after
Godot has exited, then starts Godot again with `SLIPPIOS_AUTOPLAY=0` and
`SLIPPIOS_SKIP_SPLASH=1`. This releases the GameCube adapter for Dolphin instead
of leaving it claimed by SDL, and avoids replaying the splash after a game. The
same loop reopens the shell when Dolphin is closed or crashes.

## Wi-Fi

```sh
slippios wifi scan
slippios wifi status
slippios wifi connect --ssid "Network" --password "secret"
```

The shell wraps this in **Settings → Wi-Fi network**. `nmcli` comes from the
NetworkManager package; the `player` account is in the `networkmanager` group.

## System updates

```sh
slippios os check
slippios os status
slippios os update
slippios os reboot
```

`osupdate.go` fetches `manifest.json`, zsyncs the store image against the active
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

The first-boot setup page is a two-step browser wizard served by `slippios net --mode
setup`: game image, then Slippi account or offline. The shell shows only the QR
and progress; the browser writes the normal `settings.toml` and login files.

The shell reads:

- `/data/system/net-status.json` — server URL or startup error.
- `/data/system/setup.json` — `game_ready`, `account_ready`, `done`, and error.
- `/data/system/net-upload.json` — written after a validated upload.

When `setup.json` reports `done`, the shell stops the server and launches Melee.
Port 8080 is retried for five seconds. `--mode iso` and `--mode login` remain for
the older single-purpose pages used by tests/account flows. The wizard works
from any LAN device, not just a phone.

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

See the [roadmap](../ROADMAP.md). Current priority is real-hardware USB
validation; install-to-internal-disk is scheduled after that.
