# Controller support

The shell reads standard SDL gamepads (Glyph, Xbox, DualSense, ...) and the raw
GameCube adapter in parallel.

## Glyph profile (from the user's Slippi install)

`Device = XInput/0/Gamepad`, i.e. SDL sees a standard gamepad:

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

Boot-time "hold A" maps to the SDL south face. The shell currently has no
controller navigation; that arrives with the virtual pad.

## GameCube adapter

`057e:0337` is the official Wii U/Mayflash adapter. `joybusd` is the planned
Rust daemon that will expose its four ports as virtual SDL pads through uinput.
It will release the adapter before Dolphin starts so Dolphin can claim it with
libusb. The Go CLI only reports adapter presence today.

## Bundled profiles

SlippiOS seeds these community profiles into
`/data/users/default/dolphin/Config/Profiles/GCPad/` on first boot (existing files are not
overwritten):

B0XX_Linux, Frame1_Linux, Glyph, HayBox_DInput_Linux, HayBox_XInput_Linux.

`Frame1_Linux.ini` is derived from the upstream Ishiiruka Frame1 profile with
the DirectInput bindings translated to evdev names. The device is named
`evdev/0/Frame1`; if your unit enumerates under a different name, select the
device manually in Dolphin (the button/axis bindings stay valid).

Only Linux profiles are shipped. Sources: the user's Slippi install and
`JonnyHaystack/HayBox`. Load them from Dolphin's controller profile menu.
