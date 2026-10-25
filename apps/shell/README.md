# SlippiOS shell

Godot 4 project for the console UI. It calls `slippios` for all system work;
JSON output is automatic when stdout is piped.

Run during development:

```sh
nix shell nixpkgs#godot_4 --command godot --path apps/shell
```

Export (needs Godot export templates):

```sh
godot --headless --path apps/shell \
  --export-release "Linux/X11" build/slippios-shell
```

Current scope: boot animation, status line, Play (launches Dolphin and returns),
update check, placeholders for Account/Settings.
