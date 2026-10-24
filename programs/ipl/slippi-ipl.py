#!/usr/bin/env python3
"""SlippiOS Initial Program Loader.

Boot flow:
  * settings and game image present  -> launch Dolphin
  * neither present                  -> first-time setup
  * only one present / invalid       -> alert, then setup
"""

from __future__ import annotations

import getpass
import json
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path

import slippi_adapter
import slippi_auth
import slippi_input
import slippi_settings as settings_lib
import slippi_update


def clear() -> None:
    print("\033[2J\033[H", end="", flush=True)


def boot_animation() -> None:
    frames = ("      ◇", "    ◇ ◇", "  ◇   ◇", "◇  SLIPPIOS  ◇")
    for frame in frames:
        clear()
        print("\n\n\n                 " + frame)
        print("\n                 Initializing console…")
        time.sleep(0.55)


def alert(lines: list[str]) -> None:
    clear()
    print()
    for line in lines:
        print("                 " + line)
    print()
    input("                 Press Enter to continue…")


def choose_iso() -> str | None:
    """Pick a Melee image from known locations or ask for a path."""
    found = settings_lib.find_isos()
    valid = [(path, reason) for path, ok, reason in found if ok]

    if valid:
        print("\n                 Found game images:\n")
        for index, (path, _) in enumerate(valid, start=1):
            print(f"                 [{index}] {path}")
        choice = input("\n                 Select [1]: ").strip()
        if not choice:
            return str(valid[0][0])
        if choice.isdigit() and 1 <= int(choice) <= len(valid):
            return str(valid[int(choice) - 1][0])
        print("                 Unrecognized choice.")
    else:
        print("\n                 No valid Melee image found.")
        print(f"                 Put an NTSC 1.02 ISO in {settings_lib.ISO_DIR}/")

    while True:
        path = input("\n                 Path to Melee ISO (blank to skip): ").strip()
        if not path:
            return None
        ok, reason = settings_lib.iso_status(path)
        if ok:
            return path
        print(f"                 {reason}")


def configure_network(settings: dict) -> None:
    """Apply Wi-Fi settings without blocking the boot for long."""
    ssid = settings["network"].get("ssid", "").strip()
    psk = settings["network"].get("psk", "")
    if not ssid or not shutil.which("nmcli"):
        return

    try:
        state = subprocess.run(
            ["nmcli", "-t", "-f", "STATE", "general"],
            capture_output=True, text=True, timeout=5,
        ).stdout.strip()
    except (OSError, subprocess.TimeoutExpired):
        return
    if state.startswith("connected"):
        return

    print(f"\n                 Connecting to Wi-Fi “{ssid}”…")
    try:
        subprocess.run(
            ["nmcli", "--wait", "20", "device", "wifi", "connect", ssid, "password", psk],
            capture_output=True, timeout=30, check=False,
        )
    except (OSError, subprocess.TimeoutExpired):
        pass


def onboarding(reason: str | None = None) -> dict | None:
    clear()
    print("\n                 SLIPPIOS SETUP\n")
    print("                 Melee 1.02 (NTSC) is required. SlippiOS does not")
    print("                 include or download game images.\n")
    if reason:
        print("                 " + reason + "\n")

    iso = choose_iso()
    if iso is None:
        return None

    ssid = input("\n                 Wi-Fi SSID (blank to skip): ").strip()
    psk = ""
    if ssid:
        psk = input("                 Wi-Fi password: ").strip()

    settings = settings_lib.defaults()
    chosen = Path(iso)
    try:
        settings["game"]["iso"] = str(chosen.relative_to(settings_lib.ISO_DIR))
    except ValueError:
        settings["game"]["iso"] = str(chosen)
    settings["network"]["ssid"] = ssid
    settings["network"]["psk"] = psk
    settings_lib.save(settings)
    print(f"\n                 Wrote {settings_lib.SETTINGS_FILE}")
    time.sleep(1)
    return settings


def apply_display(settings: dict) -> None:
    """Set the requested resolution/refresh rate on the connected output."""
    if not shutil.which("xrandr"):
        return
    try:
        query = subprocess.run(
            ["xrandr", "--query"], capture_output=True, text=True, timeout=5,
        ).stdout
    except (OSError, subprocess.TimeoutExpired):
        return

    output = None
    for line in query.splitlines():
        if " connected" in line:
            output = line.split()[0]
            if " primary" in line:
                break
    if not output:
        return

    resolution = str(settings["display"].get("resolution", "auto"))
    rate = settings["display"].get("refresh_hz")
    command = ["xrandr", "--output", output]
    if resolution and resolution != "auto":
        command += ["--mode", resolution]
    if rate:
        command += ["--rate", str(rate)]
    if len(command) == 3:
        return
    try:
        subprocess.run(command, capture_output=True, timeout=10)
    except (OSError, subprocess.TimeoutExpired):
        pass


