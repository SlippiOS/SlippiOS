# Onboarding and launch flow

**One flow from first boot to Melee, with a way back from every step.**

|            |                                                              |
| ---------- | ------------------------------------------------------------ |
| **Status** | Draft, input wanted                                          |
| **Owner**  | Shell and console                                            |
| **Scope**  | First boot, controller setup, game selection, launch, upload |
| **Audience** | Shell implementers and hardware testers                    |

## 1. Summary

The shell should have one predictable path: detect the controller, choose a
game, optionally connect the network, then play. Play must not bounce through
Settings. A failed or skipped step must be recoverable from Settings, and the
shell must stay visible until Melee has a focused, fullscreen window.

## 2. Current behavior

Observed on dev image `0.3.0.2vfi1nn`:

- A fresh profile defaults all four controller ports to the GameCube adapter.
  A standard pad is not read until Port 1 is changed.
- Skipping onboarding leaves `settings.toml` missing. Controller commands
  fail until a default file is created.
- Play with no game opens the ROM list, then the controller step. The
  controller step focuses **Controller settings** first, so Enter opens
  Settings instead of continuing.
- The stable Dolphin launcher frame stays mapped at 640x430 while the
  fullscreen render window is behind it. The visible result is a small
  launcher in the top half of the screen.
- Escape only works when the render window has keyboard focus. The shell
  minimizes without focusing the game window, so a physical Escape can go
  nowhere.
- The first launch extracts the stable AppImage and runs `ldconfig` on the
  data partition. This takes tens of seconds; later launches are fast.

## 3. Proposed flow

### 3.1 First boot

1. Boot splash. Holding Enter or A opens the menu instead of autoplay.
2. **Controller.** Show the detected pad and the profile that will be applied.
   Continue is the default action. Controller settings is secondary.
3. **Game.** Scan `MELEE-ROMS` and `/data/isos`. The user can select an image,
   upload one from a browser, or skip.
4. **Network.** Wired Ethernet is preferred. Wi-Fi and Slippi sign-in are
   optional; offline play is valid.
5. **Done.** Show the menu. Do not autoplay Melee on the first completed setup.

### 3.2 Play

- If no game is selected, open the game step.
- If no controller is configured, open the controller step.
- Otherwise launch immediately and show the launching screen.
- Play never opens Settings.

### 3.3 Settings

- **Run setup again** is available in expert mode and reopens the onboarding
  steps. It is a recovery path, not a second settings tree.
- **Controller Settings** remains the place to override ports and profiles.
- **Melee ROMs** always offers **Upload ISO**, whether or not onboarding is
  complete.

### 3.4 Launch

1. Show **LAUNCHING** with the game title and an animated spinner.
2. Start `slippios play --ready-file` and keep the shell visible.
3. When the Dolphin window is mapped, hide the non-fullscreen launcher frame,
   activate the fullscreen render window, and exit the shell with
   `GAME_READY_EXIT`.
4. The wrapper waits for the `slippios play` process, then restarts the shell
   at the menu.
5. Escape exits Melee from the focused game window.

## 4. Decisions

> **Why & What - Controller detection is automatic, confirmation is explicit**
>
> **What:** Detect standard pads on startup and assign empty ports, but show
> the result in onboarding and keep manual overrides in Controller Settings.
>
> **Why:** A fresh image must not require a hidden manual step. An explicit
> confirmation keeps the assignment visible and reversible. This does not
> solve unusual pads with no matching profile; those fall back to `custom`.
>
> **Alternatives considered:**
> - *Require manual setup on first boot:* it is predictable, but a user with
>   one standard pad sees a dead controller before reaching the setting.
> - *Never auto-assign:* it avoids surprises, but makes the default image
>   unusable with a standard pad.
>
> **Fallback:** The controller step can be reopened from Settings and the
> profile can be changed per port.

> **Why & What - The shell covers launch, then exits**
>
> **What:** The shell shows the launching screen, starts Dolphin itself, and
> exits only after the game window is mapped. The wrapper waits for the game
> process and restarts the shell at the menu.
>
> **Why:** The desktop gap is the difference between a console and a desktop
> with a game on it. Exiting after the window is mapped keeps the shell out of
> memory during gameplay without exposing the desktop. This does not solve the
> cold AppImage extraction time; it only covers it.
>
> **Alternatives considered:**
> - *Keep the shell minimized during Melee:* no restart cost, but the shell
>   process stays resident and the user sees a second window in the stack.
> - *Exit the shell immediately and let the wrapper launch Dolphin:* simple,
>   but exposes the desktop and gives the user no status.
> - *A separate splash process:* works for the adapter handoff, but adds a
>   second UI process and a second visual language.
>
> **Fallback:** The GameCube adapter path still exits the shell before Dolphin
> opens the adapter. The wrapper launches Dolphin and relaunches the shell.

> **Why & What - ISO upload is always available**
>
> **What:** The ROM screen starts the `iso` web mode at any time, not only
> during first boot.
>
> **Why:** Users add images after setup, and a missing image is the most common
> recovery case. The upload page must be upload-only, not a sign-in page.
>
> **Alternatives considered:**
> - *Only expose upload during onboarding:* less code, but forces a full setup
>   rerun to add a second image.
>
> **Fallback:** Copying an ISO to `MELEE-ROMS` from a PC remains supported.

## 5. Acceptance criteria

- A fresh image with one standard pad and one ISO reaches Melee without
  opening Settings.
- Skipping onboarding and rebooting still allows Controller Settings to open
  and save.
- Play with no game opens the game step; Play with a game launches directly.
- The controller step's default action is Continue.
- The launching screen remains visible until the game window is mapped.
- The game window is fullscreen and focused; Escape exits to the menu.
- Melee ROMs offers Upload ISO at any time and returns to the ROM list after
  an upload.
- Expert mode exposes Run setup again.

## 6. Open questions

- Should the first completed setup autoplay Melee, or always show the menu?
  Current proposal: show the menu.
- Should the GameCube adapter path also get a launching screen, at the cost of
  a separate splash process?
- How should an unrecognized pad be presented in the controller step? The
  current fallback is `custom`, which may leave the pad unmapped until the user
  opens Dolphin's GUI.

## 7. Costs and risks

- The shell exits during Melee. The wrapper must wait for the game process;
  the shell writes its PID before exiting. A stale PID or a crash can leave the
  wrapper waiting, so the PID file is removed before each launch and after the
  wait.
- The GameCube adapter remains on the exit-before-launch path because the
  adapter is exclusive USB.
- Hiding the Dolphin launcher frame uses X11 window inspection. A future
  Dolphin update that changes window titles or states must update that check.
- The cold AppImage extraction remains slow. Pre-seeding the extracted cache
  would remove it, at the cost of image size and a pinned cache hash.
- The upload page requires a working LAN address. Offline consoles still need
  the USB copy path.
