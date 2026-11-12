# SlippiOS developer guide

Disk layout and update protocol: [architecture](architecture.md). User behavior: [users](users.md).

## Quick start

```sh
mise trust
mise install
mise run build:image
mise run run:image
mise run test
```

No Docker or system Nix required.

### Prerequisites

- Linux x86_64 with `mise`
- `curl` to download nix-portable when system Nix is absent
- `xz` for compressed images, releases, and downloads
- `bubblewrap` for image builds through nix-portable; `run:image` and
  `dev:push` fall back to proot when it is missing
- Optional: `qemu` and UEFI firmware (otherwise `run:image` provisions them
  through Nix)

```sh
# Fedora
sudo dnf install curl xz bubblewrap qemu
# Debian/Ubuntu
sudo apt install curl xz-utils bubblewrap qemu-system-x86 ovmf
```

## Tasks

```sh
mise run build:image    # USB image for release
mise run build:image-dev  # uncompressed image, faster local boots
mise run release        # build, sign, and create a draft release
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
- `SLIPPI_CONFIG=slippios-usb-minimal` (or `mise run build:image-minimal`)
  builds the minimal image. `hosts/minimal.nix` removes the seeded app payloads
  and shrinks the initial `/data`; the first boot runs
  `slippios apps install` to download the stable, beta, and playback builds.
- Image versions are semver (`0.3.0`). `VERSION` is the source of truth.
  Release builds use it; dev builds use the release version plus the source hash
  (`0.3.0.<rev>`), which sorts above `0.3.0` and below the next release.
  systemd-boot and
  systemd-sysupdate use `strverscmp_improved`, which orders numeric segments
  correctly (`0.10.0 > 0.9.0`, `0.3.1 > 0.3.0`). Use `~rc` for a pre-release;
  a trailing `-rc` sorts above the release. The old epoch versions do not
  order against semver, so devices running them must reflash once.

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

If `qemu-img`/`qemu-system-x86_64` are not on `PATH`, `run-image` re-executes
itself in a Nix shell with QEMU (using the project's nix-portable fallback).
UEFI firmware is taken from the distro's OVMF packages or QEMU's bundled EDK2
blobs; set `SLIPPI_OVMF_DIR` to point at a specific firmware directory.

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

- **Validate:** pushes to `main`; Go tests, headless Godot, and NixOS
  evaluation.

Releases are built locally. CI does not build or sign images.

Release assets:

```text
slippios-usb-<version>.img.xz
slippios-usb-<version>-unsigned.img.xz
slippios_<version>.nix-store.raw
slippios_<version>.nix-store.raw.zsync
slippios_<version>.efi
manifest.json
SHA256SUMS
```

The default image is signed for Secure Boot. The `-unsigned` image is the
fallback for machines where the shim cannot boot.

Build, sign, and create a draft release:

```sh
mise run release
```

This builds the image incrementally, signs it with the key in
`~/.config/slippios/secureboot`, generates the assets, and creates a draft
release. Pass `--publish` to publish it:

```sh
mise run release -- --publish
```

Use `--no-build` to reuse `result-local-image`, `--dry-run` to generate assets
without uploading, and `--tag` to set the tag. The script refuses a dirty
working tree unless `--allow-dirty` is passed. It creates the release tag on
GitHub at the current commit. Assets are written to
`.cache/release/<version>/` (git-ignored).

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
scripts/release            local build, sign, and publish
scripts/run-image          QEMU/OVMF USB runner
scripts/fetch-image        release downloader
scripts/sign-image         Secure Boot signing for an image
scripts/sign-uki           Secure Boot signing for a standalone UKI
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
  7 the keyboard, 0 is disconnected. Each port is stored separately as
  `dolphin.controller1..4` and `dolphin.profile1..4`.
- `Dolphin.Core.SlotB=13` is set explicitly for the Slippi EXI device. Do not
  rely on the build default; an existing `Dolphin.ini` can override it.
- `InstallLogin` adds `latestVersion` to the Dolphin copy of `user.json`.
  Slippi's Rust `UserInfo` requires that field, and the launcher writes it.
  Do not add `latest_version` as well: it is a serde alias for the same field,
  and serde rejects the duplicate.
- `Graphics.Settings.InternalResolution` is the EFB scale (1 = native).
- `Graphics.Settings.AspectRatio`: 1 widescreen, 2 4:3, 3 Melee, 4 stretch.
- `Graphics.Settings.MSAA`: 1 off, then 2/4/8 samples.
- `Dolphin.Slippi.EnableJukebox` (beta) or `Core.SlippiJukeboxEnabled`
  (stable) is `dolphin.enable_music`.
- `Graphics.Settings.WaitForShadersBeforeStarting` is
  `dolphin.precompile_shaders`.
- `Graphics.Settings.ShaderCompilationMode`: 0 sync, 1 sync uber, 2 async
  uber, 3 async skip; stored as `dolphin.shader_mode`.
- `Graphics.Settings.ShowNetPlayPing` is `dolphin.show_netplay_ping`.
- `Graphics.Settings.ShowFPS` is `dolphin.show_fps`.
- `Dolphin.Display.RenderWindowWidth`/`RenderWindowHeight`,
  `BorderlessFullscreen`, and `RenderWindowAutoSize=False` are set from the
  current X resolution so Melee fills the screen with borderless fullscreen.
- Enums are written as integers; booleans as `True`/`False`.

**Settings → Dolphin → Gameplay → Controller Settings** writes `dolphin.controllerN` and, for pad
ports, copies the selected `Config/Profiles/GCPad/<name>.ini` into
`[GCPadN]` in `Config/GCPadNew.ini`. The `custom` profile leaves that section
alone. Detected pads are read from `/sys/class/input/js*/device/name`.

**Settings → Dolphin → Advanced → Gecko codes** reads the global
`Sys/GameSettings/GALE01r2.ini` from the extracted Slippi AppImage and writes
overrides to `users/default/dolphin/GameSettings/GALE01.ini`. Only differences
from the global defaults go into `[Gecko_Enabled]`/`[Gecko_Disabled]`; custom
codes live in `[Gecko]`. Required codes are locked. The CLI equivalents are
`slippios gecko list|set|add|remove`.

### In-game Quit button

Melee's top-level **Data** slot is repurposed as **Quit**. On first launch,
`gecko.EnsureQuitCode` writes `Required: SlippiOS Quit Button` into the local
`GALE01.ini`. The code hooks the main-menu think function at `0x8022DB40`,
checks for the Data slot (`hovered_selection == 4`) and Confirm, sends
`CMD_LOG_MESSAGE` with `SLIPPIOS_QUIT` over the Slippi EXI device, and clears
the button mask so the Data menu does not open.

`slippios play` and `slippios-session` tail `<profile>/logs/dolphin.log` with
`dolphin.WatchQuit` and send `SIGTERM` to Dolphin's process group on the
marker. `dolphin.QuitRequested` turns that exit into a clean result instead of
a launch error.

The label is
`backend/internal/dolphin/assets/tex1_176x30_d1fa83d25e26b8a9_2.png`, installed
under `Load/Textures/GALE01/SlippiOS/`. Stable enables
`Graphics.Settings.HiresTextures` in `GFX.ini`; beta passes the same key with
`-C`. The Slippi log type must be enabled too: stable writes `Logger.ini`,
beta passes `Logger.Logs.SLIPPI=True`. See ADR-0006.

**Settings → Advanced → Expert mode → Expert tools → Open Dolphin GUI** opens the
real Dolphin GUI for anything the curated list does not cover. Before the GUI
opens, the curated keys are written into the INIs. When it closes,
`slippios dolphin config` reads those keys back and updates `settings.toml` if
the GUI changed them; uncurated keys stay in the INIs. See ADR-0005.

PipeWire restores per-application stream volumes, and Dolphin can come back at
-20 dB. `slippios-audio.service` (a user unit) applies the saved sink volume and
keeps every new sink input at 100%. The volume setting is user-facing in
Settings → Display & Audio → Audio. RTKit is enabled so PipeWire can take
realtime priority.

The boot-time profile seeding creates the whole
`/data/users/default/dolphin/Config` tree as `player:users`; otherwise Dolphin
cannot write `Dolphin.ini`, `GFX.ini`, or `GCPadNew.ini`.

## Replays

The netplay Dolphin AppImage cannot play `.slp` files. The release image seeds
the pinned Ishiiruka playback build under
`/data/system/playback/<version>/`. Dev images and repairs fall back to
querying the Slippi GraphQL API, downloading the playback zip, and extracting
the AppImage plus its `Sys/` directory. Playback writes a comms file with
`mode: normal` and the replay path, then runs:

```text
slippi-appimage-run Slippi_Playback-x86_64.AppImage -u <profile>/dolphin-playback \
  -b -e <melee.iso> -i <comms.json>
