# Controller support

The Godot shell reads controllers through SDL3, including the GameCube adapter.
SDL3 has a native HIDAPI driver for the Wii U/Mayflash adapter, so no separate
uinput daemon is needed.

Wireless controllers are intentionally not supported. Melee is
latency-sensitive; use the wired GameCube adapter or a wired USB pad. Bluetooth
and `xpadneo` are not included in the image.

Press the **Guide** button on a standard pad to stop Melee and return to the
shell. The GameCube adapter is not a controller-interface device, so its
controllers have no Guide binding; use the console power button to shut down.

## Glyph profile

The Glyph reports on Linux as `Mad Catz Xbox 360 Controller`. SlippiOS ships
one Linux-only `Glyph` profile using evdev numbers and
`Device = evdev/0/Mad Catz Xbox 360 Controller`. There is no separate Windows
or SDL profile; stable Slippi (Ishiiruka) is evdev-only.

| Action | Dolphin binding |
|---|---|
| A / B / X / Y | `Button 0` / `Button 1` / `Button 2` / `Button 3` |
| Z | `Button 5` |
| Start | `Button 7` |
| L / R | `Axis 2+` / `Axis 5+` |
| Stick / C-stick | `Axis 0/1`, `Axis 3/4` |
| D-Pad | `Axis 6/7` |

Hold A during the boot screen to open the menu instead of launching Melee.
Keyboard Enter does the same for testing.

## GameCube adapter

`057e:0337` is the official Wii U/Mayflash adapter. The adapter switch must be
in **Wii U** mode.

While the shell is running, SDL3 reads the four controller ports directly.
Dolphin also supports the adapter natively through libusb. USB access is
exclusive, so when an adapter is present the shell wrapper quits Godot before
starting Dolphin and starts Godot again after Dolphin exits. With a standard
pad the shell stays on its launching screen and hides itself while Dolphin
runs. Do not run a uinput bridge at the same time.

Open **Settings → Dolphin → Gameplay → Controller Settings** to assign each port. Ports set to
**GameCube adapter** use Dolphin's native adapter support. Ports set to
**standard pad** use a GCPad profile; the screen marks a profile as
**(detected)** when a connected joystick matches its `Device` line. Use
**custom** if you mapped the pad in Dolphin's own GUI and want SlippiOS to leave
that section alone.

For deeper changes, **Settings → Advanced → Expert mode → Expert tools → Open Dolphin
GUI** opens Dolphin's controller configuration. SlippiOS closes its UI first,
so the adapter is free for Dolphin.

## Bundled profiles

SlippiOS seeds these profiles into
`/data/users/default/dolphin/Config/Profiles/GCPad/` when needed (existing
files are not overwritten):

- SlippiOS: `Xbox 360 (Microsoft)`, `Glyph`, `Keyboard`
- Slippi: B0XX_Linux, Frame1_Linux, HayBox_DInput_Linux,
  HayBox_XInput_Linux

All bundled profiles are Linux profiles. The Xbox 360 and Glyph profiles use
evdev and the standard Xbox 360 button/axis layout. The keyboard profile matches
Dolphin's default Linux keyboard bindings.

Profiles with a fixed Linux device line select that device. For four-player
local play, use the GameCube adapter, which provides all four ports natively.
Assigning the same pad profile to several ports reads the same physical pad.

`Frame1_Linux.ini` is derived from the upstream Ishiiruka Frame1 profile with
the DirectInput bindings translated to evdev names. The device is named
`evdev/0/Frame1`; if your unit enumerates under a different name, select the
device manually in Dolphin (the button and axis bindings stay valid).

Only Linux profiles are shipped. Sources: the user's Slippi install and
`JonnyHaystack/HayBox`. Load them from Dolphin's controller profile menu.
