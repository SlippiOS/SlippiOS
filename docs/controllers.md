# Controller support

The Godot shell reads controllers through SDL3, including the GameCube adapter.
SDL3 has a native HIDAPI driver for the Wii U/Mayflash adapter, so no separate
uinput daemon is needed.

## Glyph profile

`Device = XInput/0/Gamepad`. SDL sees a standard gamepad:

| Action | Dolphin binding | SDL |
|---|---|---|
| A | `Button A` | south face (button 0) |
| B | `Button B` | east face |
| X / Y | `Button X` / `Button Y` | west / north |
| Z | `Shoulder R` | right shoulder |
| Start | `Start` | start |
| L / R | `Trigger L` / `Trigger R` | analog triggers |
| Stick / C-stick | `Left X/Y`, `Right X/Y` | left / right sticks |
| D-Pad | `Pad N/S/W/E` | dpad |

Hold A during the boot screen to open the menu instead of launching Melee.
Keyboard Enter does the same for testing.

## GameCube adapter

`057e:0337` is the official Wii U/Mayflash adapter. The adapter switch must be
in **Wii U** mode.

While the shell is running, SDL3 reads the four controller ports directly.
Dolphin also supports the adapter natively through libusb. USB access is
exclusive, so the shell wrapper quits Godot before starting Dolphin and starts
Godot again after Dolphin exits. Do not run a uinput bridge at the same time.

Open **Dolphin settings** in the shell to bind controllers or load a profile.
SlippiOS closes its UI first, so the adapter is free for Dolphin's own
controller configuration.

## Bundled profiles

SlippiOS seeds these community profiles into
`/data/users/default/dolphin/Config/Profiles/GCPad/` on first boot (existing
files are not overwritten):

B0XX_Linux, Frame1_Linux, Glyph, HayBox_DInput_Linux, HayBox_XInput_Linux.

`Frame1_Linux.ini` is derived from the upstream Ishiiruka Frame1 profile with
the DirectInput bindings translated to evdev names. The device is named
`evdev/0/Frame1`; if your unit enumerates under a different name, select the
device manually in Dolphin (the button and axis bindings stay valid).

Only Linux profiles are shipped. Sources: the user's Slippi install and
`JonnyHaystack/HayBox`. Load them from Dolphin's controller profile menu.
