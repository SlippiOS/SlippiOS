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

1. Boot from the USB in UEFI mode.
2. Wait for the setup QR. The data partition grows to fill the stick.
3. Scan the QR with a phone or open the shown address.
4. **Step 1 — Game:** upload a Melee 1.02 ISO, or pick one already on the stick.
5. **Step 2 — Account:** sign in to Slippi, or choose **Skip for offline play**.
6. The console starts Melee automatically. Keep the phone on the same network
   until the page says setup is complete.

The setup page is a two-step wizard. You do not need to choose anything on the
TV during first boot.

## Connect Wi-Fi

1. **Settings → Wi-Fi network**.
2. Select a network.
3. Enter the password with the on-screen keyboard.
4. Wait for the confirmation.

The console uses NetworkManager. Ethernet works automatically.

## Add a Melee ISO

On first boot, the setup wizard handles this on your phone. You can also put an
ISO on the stick directly:

1. Mount the ext4 partition labelled `SLIPPIOS`.
2. Copy the ISO into `isos/`.
3. Reboot without a valid game image, or open setup again.

Windows and macOS cannot read ext4 without extra drivers.

## Sign in

First boot signs you in on the phone setup page. Later, use **Account → Sign in**
on the console or **Sign in on phone**.

Passwords are not stored. `slippi-user.json` is an auth token, readable only by
the appliance user, and bound to the current PC. Moving the USB to another PC
requires signing in again.

The phone pages are LAN-only, tokenized in the URL, and served over HTTP. Use
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

**“Network page stopped”** — the local page exited. The console shows the
reason. Press **Restart setup page**. Only one page can run at a time.

**Phone page does not load** — phone and console must be on the same network.
Disable VPN/client isolation. In the QEMU VM, use
`http://localhost:8080/<token>/` from the host.

**No game image** — the file must be a Melee 1.02 `.iso` with ID `GALE01`,
`GALJ01`, or `GALP01`. Copy it into `isos/` and press **Restart setup page**.

**Nothing persists** — use the USB `.img.xz` release, not the development ISO.

**Does not boot** — disable Secure Boot and select the USB device in the UEFI
boot menu.

**Update check fails** — check network access to GitHub and retry.

**End-of-support warning** — update the system or flash a current release.
