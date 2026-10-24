#!/usr/bin/env python3
"""SlippiOS Initial Program Loader.

Boot flow:
  * settings and game image present  -> launch Dolphin
  * neither present                  -> first-time setup
  * only one present / invalid       -> alert, then setup
"""

from __future__ import annotations

import shutil
import subprocess
import sys
import time
from pathlib import Path

import slippi_settings as settings_lib


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
    settings["game"]["iso"] = iso
    settings["network"]["ssid"] = ssid
    settings["network"]["psk"] = psk
    settings_lib.save(settings)
    print(f"\n                 Wrote {settings_lib.SETTINGS_FILE}")
    time.sleep(1)
    return settings


def launch(settings: dict) -> None:
    iso = settings["game"]["iso"]
    clear()
    print("\n\n                 Launching Slippi Dolphin…\n")
    print(f"                 {iso}")

    command = [
        "slippi-dolphin",
        "-u", str(settings_lib.DOLPHIN_DIR),
        "-b",
        "-e", iso,
    ]
    try:
        result = subprocess.run(command)
    except FileNotFoundError:
        alert(["slippi-dolphin is not installed."])
        return
    if result.returncode != 0:
        alert([f"Dolphin exited with status {result.returncode}."])


def main() -> None:
    boot_animation()

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
    launch(settings)


if __name__ == "__main__":
    try:
        main()
    except (EOFError, KeyboardInterrupt):
        sys.exit(0)
