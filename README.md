# SlippiOS

SlippiOS is a console-style NixOS appliance for booting directly into Slippi
Melee. This repository contains the NixOS configuration and ISO build
pipeline; virtual machines are run externally with virt-manager.

## Build and download the ISO

Push this repository to GitHub and run **Build ISO** from the Actions tab. The
workflow builds the `slippi-iso` target and uploads the ISO as an artifact.
Download it from the completed workflow run, then create a VM in virt-manager
and attach the ISO as its boot disk.

The target follows the NixOS manual's live-ISO method: it imports
`nixos/modules/installer/cd-dvd/installation-cd-minimal.nix` and builds
`config.system.build.isoImage`.

For a local build, on a machine with Nix and flakes enabled:

```sh
nix build .#nixosConfigurations.slippi-iso.config.system.build.isoImage
ls result/iso/*.iso
```

The live ISO must contain its full system closure. It cannot install packages
at runtime, so build time is dominated by compressing that closure. SlippiOS
uses `zstd -Xcompression-level 6` instead of nixpkgs' `-19` default to keep CI
builds practical while remaining bootable.

The ISO auto-logs into a minimal X11/Openbox session and starts the IPL
(Initial Program Loader). The real GameCube adapter requires exclusive USB
access and should be tested later with USB passthrough from virt-manager.

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