```

The shell's Replays screen exits with code 47; the wrapper reads
`/data/system/replay-request` and runs `slippios replay play`. The playback
build uses a separate user directory and the same AppImage runner as Dolphin.

## Hardware support

The image is x86_64 and preloads the NixOS kernel modules, full
`linux-firmware`, Mesa (RADV/ANV/llvmpipe), PipeWire/ALSA, NetworkManager, and
the common USB HID drivers. udev loads drivers on demand. The release image has
no writable store and does not install drivers after setup; the browser setup is
for Slippi login and ROM selection only.

AMD and Intel GPUs are supported. NVIDIA proprietary drivers, some Broadcom
Wi-Fi devices, and out-of-tree controller drivers are not; they need a separate
image. Wireless controllers are intentionally not included (no BlueZ or
`xpadneo`); use the wired GameCube adapter or a wired USB pad.

## Boot splash

Plymouth uses a charge-style `two-step` theme: `modules/slippios/theme.nix`
splits `assets/marks/icon.svg` into the pulsing hexagon and the static S, then
renders 24 throbber frames with a sine opacity. The C throbber plugin animates
them over two seconds. This avoids the script plugin's refresh callback, which
did not animate in QEMU's frame-buffer fallback.

Plymouth frames are a fixed 324 px high (30% of 1080p), because two-step themes
cannot scale per display. The Godot splash does scale: it sizes the mark to 30%
of the screen height and pulses it. The game launch screen reuses only the
hexagon layer as the breathing indicator instead of the old dot spinner.

The initrd loads `virtio_gpu` and `bochs` so Plymouth has a framebuffer in a VM
before stage 2. Physical machines use the firmware framebuffer through the
built-in `simpledrm` until the native driver loads. NixOS hardcodes
`DeviceTimeout=8` in `plymouthd.conf`, which leaves console text on screen until
the wait expires; the theme module replaces that file with `DeviceTimeout=1` in
both the initrd and stage 2. With `loglevel=0` and `ShowStatus=no`, the splash
covers the console immediately and no TTY lines leak through.

When the session starts, the X session script paints a wallpaper generated for
the current desktop size on the X root window before Godot starts, so the
handoff shows the same mark instead of a blank screen. `session.nix` renders
the common sizes from `slippi-mark.png`; using a matching image instead of
scaling a 4K one down keeps the thin S stroke smooth. A minimal Openbox
starts before the Godot shell. It has no decorations or keybindings and handles
fullscreen, focus, and dialogs; the shell still sizes itself explicitly as a
fallback. If the shell exits, Openbox remains as the diagnostic desktop.

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
`appimageTools.defaultFhsEnvArgs`. Slippi's AppImage expects a normal desktop
set of system libraries (`harfbuzz`, `gmp`, `libgpg-error`, `libcom_err` and
friends). Keep the upstream environment rather than a hand-picked package
list; the upstream AppImage changes which libraries it leaves to the system
between releases.

The release image seeds the pinned stable AppImage under
`/data/system/apps/stable/`; `apps.EnsureStable` returns it without a network
lookup. It also seeds the pinned beta AppImage under
`/data/system/apps/beta/`; `dolphin.Ensure` returns it before contacting
GitHub. Dev images and repairs fall back to downloading.

## Melee ROMs

The USB image creates a 5 GiB FAT32 partition labelled `MELEE-ROMS`.
`slippios-isos.service` waits for the label, mounts it at `/data/isos`, and
writes the result to `/data/system/isos-mount`. `slippios isos` reports
`mounted` and `mount_error`; the shell shows a mount failure instead of an empty
ROM list. The partition is fixed-size; the ext4 `/data` partition after it grows
to fill the stick. The initrd grows it once (parted, `e2fsck -f`, `resize2fs`)
and skips the work on later boots when the partition already reaches the end of
the disk. Images keep their original filenames. SlippiOS validates only the
GameCube game ID, not a hash.

## Boot and game handoff

Godot never launches Dolphin itself. It exits with a code and the
`slippios-shell` wrapper acts on it:

| Exit | Wrapper action |
| --- | --- |
| 42 | `slippios play`, then reopen the shell |
| 43 | `slippios dolphin config` (Dolphin settings window), then reopen |
| 44 | xterm running a login shell, then reopen |
| 45 | xterm running nano on settings.toml, then reopen |
| 46 | restart the shell at the new display resolution |
| anything else | exit; the X session starts Openbox |

This releases the GameCube adapter for Dolphin instead of leaving it claimed by
SDL. The wrapper reopens Godot with `SLIPPIOS_AUTOPLAY=0` and
`SLIPPIOS_SKIP_SPLASH=1`, so the splash is cold-boot only.

The X session starts a minimal Openbox, then the shell, and keeps Openbox
running as the fallback desktop. If the shell wrapper exits, the session opens
a terminal; if Openbox exits, it writes `/data/system/shell-failed`;
`slippios-recovery.service` then conflicts with the display manager and takes
over tty1 with a recovery console: last shell log lines, failed units, and Try
again / Power off / Reboot.

### Window manager

SlippiOS runs a minimal Openbox before the shell. It has no decorations or
keybindings and its root menu is limited to the shell, terminal, text editor,
Dolphin settings, reboot, and power off. Openbox handles
`_NET_WM_STATE_FULLSCREEN`, focus, and dialogs. The shell and Dolphin still
size themselves explicitly, so a resolution change or a missing window manager
does not break the layout.

Openbox was adopted after hardware testing on the EM680: fullscreen, focus,
the Dolphin settings window, and game frame pacing were acceptable, and the
fallback desktop is useful when the shell exits.

## Secure Boot

The signed image uses the Fedora 16.1-7 shim and a SlippiOS MOK key. See
[secure-boot/README.md](../secure-boot/README.md) for the chain and limits.

`scripts/release` signs the image and extracts the signed UKI from it, so the
update asset matches the image. It reads the key from `SLIPPIOS_SB_KEY` and
`SLIPPIOS_SB_CERT`, or from `~/.config/slippios/secureboot/slippios.key` and
`.crt`. Keep the private key offline and out of the repository.

To sign an image without publishing:

```sh
SLIPPIOS_SB_KEY=/path/to/slippios.key \
SLIPPIOS_SB_CERT=/path/to/slippios.crt \
  mise run sign:image -- result-local-image/slippios_<version>.raw signed.raw
