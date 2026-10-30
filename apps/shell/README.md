# SlippiOS shell

Godot 4 project for the console UI. It calls `slippios` for all system work;
JSON output is automatic when stdout is piped. The image builds this project
into a single compiled binary with an embedded PCK; the `.gd` sources are not
installed in the image.

## Layout

- `main.gd`: app state, boot flow, screen routing, and update orchestration.
- `screens/`: one screen per flow (menu, ROMs, setup, account, settings, and so on).
- `services/`: CLI wrapper, browser setup server, OS update job, UI builders,
  status reads, and OS hand-off actions.
- `widgets/keyboard.gd`: shared on-screen keyboard.
- `theme.gd`: reads `/etc/slippios/theme.json`.
- `boot_logo.gd`: pulsing boot mark.

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

## UI conventions

- Build screens with the helpers in `screens/screen.gd` (`title`, `label`,
  `section`, `button`).
- Group related buttons with `section("Name")`. Section rows are labels, so
  gamepad focus skips them.
- Confirmations and errors are toasts: call `toast("Saved.", "success")` or
  `toast("Could not save.", "error")` before navigating. Do not add inline
  message labels to a screen.
- Screens are rebuilt on navigation. `ui.gd` restores focus to the button with
  the same key (the text before `:`) when the same screen is presented again.

## Runtime contract

- `slippios` must be on `PATH` and return JSON when stdout is piped.
- Theme colors are read from `/etc/slippios/theme.json` when present.
- Optional environment:
  - `SLIPPIOS_PARSE_ONLY=1`: import/parse only, no CLI or display calls.
  - `SLIPPIOS_SKIP_SPLASH=1`: skip the pulsing boot mark.
  - `SLIPPIOS_AUTOPLAY=0`: show the menu instead of launching immediately.
  - `SLIPPIOS_DATA`: data root used by the browser upload page (default
    `/data`).
- Exit codes are a contract with the OS session wrapper:
  - `0`: quit normally
  - `42`: launch Slippi Dolphin
  - `43`: open Dolphin settings
  - `44`: open a recovery terminal
  - `45`: edit `settings.toml`
  - anything else: crash
