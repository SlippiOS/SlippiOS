# Controller support

The Godot shell reads controllers through SDL3, including the GameCube adapter.
SDL3 has a native HIDAPI driver for the Wii U/Mayflash adapter, so no separate
uinput daemon is needed.

Wireless controllers are intentionally not supported. Melee is
latency-sensitive; use the wired GameCube adapter or a wired USB pad. Bluetooth
and `xpadneo` are not included in the image.

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

Open **Settings → Controller Settings** to assign each port. Ports set to
**GameCube adapter** use Dolphin's native adapter support. Ports set to
**standard pad** use a GCPad profile; the screen marks a profile as
**(detected)** when a connected joystick matches its `Device` line. Use
**custom** if you mapped the pad in Dolphin's own GUI and want SlippiOS to leave
that section alone.

For deeper changes, **Settings → Expert mode → Expert tools → Open Dolphin
GUI** opens Dolphin's controller configuration. SlippiOS closes its UI first,
so the adapter is free for Dolphin.

## Bundled profiles

SlippiOS seeds these profiles into
`/data/users/default/dolphin/Config/Profiles/GCPad/` when needed (existing
files are not overwritten):

- SlippiOS: `Xbox 360 (Mad Catz)`, `Xbox 360 (Microsoft)`, `Keyboard`
- Slippi: B0XX_Linux, Frame1_Linux, Glyph, HayBox_DInput_Linux,
  HayBox_XInput_Linux

The Xbox profiles use evdev and the standard Xbox 360 button/axis layout. The
keyboard profile matches Dolphin's default Linux keyboard bindings.

A standard-pad profile selects the first matching evdev device. For four-player
local play, use the GameCube adapter, which provides all four ports natively.
Assigning the same pad profile to several ports reads the same physical pad.

`Frame1_Linux.ini` is derived from the upstream Ishiiruka Frame1 profile with
the DirectInput bindings translated to evdev names. The device is named
`evdev/0/Frame1`; if your unit enumerates under a different name, select the
device manually in Dolphin (the button and axis bindings stay valid).

Only Linux profiles are shipped. Sources: the user's Slippi install and
`JonnyHaystack/HayBox`. Load them from Dolphin's controller profile menu.
