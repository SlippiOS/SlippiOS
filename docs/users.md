# SlippiOS player guide

## Requirements

- 16 GB or larger USB stick.
- UEFI boot. Secure Boot off.
- Melee 1.02 ISO (NTSC-U, NTSC-J, or PAL).
- Network access for Slippi sign-in and the first Dolphin download.

## Flash

Download `slippios-usb-<version>.img.xz` from the
[latest release](https://github.com/SlippiOS/SlippiOS/releases/latest).

- **Etcher:** select the `.img.xz`.
- **Rufus:** select the `.img.xz` and use DD mode.
- **dd:**

```sh
sudo dd if=slippios-usb-<version>.img.xz of=/dev/sdX bs=4M status=progress conv=fsync
```

Use the whole device, not a partition. This erases the USB.

## First boot

1. Flash the USB image.
2. Open the `MELEE-ROMS` drive on your computer and copy in your Melee images.
   This is optional; you can also add them from the setup page.
3. Boot from the USB in UEFI mode.
4. Wait for setup. The data partition grows to fill the stick.
5. Scan the setup QR with a phone, or open the shown address on any computer on
   the same network.
6. **Step 1 — Game:** choose one of the images on `MELEE-ROMS`, or upload another.
7. **Step 2 — Account:** sign in to Slippi, or choose **Skip for offline play**.
8. The console starts the selected image automatically.

The setup page is a two-step browser wizard. It works on a phone, tablet, or
computer. Afterwards, use **Melee ROMs** in the shell to change the selected
image.

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
on the console or **Sign in on another device**.

Passwords are not stored. `slippi-user.json` is an auth token, readable only by
the appliance user, and bound to the current PC. Moving the USB to another PC
requires signing in again.

The setup pages are LAN-only, tokenized in the URL, and served over HTTP. Use
them on a trusted network.

## Play

**Play** launches Slippi Dolphin with the selected ISO. Standard SDL controllers
work. Controller profiles:

[controllers.md](controllers.md)

Dolphin is downloaded and verified on first launch and updates separately from
the system.

## Update

1. **Update**.
2. **Install system update**.
3. Wait, then **Restart**.

The inactive system slot is updated. `/data` is untouched. A failed boot falls
back to the previous system after three attempts. Downloads use zsync deltas
when possible and fall back to a full download.

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

## Troubleshooting

**“Network page stopped”** — the setup page exited. The console shows the
reason. Press **Restart setup page**. Only one page can run at a time.

**Setup page does not load** — the device and console must be on the same network.
Disable VPN/client isolation. In the QEMU VM, use
`http://localhost:8080/<token>/` from the host.

**No Melee image** — the file must be a GameCube Melee `.iso` with ID `GALE01`,
`GALJ01`, or `GALP01`. Copy it to the `MELEE-ROMS` drive, then use **Melee ROMs**
or **Restart setup page**.

**Nothing persists** — use the USB `.img.xz` release, not the development ISO.

**Does not boot** — disable Secure Boot and select the USB device in the UEFI
boot menu.

**Update check fails** — check network access to GitHub and retry.

**End-of-support warning** — update the system or flash a current release.
