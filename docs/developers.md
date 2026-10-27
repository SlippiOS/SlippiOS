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
mise run dev:cli -- status   # CLI against that sandbox
mise run dev:ssh        # SSH into the running dev VM
mise run dev:push       # build CLI + shell and push them into the dev VM
mise run run:image      # boot the local image (virtio disk; fast)
mise run stop:image     # clean ACPI power-off of that VM
mise run fetch:image    # download the latest published USB image
mise run test           # Go tests + headless Godot parse
mise run nix            # shell with Godot, QEMU, zsync, xz, jq
```

The image is written to `result-local-image/` (git-ignored).

## Settings format

`/data/users/default/settings.toml` stays TOML. It is small, human-editable for
recovery from another machine, and has a maintained Go decoder. JSON remains the
shell/CLI protocol; changing the config format would add migration work without
improving the system.

## Build notes

- With system Nix: `scripts/build-image` calls `nix build`.
- Without Nix: it downloads nix-portable to
  `.cache/nix-portable-bin` and keeps its store in `.cache/nix`.
- nix-portable uses `--option sandbox false`; `systemd-repart` needs nested
  user namespaces.
- Build artifacts need about 12 GB free. `TMPDIR` defaults to
  `.cache/build-tmp`, not `/tmp`.
- First build downloads the closure. Later builds reuse the project-local Nix
  store in `.cache/nix`.
- Full image and update slot must stay below GitHub's 2 GiB per-asset limit.
  Compression only matters for published images, so `build:image-dev` skips it
  (`SLIPPI_DEV=1`). Local boots then use the `.raw` directly.
- For local update tests, build the image and the UKI in one evaluation:
  `./scripts/nix build ...build.image ...build.uki`. A UKI rebuilt after
  further edits embeds initrd store paths that the earlier store image does not
  contain, so the installed system drops to emergency mode. CI is safe because
  it builds from one clean checkout.
- Image versions are `<commit-epoch>.<revision>`, with the first seven
  hexadecimal digits of the commit converted to decimal. systemd-boot and
  systemd-sysupdate use `strverscmp_improved`, so a revision must be a numeric
  segment. A hexadecimal hash that starts with a letter sorts below one that
  starts with a digit. Do not publish builds using the old `<epoch>-<hash>`
  format.

## Local cache and cleanup

QEMU overlays, monitor sockets, build scratch, downloaded test ISOs, the
nix-portable binary and its Nix store all live in `.cache/` in the repository,
which is gitignored. The physical store is `.cache/nix`; nix-portable's
hardcoded `.nix-portable` entry is only a compatibility symlink to it.

```sh
mise run clean         # overlays, ISOs, build scratch + GC unreachable Nix paths
mise run clean:hard    # also remove the project Nix store (full re-download next build)
```

## Dev VM loop

Build a dev image once with `mise run build:image-dev` and boot it. That image
enables SSH on port 2222 (`player` / `slippios`) and creates `/data/dev`.
After that, do not rebuild for application changes:

```sh
mise run dev:push   # Go CLI + Godot export -> /data/dev, restart the session
mise run dev:ssh    # shell/logs on the VM
```

The shell wrapper prefers `/data/dev/slippios` and `/data/dev/slippios-shell`
when they exist, so a pushed build takes effect after the session restart.
Because the shell export goes through the flake, new shell files must be staged
(`git add apps/shell`) before `dev:push`; untracked files are not part of the
Nix source tree. Image changes (NixOS modules, packages) still need
`build:image-dev`.

`scripts/dev` builds a throwaway data tree with a tiny fake Melee image
(`GALE01` header, not a playable disc) and a `settings.toml`. The CLI, ISO
checks, and Dolphin runner can be exercised there without an image build.

`mise run test` starts the shell headlessly with `SLIPPIOS_PARSE_ONLY=1`, so
it only checks GDScript/project parsing; it does not need the CLI.

## VM

```sh
mise run run:image
```

`run-image` accepts `.raw` or `.raw.xz`, creates a 16 GB qcow2 overlay in
`.cache/`, and boots it under OVMF with a virtio disk by default. Set
`SLIPPI_DISK=usb` to exercise the USB-storage path; USB emulation adds about
20 seconds to a boot. It also exposes a QEMU monitor socket at
`.cache/<image>.sock`.

Closing the QEMU window is a hard power cut. Power off from the guest (**Quit →
Power off**) or run `mise run stop:image`, which sends an ACPI power button over
the monitor socket.

```sh
SLIPPI_DISPLAY=none SLIPPI_SERIAL=/tmp/slippios-serial.log mise run run:image
SLIPPI_DISK_SIZE=32G mise run run:image
SLIPPI_SECURE_BOOT=1 SLIPPI_DISPLAY=none mise run run:image
```

QEMU has no Wi-Fi hardware, so `wpa_supplicant.service` stays in its "wait for a
wireless interface" loop and eventually reports failure. That is expected; the
boot console no longer shows service status lines.

Host port 8080 forwards to the guest. Use
`http://localhost:8080/<token>/` from the host. QEMU user networking does not
reach other devices.

