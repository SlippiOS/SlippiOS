# Controller support

The IPL reads standard SDL gamepads (Glyph, Xbox, DualSense, ...) and the raw
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

B0XX, B0XX_Linux, B0XX_macOS, Frame1, Glyph, HayBox_DInput,
HayBox_DInput_Linux, HayBox_DInput_macOS, HayBox_XInput,
HayBox_XInput_Linux, slippibot.

Sources: the user's Slippi install, `JonnyHaystack/HayBox`, and GRAM/HayBox
forks. Load them from Dolphin's controller profile menu.
