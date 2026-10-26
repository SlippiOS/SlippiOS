# Branding

`SlippiLogo.54926e8ab7de3cabc854a38e5022d50f.svg` is the Slippi mark from
<https://slippi.gg/>, recoloured to Slippi green (`#44A963`).

It is a **temporary** stand-in until SlippiOS has its own mark. Both the Godot
splash (`apps/shell/assets/slippi-mark.png`) and the Plymouth theme use the
recoloured asset.

Regenerate the raster assets from the SVG:

```sh
rsvg-convert -w 640 -h 464 slippi-mark-green.svg -o ../../apps/shell/assets/slippi-mark.png
magick -size 3840x2160 xc:'#0b0b10' ../../apps/shell/assets/slippi-mark.png \
  -gravity center -composite ../../apps/shell/assets/slippi-boot.png
```