Delete the qcow2 named after the image hash to reset the guest.

## CI

- **Validate:** pushes to `main`; Go tests and headless Godot.
- **Build Image:** `v*`/`os-*` tags and manual dispatch only.

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
hosts/usb.nix              USB image composition
hosts/usb/                 boot, data, image, services, and update modules
modules/slippios.nix       base module composition
modules/slippios/          theme, session, data, and system modules
backend/                   Go CLI, session fallback, auth, updates, LAN pages
apps/shell/                Godot UI
  main.gd                  app state, boot flow, screen routing, update orchestration
  screens/                 one screen module per flow
  services/                CLI, browser setup, update job, UI builders, status, actions
  widgets/                 shared controls
pkgs/slippios.nix          Go binaries
pkgs/appimage-runner.nix   Dolphin AppImage sandbox
scripts/build-image        local image build
scripts/run-image          QEMU/OVMF USB runner
scripts/fetch-image        release downloader
scripts/sign-image         Secure Boot signing for an image
secure-boot/               shim, enrollment note, and signing docs
```

## Shell binary

`apps/shell` is exported at image-build time to a compiled Godot binary with an
embedded PCK. The USB image contains that binary, not the `.gd`/`.tscn` source
project. `mise run test` still parses the source project.

Export with the template nixpkgs builds (`godot_4.export-template`), not
`godot_4.export-templates-bin`. The prebuilt upstream template expects a generic
desktop graphics stack, cannot initialise OpenGL on NixOS, falls back to OpenGL
ES, and exits. The symptom is the shell dying immediately and Openbox falling
back to the terminal session.

## Dolphin configuration

The shell edits a curated subset of Dolphin settings in settings.toml and maps
them to `-C System.Section.Key=Value` at launch. The important values:

- `Dolphin` is the main INI; `Graphics` is the GFX INI (not `GFX`).
- `Dolphin.Core.SIDevice0..3`: 12 is the Wii U adapter, 6 a standard pad,
  7 the keyboard.
- `Graphics.Settings.InternalResolution` is the EFB scale (1 = native).
- `Graphics.Settings.AspectRatio`: 1 widescreen, 2 4:3, 3 Melee, 4 stretch.
- `Graphics.Settings.MSAA`: 1 off, then 2/4/8 samples.
- Enums are written as integers; booleans as `True`/`False`.

**Expert mode → Dolphin settings window** still opens the real Dolphin GUI for
anything the curated list does not cover (controller bindings, profiles).

## Boot splash

Plymouth uses a charge-style `two-step` theme: `modules/slippios/theme.nix`
renders 24 throbber frames from the mark with a sine opacity, and the C throbber
plugin animates them over two seconds. This avoids the script plugin's refresh
callback, which did not animate in QEMU's frame-buffer fallback.

Plymouth frames are a fixed 324 px high (30% of 1080p), because two-step themes
cannot scale per display. The Godot splash does scale: it sizes the mark to 30%
of the screen height and pulses it.

The initrd loads `virtio_gpu` and `bochs` so Plymouth has a framebuffer in a VM
before stage 2. Physical machines use the firmware framebuffer through the
built-in `simpledrm` until the native driver loads. NixOS hardcodes
`DeviceTimeout=8` in `plymouthd.conf`, which leaves console text on screen until
the wait expires; the theme module replaces that file with `DeviceTimeout=1` in
both the initrd and stage 2. With `loglevel=0` and `ShowStatus=no`, the splash
covers the console immediately and no TTY lines leak through.

When the session starts, the X session script paints
`apps/shell/assets/slippi-boot.png` on the X root window before Godot starts, so
the handoff shows the same mark instead of a blank screen. There is no window
manager in the normal path: the Godot shell is the X session, so it has no
decorations and owns the whole screen. Openbox is only started if the shell
exits, as a desktop for diagnosis.

`assets/marks/` holds the temporary Slippi mark and how to regenerate the
raster assets; replace it when the SlippiOS mark exists.

## Theming

`theme/default.nix` is the single source of truth for the background, surface,
accent and text colors, plus the splash mark size and pulse. Nix uses it for the
Plymouth theme, LightDM's greeter background, and the X root color. The module
also installs it as `/etc/slippios/theme.json`; the `SlippiTheme` Godot autoload
reads that file at startup and applies the clear color and UI colors. The
compiled fallback in `theme.gd` keeps the dev sandbox usable without the file.
Changing a color therefore needs an image rebuild for Plymouth/LightDM and a
shell push for the Godot UI.

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
stick. The initrd grows it once (parted, `e2fsck -f`, `resize2fs`) and skips the
work on later boots when the partition already reaches the end of the disk.
Images keep their original filenames. SlippiOS validates only the GameCube game
ID, not a hash.

## Boot and game handoff

Godot never launches Dolphin itself. It exits with a code and the
`slippios-shell` wrapper acts on it:

| Exit | Wrapper action |
| --- | --- |
| 42 | `slippios play`, then reopen the shell |
| 43 | `slippios dolphin config` (Dolphin settings window), then reopen |
| 44 | xterm running a login shell, then reopen |
| 45 | xterm running nano on settings.toml, then reopen |
| anything else | exit; the X session starts Openbox |

This releases the GameCube adapter for Dolphin instead of leaving it claimed by
SDL. The wrapper reopens Godot with `SLIPPIOS_AUTOPLAY=0` and
`SLIPPIOS_SKIP_SPLASH=1`, so the splash is cold-boot only.

The normal X session runs the shell directly, without a window manager. On a
crash the session script starts Openbox so the machine can be diagnosed. If
Openbox also exits, it writes `/data/system/shell-failed`;
`slippios-recovery.service` then conflicts with the display manager and takes
over tty1 with a recovery console: last shell log lines, failed units, and Try
again / Power off / Reboot.

## Secure Boot

The signed image uses the Fedora 16.1-7 shim and a SlippiOS MOK key. See
[secure-boot/README.md](../secure-boot/README.md) for the chain and limits.

Sign a local image:

```sh
SLIPPIOS_SB_KEY=/path/to/slippios.key \
SLIPPIOS_SB_CERT=/path/to/slippios.crt \
  mise run sign:image -- result-local-image/slippios_<version>.raw signed.raw
