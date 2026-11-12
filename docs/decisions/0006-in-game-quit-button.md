# ADR-0006: Add an in-game Quit button without rebuilding Dolphin

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-11-02 |
| **Deciders** | Ryan Wendell |

## Context

Players need a controller-only way to leave Melee and return to SlippiOS. The
shell already maps the Guide button to Dolphin's Stop hotkey, but that only
works for standard pads, not the GameCube adapter. A visible **Quit** item in
Melee's main menu works with every controller.

SlippiOS ships prebuilt Slippi Dolphin AppImages for stable Ishiiruka 3.6.4 and
beta Mainline 4.0.0-mainline-beta.19. Adding a C++ EXI quit command would mean
building and distributing Dolphin ourselves, which the project does not do.
Melee's top-level menu has five fixed slots (1-P Mode, VS. Mode, Trophies,
Options, Data), so a sixth slot would require model and animation changes.

## Decision

Repurpose the **Data** slot as **Quit** and implement the exit path entirely in
the game and the launcher:

- A Gecko code hooks the main-menu think function at `0x8022DB40`. When the
  top-level cursor is on Data (`hovered_selection == 4`) and Confirm is
  pressed, it sends `CMD_LOG_MESSAGE` with `SLIPPIOS_QUIT` through the Slippi
  EXI device, then clears the button mask so the Data menu does not open.
- `slippios play` and `slippios-session` tail the captured Dolphin log for
  `SLIPPIOS_QUIT`. On a match they send `SIGTERM` to Dolphin's process group,
  escalating to `SIGKILL` after three seconds. The marker also turns the exit
  into a clean result so the shell returns to the menu instead of showing a
  launch error.
- The Data label is replaced with **Quit** by a Dolphin custom texture
  (`tex1_176x30_d1fa83d25e26b8a9_2.png`) installed under
  `Load/Textures/GALE01/SlippiOS/`. Hires textures are enabled in both builds.
- The Gecko code is installed as `Required: SlippiOS Quit Button` in the
  profile's local `GALE01.ini`. It is enabled by default and locked in the
  shell's Gecko screen so the label and behavior cannot drift apart.
- The Slippi log type is enabled so the marker reaches the launcher. Stable
  writes `Logger.ini`; beta passes the equivalent `-C` arguments.

## Consequences

- The Quit button works on both Dolphin builds and with every controller,
  including the GameCube adapter.
- The **Data** menu (trophies, archives, sound test) is no longer reachable
  from the main menu. This is the cost of using the fifth fixed slot. The
  decision can be revisited if Melee's menu model is extended.
- Custom textures must stay enabled. If a user disables `HiresTextures`, the
  slot still quits but is labelled **Data**.
- The launcher depends on the Slippi log marker. If Dolphin's logging changes,
  the Quit button would stop exiting; the Gecko code and texture would remain.
- The code is locked in the Gecko screen, so a user cannot disable the Quit
  button and leave a dead label behind.
