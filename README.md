# SlippiOS

A console-style NixOS appliance for Slippi Melee. The live ISO boots straight
into a fullscreen Godot shell with no desktop environment.

## Layout

```text
hosts/        NixOS configurations (live ISO today)
modules/      shared NixOS modules
pkgs/         Nix package definitions
apps/
  session/    Python boot/session logic (settings, auth, updates, launch)
  shell/      Godot UI
  cli/        slippios, the JSON control CLI
assets/       controller profiles and branding
scripts/      host tooling (build, fetch, run)
docs/         notes
```

Runtime pieces:

- `slippios-shell` — fullscreen Godot menu (the console UI)
- `slippios-session` — headless fallback and boot flow (Python)
- `slippios` — state/actions consumed by the shell, human-first with
  automatic JSON when piped

## Build

Pushes to `main` build the ISO in GitHub Actions and upload it to the
transfer.sh mirror; the download link and SHA-256 are in the run summary.

Local builds need Nix once:

```sh
sh <(curl -L https://nixos.org/nix/install) --daemon
./scripts/build-iso
```

`boot.loader.timeout` is 1 second, so the ISO boots automatically; GRUB treats
`0` as “wait forever”.

## Deploy to libvirt

```sh
./scripts/fetch-iso --restart
```

Downloads the newest successful ISO into `~/.cache/slippios/`, verifies its
SHA-256, installs it to `/var/lib/libvirt/images/slippios.iso`, then cold-boots
the domain (a warm reboot can keep the old CD image open). Use `--offline` to
reuse a verified cached ISO, `--dry-run` to preview, and `--yes` to skip the
prompt when the newest run has not succeeded.

For a direct QEMU test without libvirt:

```sh
./scripts/run-vm /path/to/slippios.iso
```

## Data and settings

Everything user-owned lives on a partition labelled `SLIPPI`, mounted at
`/slippi`. Without it the live ISO uses a temporary `/slippi`, so nothing
persists.

```text
/slippi
├── settings.toml
├── slippi-user.json
├── isos/
├── dolphin-app/
├── dolphin/
├── replays/
└── logs/
```

```toml
schema_version = 1

[game]
iso = "melee.iso"          # relative to /slippi/isos

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
`/slippi/dolphin/Config/Profiles/GCPad/` on first boot. See
[`docs/controllers.md`](docs/controllers.md).

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

1. Shell polish: boot animation, account and settings screens.
2. Adapter helper: expose the GameCube adapter as a virtual SDL pad, then
   release it before Dolphin starts.
3. Live/install boot entries in the same ISO: one entry boots the kiosk, one
   runs the installer.
4. Persistent USB and install-to-disk image.
