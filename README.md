# SlippiOS

A console-style NixOS appliance for Slippi Melee. Boots into a fullscreen Godot
shell and stores games, settings, the Slippi login, and replays on the USB.

**Status:** alpha. A/B system updates, phone sign-in, Wi-Fi setup, and network
ISO upload work. The Go control backend is in place; the Rust GameCube-adapter
daemon is in progress.

## Players

Requirements: 16 GB USB, UEFI boot, Secure Boot off.

1. Download `slippios-usb-<version>.img.xz` from the
   [latest release](https://github.com/SlippiOS/SlippiOS/releases/latest).
2. Flash with Etcher, Rufus (DD mode), or `dd`.
3. Boot the stick.
4. In Setup, add a Melee 1.02 ISO and sign in to Slippi.

Updates: **Update → Install system update → Restart**. Games, settings, login,
Dolphin, and replays are preserved.

**Full guide:** [docs/users.md](docs/users.md)

## Developers

```sh
mise trust
mise install
mise run build:image
mise run run:image
mise run test
```

No Docker or system Nix required; the build falls back to nix-portable.

**Full guide:** [docs/developers.md](docs/developers.md)

## Docs

- [Player guide](docs/users.md)
- [Developer guide](docs/developers.md)
- [Architecture and updates](docs/architecture.md)
- [Controller support](docs/controllers.md)
