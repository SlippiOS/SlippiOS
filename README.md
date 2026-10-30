# SlippiOS

SlippiOS is a Melee console on a USB stick. It boots into a fullscreen Godot
shell and keeps games, settings, the Slippi login, and replays on the stick.

**Status:** 0.4.2. A/B system updates, browser sign-in, Wi-Fi setup, network ISO
upload, native GameCube-adapter input, and signed Secure Boot images work.
Booting straight into a game and opening the menu with hold-A still need
real-hardware validation.

## Users

Requirements: a 16 GB USB stick and UEFI boot. The unsigned image requires
Secure Boot off (a one-time firmware setting). The signed image uses a one-time
MOK enrollment; see [secure-boot/README.md](secure-boot/README.md).

1. Download `slippios-usb-<version>.img.xz` from the
   [SlippiOS SourceForge project](https://sourceforge.net/projects/slippi-os/files/).
   Release notes and source are on
   [GitHub](https://github.com/SlippiOS/SlippiOS/releases/latest).
2. Flash it with balenaEtcher, or extract it with 7-Zip and use Rufus in DD
   mode. See the [user guide](docs/users.md#flash) for macOS and Linux
   commands.
3. Boot the stick.
4. Scan the setup QR with a phone, or open the address on a computer. Add a
   Melee 1.02 ISO, then sign in or skip.

Updates: **Update → Install system update → Restart**. Games, settings, the
login, Dolphin, and replays are preserved.

**Full guide:** [docs/users.md](docs/users.md)

## Developers

```sh
mise trust
mise install
mise run build:image
mise run run:image
mise run test
```

No Docker or system Nix is required. The build falls back to nix-portable.

**Full guide:** [docs/developers.md](docs/developers.md)

## License

SlippiOS is free software under **GPL-3.0-or-later**. See
[LICENSE](LICENSE), [NOTICE](NOTICE),
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md), and [SOURCE](SOURCE).
The image includes third-party software under its own licenses, including
GPL-2.0-or-later Slippi/Dolphin builds and redistributable firmware. The
Melee ISO is not included; users supply their own game image.

## Docs

- [User guide](docs/users.md)
- [Developer guide](docs/developers.md)
- [Architecture and updates](docs/architecture.md)
- [Controller support](docs/controllers.md)
- [Onboarding and launch flow](docs/onboarding-flow.md)
- [Secure Boot](secure-boot/README.md)
- [Roadmap](ROADMAP.md)