```

The script downloads the pinned shim RPM, verifies its SHA-256 and both
Microsoft signatures, signs systemd-boot and the UKI, and installs the shim,
MokManager, and `slippios.cer`. Keep the private key offline.

The release workflow still publishes the unsigned image. Signing must be
integrated before a signed release asset is published, and the manifest must
use the signed UKI.

Test with OVMF Secure Boot:

```sh
SLIPPI_SECURE_BOOT=1 SLIPPI_DISPLAY=none mise run run:image
```

The first boot shows the shim verification error and MokManager. Enroll
`slippios.cer`; later boots and updates use the MOK key.

## Display and audio

```sh
slippios display apply        # apply resolution, refresh, and audio sink
slippios audio sinks          # JSON list of PulseAudio outputs
```

`backend/internal/desktop` is shared by the shell (through these commands) and
the terminal session, so both apply the same settings.toml values.
`slippios display modes` reads the driver's xrandr mode list (which is derived
from EDID) plus the EDID monitor name; `slippios display mode` and
`slippios display custom` switch or create modes. The shell applies the saved
mode at boot, and re-adds a custom CVT mode because X forgets runtime modes.

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

The updater fetches `manifest.json`, zsyncs the store image against the active
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

The first-boot setup page is a two-step browser wizard served by
`slippios net --mode setup`: game image, then Slippi account or offline. The
shell shows only the QR and progress; the browser writes the normal
`settings.toml` and login files.

The shell reads:

- `/data/system/net-status.json`: server URL or startup error.
- `/data/system/setup.json`: `game_ready`, `account_ready`, `done`, and error.
- `/data/system/net-upload.json`: written after a validated upload.

When `setup.json` reports `done`, the shell stops the server and launches Melee.
Port 8080 is retried for five seconds. `--mode iso` and `--mode login` remain for
the older single-purpose pages used by tests and account flows. The wizard works
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
validation; install-to-internal-disk is scheduled after that. The signed Secure
Boot image is a verified prototype; release signing and Mac boot are still
open.
