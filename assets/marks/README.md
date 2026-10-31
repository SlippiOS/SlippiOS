# Marks

`logo.svg` is the SlippiOS mark: the blue shine hexagon plus a neutral S
(white for dark backgrounds). `logo-light.svg` uses a dark S for light
backgrounds. `logo-grayscale.svg` is the grayscale master, and
`shine-colors.svg` is the original colored shine source. The S is a path, not
live text; there is no font dependency.

`split-layers.py` splits `logo.svg` into the pulsing shine and the static S.
Plymouth uses it at build time, and the Godot boot splash uses committed
renders of both layers so only the shine moves.

Regenerate the shell assets:

```sh
python3 split-layers.py shine.svg s.svg logo.svg
rsvg-convert -w 1024 -h 1024 shine.svg -o ../../apps/shell/assets/slippi-mark-shine.png
rsvg-convert -w 1024 -h 1024 s.svg -o ../../apps/shell/assets/slippi-mark-s.png
rsvg-convert -w 1024 -h 1024 logo.svg -o ../../apps/shell/assets/slippi-mark.png
```

The X root wallpaper is not stored here: `modules/slippios/session.nix` renders
it for the common desktop sizes from `slippi-mark.png`, so the session can pick
an image that matches the display instead of scaling a 4K one down.

`icon.png` is the hand-made organization/project icon: the filled blue
hexagon with the solid S glyph. It has a transparent background. `icon.svg`
is the vector source for that icon. The shell uses a copy of `icon.svg` at
`apps/shell/assets/slippi-os-icon.svg` for the menu wordmark, paired with
"Slippi" set in Maven Pro.

`icon-square.png` is the same mark centered on a 1024x1024 transparent canvas,
sized so GitHub's circular crop does not clip it. The black S needs a light
background to stay visible.