def apply_audio(settings: dict) -> None:
    """Select the configured audio sink when one is named."""
    sink = settings["audio"].get("sink", "auto")
    if sink in ("", "auto") or not shutil.which("pactl"):
        return
    try:
        subprocess.run(
            ["pactl", "set-default-sink", sink], capture_output=True, timeout=5,
        )
    except (OSError, subprocess.TimeoutExpired):
        pass


def ensure_login(settings: dict) -> tuple[bool, str]:
    """Prompt for Slippi credentials and write the login file."""
    ok, detail = settings_lib.install_login(settings)
    if ok:
        return True, detail

    for _ in range(3):
        clear()
        print("\n                 SLIPPI ACCOUNT\n")
        email = input("                 Email (blank to skip): ").strip()
        if not email:
            return False, "skipped"
        password = getpass.getpass("                 Password: ")
        try:
            user = slippi_auth.login(email, password)
        except slippi_auth.AuthError as exc:
            print(f"\n                 {exc}")
            time.sleep(2)
            continue

        target = settings_lib.SLIPPI_DIR / settings_lib.DEFAULT_USER_JSON
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(json.dumps(user, indent=2))
        target.chmod(0o600)
        return settings_lib.install_login(settings)

    return False, "sign-in failed"


def ensure_dolphin() -> Path | None:
    clear()
    print("\n\n                 Checking for Slippi Dolphin updates…\n")
    try:
        appimage = slippi_update.ensure(
            log=lambda message: print("                 " + message, flush=True)
        )
    except slippi_update.UpdateError as exc:
        alert(["Could not prepare Slippi Dolphin.", str(exc)])
        return None
    return appimage


def launch(settings: dict, appimage: Path) -> None:
    iso = str(settings_lib.iso_path(settings["game"]["iso"]))
    clear()
    print("\n\n                 Launching Slippi Dolphin…\n")
    print(f"                 {iso}")

    command = [
        "slippi-appimage-run", str(appimage),
        "-u", str(settings_lib.DOLPHIN_DIR),
        "-C", "Dolphin.Display.Fullscreen=True",
        "-C", "Graphics.Hardware.VSync=False",
        "-b",
        "-e", iso,
    ]

    log_path = settings_lib.SLIPPI_DIR / "logs" / "dolphin.log"
    log_path.parent.mkdir(parents=True, exist_ok=True)

    environment = os.environ.copy()
    runtime_dir = Path(f"/run/user/{os.getuid()}")
    if runtime_dir.is_dir():
        environment.setdefault("XDG_RUNTIME_DIR", str(runtime_dir))

    try:
        with log_path.open("w") as log:
            result = subprocess.run(
                command, stdout=log, stderr=subprocess.STDOUT, env=environment,
            )
    except FileNotFoundError:
        alert(["slippi-appimage-run is not installed in this image."])
        return

    if result.returncode != 0:
        try:
            tail = log_path.read_text(errors="replace").strip().splitlines()[-4:]
        except OSError:
            tail = []
        alert([
            f"Dolphin exited with status {result.returncode}.",
            f"Log: {log_path}",
            *tail,
        ])


def main() -> None:
    boot_animation()

    pads = slippi_input.Gamepads()
    try:
        held = slippi_adapter.a_held(1.5)
    except Exception:
        held = None
    held = bool(held) or pads.any_south_held()
    pads.close()
    if held:
        print("\n                 Controller A held at boot.")
        print("                 Configuration menu is the next milestone.")
        time.sleep(1.5)

    if settings_lib.check_hardware_binding():
        alert([
            "This cartridge was used on different hardware.",
            "The saved Slippi login has been removed.",
            "Sign in again to play online.",
        ])

    have_settings = settings_lib.SETTINGS_FILE.is_file()
    settings, error = settings_lib.load() if have_settings else (None, None)

    if not have_settings and not settings_lib.any_valid_iso():
        settings = onboarding("First-time setup: no settings and no game image.")
    elif not have_settings:
        alert([
            "Settings file is missing.",
            f"Expected {settings_lib.SETTINGS_FILE}",
            "An existing game image was found, but SlippiOS is not configured.",
        ])
        settings = onboarding("Setup: settings file missing.")
    elif error is not None:
        alert(["Settings file is invalid.", error])
        settings = onboarding("Setup: settings file invalid.")
    else:
        ok, reason = settings_lib.iso_status(settings["game"]["iso"])
        if not ok:
            alert(["Game image problem.", reason])
            settings = onboarding("Setup: game image missing or invalid.")

    if settings is None:
        alert(["Setup was not completed.", "Falling back to a shell."])
        return

    configure_network(settings)

    apply_display(settings)
    apply_audio(settings)

    logged_in, detail = ensure_login(settings)
    if logged_in:
        print(f"\n                 Slippi login: {detail}")
    else:
        print("\n                 No Slippi login; online play unavailable.")

    appimage = ensure_dolphin()
    if appimage is None:
        return
    launch(settings, appimage)


if __name__ == "__main__":
    try:
        main()
    except (EOFError, KeyboardInterrupt):
        sys.exit(0)
