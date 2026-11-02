# ADR-0004: Diagnose the beta Dolphin launch failure on the console

| | |
|---|---|
| **Status** | Proposed |
| **Date** | 2026-11-02 |
| **Deciders** | Ryan Wendell |

## Context

The stable Ishiiruka build launches on the console, but selecting the beta
Mainline build (`v4.0.0-mainline-beta.19`) leaves the console without a game
window. The console is an EM680 (Ryzen 7 6800U, Radeon 680M) running Mesa
26.1.8. The shell and stable build work on the same display and ISO.

The console logs show:

- `shell.log` has no launch error; the shell exits and restarts.
- `users/default/logs/dolphin.log` is empty.
- The beta `dolphin/Config/Dolphin.ini`, `GFX.ini`, and `Logger.ini` are 0 bytes.
- The beta AppImage is present and extracted under
  `cache/appimage-run/da21b8fbd08b3b8affc589104b1f9011fdb689e72229d8f9eabb607235a1bac3`.
- `game-ready` was written, so the window watcher saw a Dolphin window.

The same beta AppImage and ISO boot on the host and in the dev QEMU image
when the AppImage cache is freshly extracted. That rules out the AppImage
payload, the ISO, and the launch arguments.

## Proposed fix

Test two console-specific causes in order:

1. **Clear the beta AppImage cache.** Delete
   `cache/appimage-run/da21b8fbd08b3b8affc589104b1f9011fdb689e72229d8f9eabb607235a1bac3`
   and launch beta again. The cache is extracted on first launch; an
   interrupted extraction can leave a directory that looks complete but
   cannot run. A fresh extraction should reproduce the host/dev result.
2. **Try beta with OpenGL.** Settings → Dolphin Settings → Renderer: OGL,
   Build: beta. This isolates the console's AMD Vulkan path from the beta
   build.

If neither works, capture the console's `dolphin.log` and `journalctl -b`
after a beta launch and compare the Vulkan/EGL errors with the host run.

## Consequences

- If clearing the cache fixes it, the runner should validate extracted caches
  (or the image should pre-seed the beta cache) so a partial extraction cannot
  persist.
- If OpenGL fixes it, beta needs an OGL fallback on this GPU, or the default
  renderer should be OGL when the beta build is selected.
- Until the console is available, the status line reports the selected build
  separately from the installed beta tag so the UI does not imply beta is
  active when stable is selected.
