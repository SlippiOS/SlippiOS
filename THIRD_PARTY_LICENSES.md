# Third-party licenses

SlippiOS itself is licensed under **GPL-3.0-or-later**. See `LICENSE` and
`NOTICE`.

This file lists the main third-party components in the release image and
their licenses. It is not exhaustive: NixOS packages retain their upstream
licenses, and their license files are available in the Nix store and under
`/run/current-system/sw/share/licenses` on a running system.

License texts for the common licenses are in `licenses/`. The Go dependency
notices are in `licenses/go/`.

## Slippi / Dolphin

- **Slippi stable 3.6.4 (Ishiiruka)** — GPL-2.0-or-later.
  Source: <https://github.com/project-slippi/Ishiiruka>
- **Slippi beta 4.0.0-mainline-beta.19 (Dolphin)** — GPL-2.0-or-later.
  Source: <https://github.com/project-slippi/dolphin>
- **Slippi playback 3.5.2** — GPL-2.0-or-later.
  Source: <https://github.com/project-slippi/Ishiiruka-Playback>

  Note: the GitHub API currently returns 404 for the playback repository.
  Before commercial distribution, confirm the corresponding source with the
  Slippi project. The stable and beta sources are available.

The Slippi AppImages include additional libraries. Their copyright notices
are in `usr/share/doc/*/copyright` inside each AppImage. The GameCube font
licenses are in `Sys/GC/font-licenses.txt`.

The GPLv2 text is in `licenses/GPL-2.0.txt`.

## Kernel

- **CachyOS BORE kernel 7.3.0** — GPL-2.0-only.
  Source: <https://github.com/xddxdd/nix-cachyos-kernel> and the CachyOS
  kernel source. The exact inputs are pinned by `flake.lock`.

## NixOS and system packages

- **NixOS** — MIT. Source: <https://github.com/NixOS/nixpkgs>
- **systemd** — LGPL-2.1-or-later and permissive licenses.
- **glibc** — LGPL-2.1-or-later.
- **Openbox** — GPL-2.0-or-later.
- **LightDM** — GPL-3.0-or-later / LGPL-2.0-only / LGPL-3.0-only.
- **PulseAudio** — LGPL-2.1-or-later.
- **qrencode** — LGPL-2.1-or-later.
- **NetworkManager** — GPL-2.0-or-later.
- **Bash, coreutils, nano** — GPL-3.0-or-later.
- **xz** — GPL-2.0-or-later / LGPL-2.1-or-later.
- **zsync** — Artistic-2.0.
- **Mesa** — MIT.
- **Vulkan loader** — Apache-2.0.
- **Xorg utilities** — MIT / HPND.
- **Godot 4** — MIT. Source: <https://github.com/godotengine/godot>
  Godot's third-party notices are in its `COPYRIGHT.txt`.
- **Secure Boot shim** — BSD-2-Clause. The license is included in the image
  as `/SHIM-LICENSE.txt` and in this repository as
  `secure-boot/LICENSE.shim`.

The NixOS package set includes many additional GPL, LGPL, MIT, BSD, and
permissive components. Their license files are in their Nix store paths.
The project's Nix expressions pin the exact revisions in `flake.lock`.

## Go dependencies

- **github.com/spf13/cobra** — Apache-2.0.
- **github.com/spf13/pflag** — BSD-3-Clause.
- **github.com/inconshreveable/mousetrap** — Apache-2.0.
- **github.com/pelletier/go-toml/v2** — MIT.
- **Go standard library** — BSD-3-Clause.

The notices are in `licenses/go/`. The versions are pinned in
`backend/go.mod` and `backend/go.sum`.

## Firmware

The image includes redistributable firmware for hardware support. These are
not OSI open-source licenses, but they permit redistribution:

- **linux-firmware / amdgpu firmware** — `unfreeRedistributableFirmware`.
  Source and licenses:
  <https://git.kernel.org/pub/scm/linux/kernel/git/firmware/linux-firmware.git>
  See the `WHENCE` file and `LICENSE.*` files.
- **ipw2200-firmware** — Intel redistributable firmware license.
- **rtl8761b-firmware, rt5677-firmware, rtl8192su-firmware** — Realtek
  redistributable firmware licenses.
- **libreelec-dvb-firmware** — redistributable DVB firmware licenses.
- **sof-firmware** — BSD-3-Clause / ISC.

The firmware is included only to make the hardware work. It is not part of
the GPL-3.0-or-later project code.

## Fonts

- Dolphin GameCube fonts: see `Sys/GC/font-licenses.txt` in the Slippi
  AppImages.
- Maven Pro: SIL Open Font License 1.1, see
  `apps/shell/assets/fonts/OFL.txt`.
- Godot default fonts: see Godot's `COPYRIGHT.txt`.
- Any additional font files retain their upstream licenses.

## Branding

The Slippi logo used in the splash and Plymouth theme is a temporary
third-party mark from <https://slippi.gg/>. It is not covered by the GPL.
Replace it with an original SlippiOS mark or obtain permission from the
Slippi project before commercial distribution. The name "SlippiOS" includes
"Slippi"; check the Slippi project's trademark policy.

## Game image

The Melee ISO is not included in the image. The `MELEE-ROMS` partition is
empty; users must supply their own legally obtained game image.

## Source offer

The corresponding source for the GPL/LGPL components is available from the
upstream repositories listed in `SOURCE`. The SlippiOS source is at
<https://github.com/SlippiOS/SlippiOS>. The release manifest records the
exact SlippiOS commit. For a physical distribution, include this file,
`LICENSE`, `SOURCE`, and the license texts with the USB, and keep the source
available for at least three years.
