# SlippiOS shell

Godot 4 project for the console UI. It calls `slippios` for all system work;
JSON output is automatic when stdout is piped.

## Layout

- `main.gd` — app state, boot flow, screen routing, system actions
- `screens/` — one screen per flow (menu, ROMs, setup, account, settings, …)
- `services/` — CLI wrapper, browser setup server, OS update job
- `widgets/keyboard.gd` — shared on-screen keyboard
- `theme.gd` — reads `/etc/slippios/theme.json`
- `boot_logo.gd` — pulsing boot mark

## Run

From this directory:

```sh
nix shell nixpkgs#godot_4 --command godot --path .
```

## Test

```sh
SLIPPIOS_PARSE_ONLY=1 godot --headless --path . --import
SLIPPIOS_PARSE_ONLY=1 godot --headless --path . --quit
```

`SLIPPIOS_PARSE_ONLY=1` skips the CLI and display probing.

## Export

The image build exports the `Linux` preset with the nixpkgs Godot export
template. For a manual export with templates installed:

```sh
godot --headless --path . --export-release "Linux" build/slippios-shell
```
