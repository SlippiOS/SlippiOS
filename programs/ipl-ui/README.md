# SlippiOS graphical IPL

Godot 4 project for the console UI. It calls `slippios-cli` for all system work;
JSON output is automatic when stdout is piped.

Run during development:

```sh
nix shell nixpkgs#godot_4 --command godot --path programs/ipl-ui
```

Export (needs Godot export templates):

```sh
godot --headless --path programs/ipl-ui \
  --export-release "Linux/X11" build/slippios-ipl
```

Current scope: boot animation, status line, Play (launches Dolphin and returns),
update check, placeholders for Account/Settings.
