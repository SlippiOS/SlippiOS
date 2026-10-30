# SlippiOS user guide

## Requirements

- A 16 GB or larger USB stick.
- UEFI boot. The unsigned image needs Secure Boot off (a one-time firmware
  setting). The signed image needs a one-time MOK enrollment.
- A Melee 1.02 ISO (NTSC-U, NTSC-J, or PAL).
- Network access for Slippi sign-in and system updates. The full image bundles
  the stable, beta, and playback builds; the minimal image downloads them on
  first boot.

## Image variants

- `slippios-usb-<version>.img.xz` — full image. Stable, beta, and playback are
  preinstalled, so it works offline after the Melee ISO is added.
- `slippios-usb-<version>-minimal.img.xz` — minimal image. The apps are
  downloaded and installed on first boot, so setup needs internet.

Both variants use the same A/B system updates.

## Flash

Download `slippios-usb-<version>.img.xz` and `SHA256SUMS` from the
[SlippiOS SourceForge project](https://sourceforge.net/projects/slippi-os/files/).
Release notes and source are on
[GitHub](https://github.com/SlippiOS/SlippiOS/releases/latest).

The download is an xz-compressed image. Most desktops cannot open it by
double-clicking, so use one of the methods below.

Verify the download first:

```sh
# Linux
grep 'slippios-usb-<version>.img.xz' SHA256SUMS | sha256sum -c -

# macOS
grep 'slippios-usb-<version>.img.xz' SHA256SUMS | shasum -a 256 -c -
```

On Windows, compare the output of this with the matching line in `SHA256SUMS`:

```powershell
Get-FileHash .\slippios-usb-<version>.img.xz -Algorithm SHA256
```

### Windows

Easiest: install [balenaEtcher](https://etcher.balena.io/), select the
`.img.xz`, and flash.

If Rufus does not accept the `.xz` directly:

1. Install [7-Zip](https://www.7-zip.org/).
2. Right-click the `.img.xz` and extract the `.img`.
3. Open Rufus, select the `.img`, choose **DD Image mode**, and flash.

Do not use Windows Explorer to open the image.

### macOS

Install balenaEtcher and select the `.img.xz`, or use the command line:

```sh
brew install xz
diskutil list
diskutil unmountDisk /dev/diskN
xzcat slippios-usb-<version>.img.xz | \
  sudo dd of=/dev/rdiskN bs=4m status=progress
```

Use the raw `rdiskN` device. Do not write to a partition such as `diskNs1`.

### Linux

```sh
xzcat slippios-usb-<version>.img.xz | \
  sudo dd of=/dev/sdX bs=4M status=progress conv=fsync
```

Use the whole device, not a partition. This erases the USB.

After flashing, the `MELEE-ROMS` FAT32 drive should appear. If it does not, eject
and reconnect the stick, or check that the image finished writing.

## First boot

1. Flash the USB image.
2. Open the `MELEE-ROMS` drive on your computer and copy in your Melee images.
   This is optional; you can also add them from the setup page.
3. Boot from the USB in UEFI mode.
4. Wait for setup. The data partition grows to fill the stick.
5. **Game.** If the console asks, choose one of the images on `MELEE-ROMS`.
6. **Controller.** Confirm the detected wired controller, or choose
   **Skip for offline play**.
7. Scan the setup QR with a phone, or open the shown address on any computer on
   the same network.
8. **Account.** Sign in to Slippi, or choose **Skip for offline play**.
9. **Minimal image only:** the console downloads and installs the stable, beta,
   and playback builds. Keep it online until the install finishes.
10. The console returns to the shell menu. Press **Play** to start Melee.

The setup page is a two-step browser wizard. It works on a phone, tablet, or
computer. The sign-in step is offered on first boot even when Ethernet is
connected and a Melee ISO is already on the drive. Afterwards, use **Melee ROMs**
in the shell to change the selected image.

`MELEE-ROMS` is a normal FAT32 drive, so you can plug the stick into any PC and
add or remove images later. SlippiOS does not rename them.

## Connect Wi-Fi

1. **Settings → Wi-Fi network**.
2. Select a network.
3. Enter the password with the on-screen keyboard.
4. Wait for the confirmation.

The console uses NetworkManager. Ethernet works automatically.

## Add Melee images

1. Plug the SlippiOS stick into a PC.
2. Open the FAT32 drive labelled `MELEE-ROMS`.
3. Copy in vanilla, training, or modded Melee ISOs.
4. Eject the drive and boot the console.
5. In the shell, open **Melee ROMs** to choose the default image, or use the
   setup page on first boot.

The partition is sized for vanilla Melee, Training Mode CE, and one modded build.
Files keep their original names. Do not put other user data on this drive; it is
only for Melee images.

## Sign in

First boot signs you in on the browser setup page. Later, use **Account → Sign in**
on the console or **Sign in on another device**. To remove the saved login, use
**Account → Sign out**; Melee will ask you to sign in again the next time you
start it.

Passwords are not stored. `slippi-user.json` is an auth token, readable only by
the `player` user, and bound to the current PC. Moving the USB to another PC
requires signing in again.

The setup pages are LAN-only, tokenized in the URL, and served over HTTP. Use
them on a trusted network.

## Play

**Play** launches Slippi Dolphin with the selected ISO. Use wired controllers:
the GameCube adapter or a wired USB pad. Wireless controllers are not
supported because Melee is latency-sensitive. Press **Escape** to exit Melee and
return to the shell. Press **Guide** to stop the current match and stay in
Dolphin. While the launching screen is visible, **Escape** cancels the launch
and returns to the menu. **Settings → Dolphin Settings**
changes Dolphin's graphics backend,
internal resolution, aspect ratio, VSync, anti-aliasing, and audio without
leaving the console. SlippiOS maps these onto Dolphin's command-line config.
The same screen has **Gecko codes** for the optional Melee patches.

[controllers.md](controllers.md)

**Settings** lists the display modes the monitor advertises (read from
xrandr/EDID), the audio outputs, Wi-Fi, and **Update**. Changes apply
immediately and are saved for the next boot.

**Settings → Controller Settings** assigns each of the four GameCube ports to a
GameCube adapter, a standard pad with a Dolphin profile, the keyboard, or
nothing. The screen marks profiles whose device is currently connected.

**Settings → Dolphin Settings → Gecko codes** toggles the built-in Slippi codes.
The required Slippi Online codes are locked; the optional codes (widescreen,
flash on failed L-cancel, and so on) can be changed. Custom codes can be added
from the Expert terminal with `slippios gecko add <file>`.

**Expert mode** (inside Settings) unlocks tools for troubleshooting and
tinkering: the Dolphin GUI, a terminal, a text editor, and a custom display
mode. The custom mode takes `WIDTHxHEIGHT` or `WIDTHxHEIGHT@HZ`; it is added
with a CVT modeline and kept across reboots.

**Replays** lists the Slippi `.slp` files and opens them in the bundled Slippi
playback build. It starts immediately; no download is needed.

**Quit** powers the console off cleanly. Wait for the screen to go dark before
unplugging the drive.

The stable and beta Dolphin builds and the playback build are bundled and
verified with the image. They update with system updates.

## Update

1. **Settings → Update**.
2. **Install system update**.
3. Wait, then **Restart**.

The inactive system slot is updated. `/data` is untouched. A failed boot falls
back to the previous system after three attempts. Downloads use zsync deltas
when possible and fall back to a full download.

Versions before `0.3.0` used a numeric build ID. Those devices must reflash
once; automatic updates start with `0.3.0`. After that, updates use semver.

## Data

```text
/data
├── isos/
├── system/                  Dolphin, hardware ID, update state
└── users/default/
    ├── settings.toml
    ├── slippi-user.json
    ├── dolphin/
    ├── replays/
    └── logs/
```

Reflashing the whole USB erases this partition. There is no automatic backup
yet.

## License

SlippiOS is free software under **GPL-3.0-or-later**. The source is at
<https://github.com/SlippiOS/SlippiOS>. The release includes `LICENSE`,
`NOTICE`, `THIRD_PARTY_LICENSES.md`, and `SOURCE`. The Melee ISO is not
included; you must supply your own game image.

## Secure Boot

The signed image boots with Secure Boot on. The first boot needs a one-time
MOK enrollment per PC. See [the enrollment steps](../secure-boot/ENROLL.txt).

After enrollment, updates use the same key and do not ask again. Some PCs
require **Allow Microsoft 3rd Party UEFI CA**; Intel T2 Macs require
**Allow booting from external or removable media**; Apple Silicon cannot run
this x86_64 image.

If the shim cannot boot, use the unsigned image with Secure Boot off.

## Troubleshooting

**The network page stopped.** The setup page exited. The console shows the
reason. Press **Skip for offline play** to reach the menu, or reboot to retry
the wizard. Only one page can run at a time.

**The setup page does not load.** The device and console must be on the same
network. Disable VPN or client isolation. In the QEMU VM, use
`http://localhost:8080/<token>/` from the host.

**No Melee image.** The file must be a GameCube Melee `.iso` with ID `GALE01`,
`GALJ01`, or `GALP01`. Copy it to the `MELEE-ROMS` drive, then use **Melee ROMs**
to choose it, or **Settings → Run setup again**.

**The console shows a recovery menu instead of the shell.** The graphical shell
crashed and the Openbox fallback also exited. The menu shows the last error and
the failed systemd units, and offers **Try again**, **Power off**, and
**Reboot**. The full log is `/data/system/shell.log` on the SlippiOS data
partition. If only the shell crashed, you get an Openbox desktop instead.

**Nothing persists.** Use the USB `.img.xz` release, not a live environment.

**It does not boot.** Select the USB device in the UEFI boot menu. The unsigned
image requires Secure Boot off. The signed image requires a one-time MOK
enrollment; see [the enrollment steps](../secure-boot/ENROLL.txt). Some
firmware also requires **Allow Microsoft 3rd Party UEFI CA**.

**The update check fails.** Check network access to the SlippiOS mirror and
retry.

**End-of-support warning.** Update the system or flash a current release.
