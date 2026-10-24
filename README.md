# SlippiOS

SlippiOS is a console-style NixOS appliance for booting directly into Slippi
Melee. This repository contains the NixOS configuration and ISO build
pipeline; virtual machines are run externally with virt-manager.

## Build the ISO locally

Local builds are the fast path: Nix caches every dependency, so only the
changed parts rebuild. One-time setup on a non-NixOS host:

```sh
sh <(curl -L https://nixos.org/nix/install) --daemon
```

After logging back in, enable flakes:

```sh
mkdir -p ~/.config/nix
echo 'experimental-features = nix-command flakes' >> ~/.config/nix/nix.conf
```

Then:

```sh
./scripts/build-iso
```

The ISO lands in `result/iso/`. A GitHub Actions workflow still exists as a
fallback, but it is not needed for normal iteration.

## Development loop

`./scripts/dev` builds the ISO and deploys it into libvirt in one step:

```sh
./scripts/dev
```

It installs the ISO at `/var/lib/libvirt/images/slippios.iso`, updates the
domain's CD-ROM if `slippios-iso` exists, and reboots or starts it. Override
the defaults with `SLIPPI_DOMAIN`, `SLIPPI_IMAGE`, or `LIBVIRT_URI`.

First run downloads the full closure (a few minutes). Later runs are much
faster because the Nix store is already populated.

The live ISO must contain its full system closure. It cannot install packages
at runtime, so build time is dominated by compressing that closure. SlippiOS
uses `zstd -Xcompression-level 6` instead of nixpkgs' `-19` default to keep CI
builds practical while remaining bootable.

## Quick test without libvirt

libvirt is only useful later, for persistent VMs and USB passthrough. To just
boot the ISO:

```sh
./scripts/run-vm /path/to/slippios.iso
```

That runs plain QEMU with KVM as your user — no daemon, no sudo.

The ISO auto-logs into a minimal X11/Openbox session and starts the IPL
(Initial Program Loader). The real GameCube adapter requires exclusive USB
access and should be tested later with USB passthrough from virt-manager.

## Data, settings, and game images

Everything user-owned lives on a partition labelled `SLIPPI`, mounted at
`/slippi` when present. Without that partition the live ISO falls back to a
temporary `/slippi` on the root filesystem, so nothing persists.

```text
/slippi
├── settings.toml     hand-editable; written by first-time setup
├── slippi-user.json  optional Slippi login export (playKey)
├── isos/             Melee 1.02 and derivative builds (*.iso)
├── dolphin-app/      downloaded Slippi Dolphin releases
│   ├── versions/<tag>/Slippi_Netplay_Mainline-x86_64.AppImage
│   └── current       → newest verified release
├── dolphin/          Dolphin user dir (accounts, config, saves)
├── replays/
└── logs/
```

`settings.toml` references a single game image for now:

```toml
schema_version = 1

[game]
iso = "melee.iso"

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

SlippiOS never stores a Slippi password. On first run it prompts for the Slippi
email and password, authenticates against the same public web endpoints used by
slippi.gg, fetches the account's `playKey`, and writes `slippi-user.json` into
Dolphin's user directory. The password is only held in memory for that request.

On boot the IPL behaves as follows:

- settings **and** a valid image present → launches Slippi Dolphin
- neither present → first-time setup
- only one present, or the image is invalid → alert, then setup

Game images are validated by their GameCube ID (`GALE01`/`GALJ01`/`GALP01`).
SlippiOS never downloads or ships Melee.

Dolphin is not baked into the image. On boot the IPL checks the newest Slippi
Mainline release, downloads it to `/slippi/dolphin-app/versions/<tag>/`,
verifies GitHub's SHA-256 digest, and points `current` at it. Netplay requires
matching Dolphin versions, so the appliance always follows upstream. If a
check or download fails, the current build keeps working.

`slippi-iso` is the current hardware-independent target. GPU drivers, adapter
rules, persistence, and the installed-PC target will be added as separate NixOS
modules rather than tied to a VM.

## Roadmap

1. Bootable ISO and IPL state machine (current milestone).
2. Controller-driven config UI using keyboard input in virt-manager and raw
   adapter input on hardware.
3. USB hand-off tests: IPL releases `057e:0337` before Dolphin starts.
4. Package/pin the mainline Slippi Dolphin beta.
5. Persistent USB image and install-to-disk flow.

## slippios-cli

The graphical IPL talks to system logic through a JSON CLI rather than
importing Python:

```sh
slippios-cli status            # settings, login, Dolphin version, adapter
slippios-cli update --check    # current vs newest upstream release
slippios-cli update            # download and verify the newest release
slippios-cli login --stdin     # {"email": "...", "password": "..."}
slippios-cli play              # install login, update, launch Dolphin
slippios-cli adapter           # adapter presence and A-button state
```

Every command prints one JSON object; long-running commands stream progress to
stderr. Passwords are read from stdin, never argv.
