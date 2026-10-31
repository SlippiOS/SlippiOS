# Marks

`icon.svg` is the current SlippiOS mark: the filled blue hexagon with the
dark S and blue outline. `logo.svg` is the earlier outline mark, with
`logo-light.svg` and `logo-grayscale.svg` as its light/grayscale variants;
`shine-colors.svg` is the original colored shine source.

`split-layers.py` splits `icon.svg` into the pulsing shine and the static S.
Plymouth uses it at build time, and the Godot boot splash uses committed
renders of both layers so only the shine moves. The game launch screen uses
the shine layer alone as the breathing hexagon.

Regenerate the shell assets:

```sh
python3 split-layers.py shine.svg s.svg icon.svg
rsvg-convert -w 1024 -h 1024 shine.svg -o ../../apps/shell/assets/slippi-mark-shine.png
rsvg-convert -w 1024 -h 1024 s.svg -o ../../apps/shell/assets/slippi-mark-s.png
rsvg-convert -w 1024 -h 1024 icon.svg -o ../../apps/shell/assets/slippi-mark.png
```

The X root wallpaper is not stored here: `modules/slippios/session.nix` renders
it for the common desktop sizes from `slippi-mark.png`, so the session can pick
an image that matches the display instead of scaling a 4K one down.

`icon.png` is the hand-made organization/project icon: the filled blue
hexagon with the solid S glyph. It has a transparent background. `icon.svg`
is the vector source for that icon. The shell uses the 640px `icon.png`
bake at `apps/shell/assets/slippi-os-icon.png` for the menu wordmark,
paired with "Slippi" set in Maven Pro. Godot's SVG importer aliases the
thin blue outline at menu size, so the shell uses the high-resolution bake
instead of the SVG directly.

Regenerate the shell bake after changing `icon.svg`:

```sh
inkscape icon.svg --export-type=png --export-filename=../../apps/shell/assets/slippi-os-icon.png --export-width=640
```

`icon-square.png` is the same mark centered on a 1024x1024 transparent canvas,
sized so GitHub's circular crop does not clip it. The black S needs a light
background to stay visible.
