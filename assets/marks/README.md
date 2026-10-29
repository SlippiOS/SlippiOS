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

`icon.png` is the 1024x1024 organization/project icon (dark background), and
`icon-light.png` is the light-background variant:

```sh
magick -background none logo.svg -resize 1024x1024 -trim +repage -resize 680x680 \
  -background '#0b0b10' -gravity center -extent 1024x1024 icon.png
magick -background none logo-light.svg -resize 1024x1024 -trim +repage -resize 680x680 \
  -background '#f5f6f8' -gravity center -extent 1024x1024 icon-light.png
```
