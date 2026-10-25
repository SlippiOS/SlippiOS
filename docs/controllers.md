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

Boot-time "hold A" therefore maps to the SDL south face, which is what
`slippi_input.Gamepads.any_south_held()` checks.

## GameCube adapter

`057e:0337` is polled directly for A during boot. While the config menu exists,
the adapter must be exposed as a virtual SDL pad (uinput) and then released
before Dolphin starts, so Dolphin can claim it with libusb.

## Bundled profiles

SlippiOS seeds these community profiles into
`/slippi/dolphin/Config/Profiles/GCPad/` on first boot (existing files are not
overwritten):

B0XX_Linux, Frame1_Linux, Glyph, HayBox_DInput_Linux, HayBox_XInput_Linux.

`Frame1_Linux.ini` is derived from the upstream Ishiiruka Frame1 profile with
the DirectInput bindings translated to evdev names. The device is named
`evdev/0/Frame1`; if your unit enumerates under a different name, select the
device manually in Dolphin (the button/axis bindings stay valid).

Only Linux profiles are shipped. Sources: the user's Slippi install and
`JonnyHaystack/HayBox`. Load them from Dolphin's controller profile menu.