```

The script downloads the pinned shim RPM, verifies its SHA-256 and both
Microsoft signatures, signs systemd-boot and the UKI, and installs the shim,
MokManager, and `slippios.cer`.

Test with OVMF Secure Boot:

```sh
SLIPPI_SECURE_BOOT=1 SLIPPI_DISPLAY=none mise run run:image
```

The first boot shows the shim verification error and MokManager. Enroll
`slippios.cer`; later boots and updates use the MOK key.

## Display and audio

```sh
slippios display apply        # apply resolution, refresh, and audio
slippios audio apply          # apply the saved sink and volume
slippios audio volume [0-100] # show or set the sink volume
slippios audio sinks          # JSON list of PulseAudio outputs
```

`backend/internal/desktop` is shared by the shell (through these commands) and
the terminal session, so both apply the same settings.toml values.
`slippios display modes` reads the driver's xrandr mode list (which is derived
from EDID) plus the EDID monitor name; `slippios display mode` and
`slippios display custom` switch or create modes. The session script applies the
saved mode before starting the shell, so the X screen is already at the target
resolution when Godot creates its window. The shell applies it again, sizes a
borderless window to the screen, and re-adds a custom CVT mode because X forgets
runtime modes. Changing the mode from the shell exits with code 46; the wrapper
recomputes the screen resolution and reopens the shell at the new size.

## Wi-Fi

```sh
slippios wifi scan
slippios wifi status
slippios wifi connect --ssid "Network" --password "secret"
```

The shell wraps this in **Settings → Network → Wi-Fi**. `nmcli` comes from the
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
Boot image is a verified prototype; hardware boot and Mac boot are still open.

## Publishing releases to R2

The updater defaults to the public R2 bucket:

```text
https://pub-ac47e58c86e44a728518e30a6eb756da.r2.dev/manifest.json
```

Generate the signed assets with `scripts/release --dry-run --keep-raw`, then
publish the asset directory:

```sh
scripts/publish-r2 .cache/release/<version>
```

The script reads `~/.config/slippios/r2.env` (stored encrypted with chezmoi) and
uploads the full/unsigned images, store image, UKI, and checksums before
`manifest.json`. R2 has no GitHub 2 GiB asset limit, so the full image can be a
single file.
