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
├── isos/             Melee 1.02 and derivative builds (*.iso)
├── dolphin/          Dolphin user dir (accounts, config, saves)
├── replays/
└── logs/
```

`settings.toml` references a single game image for now:

```toml
schema_version = 1

[game]
iso = "/slippi/isos/melee.iso"

[display]
refresh_hz = 120
resolution = "auto"

[audio]
sink = "auto"

[network]
ssid = "Home"
psk = "plaintext-for-now"
```

On boot the IPL behaves as follows:

- settings **and** a valid image present → launches Slippi Dolphin
- neither present → first-time setup
- only one present, or the image is invalid → alert, then setup

Game images are validated by their GameCube ID (`GALE01`/`GALJ01`/`GALP01`).
SlippiOS never downloads or ships Melee.

Dolphin is the pinned Slippi Mainline release, wrapped with `appimageTools`.
A runtime updater that tracks the newest upstream build is the next milestone.

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
