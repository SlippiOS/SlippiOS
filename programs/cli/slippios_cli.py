#!/usr/bin/env python3
"""slippios-cli: JSON interface between the IPL UI and system logic."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

import slippi_auth
import slippi_settings as settings_lib
import slippi_update

RUNNER = "slippi-appimage-run"


def emit(payload: dict, code: int = 0) -> int:
    print(json.dumps(payload, sort_keys=True))
    return code


def _settings_summary() -> tuple[dict, dict | None]:
    exists = settings_lib.SETTINGS_FILE.is_file()
    settings, error = settings_lib.load() if exists else (None, None)
    summary: dict = {"exists": exists, "path": str(settings_lib.SETTINGS_FILE)}
    if error:
        summary["error"] = error
    if settings:
        raw = settings["game"]["iso"]
        resolved = settings_lib.iso_path(raw)
        valid, reason = settings_lib.iso_status(raw)
        summary.update({
            "iso": raw,
            "iso_path": str(resolved),
            "iso_valid": valid,
            "iso_reason": reason,
            "refresh_hz": settings["display"].get("refresh_hz"),
            "resolution": settings["display"].get("resolution"),
        })
    return summary, settings


def cmd_status(_args) -> int:
    summary, settings = _settings_summary()

    login_ok, login_detail = settings_lib.login_status(settings_lib.DOLPHIN_USER_FILE)

    current = None
    try:
        if slippi_update.current_path():
            current = slippi_update.current_tag()
    except OSError:
        current = None

    try:
        import usb.core
        adapter = usb.core.find(
            idVendor=slippi_adapter.VID, idProduct=slippi_adapter.PID
        ) is not None
    except Exception:
        adapter = False

    return emit({
        "ok": True,
        "settings": summary,
        "login": {"signed_in": login_ok, "detail": login_detail},
        "dolphin": {"current": current},
        "adapter": {"present": adapter},
    })


def cmd_login(args) -> int:
    email = args.email
    password = args.password
    if args.stdin:
        try:
            payload = json.load(sys.stdin)
        except ValueError as exc:
            return emit({"ok": False, "error": f"invalid JSON: {exc}"}, 2)
        email = payload.get("email", email)
        password = payload.get("password", password)

    if not email or not password:
        return emit({"ok": False, "error": "email and password are required"}, 2)

    try:
        user = slippi_auth.login(email, password)
    except slippi_auth.AuthError as exc:
        return emit({"ok": False, "error": str(exc)}, 1)

    target = settings_lib.SLIPPI_DIR / settings_lib.DEFAULT_USER_JSON
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(user, indent=2))
    target.chmod(0o600)
    installed, detail = settings_lib.install_login(settings_lib.defaults())

    return emit({
        "ok": installed,
        "connect_code": user.get("connectCode"),
        "display_name": user.get("displayName"),
        "detail": detail,
    }, 0 if installed else 1)


def cmd_update(args) -> int:
    if args.check:
        try:
            tag, _asset = slippi_update._latest_release()
        except Exception as exc:
            return emit({"ok": False, "error": str(exc)}, 1)
        return emit({
            "ok": True,
            "current": slippi_update.current_tag(),
            "latest": tag,
        })

    try:
        path = slippi_update.ensure(log=lambda message: print(message, file=sys.stderr))
    except slippi_update.UpdateError as exc:
        return emit({"ok": False, "error": str(exc)}, 1)

    return emit({
        "ok": True,
        "current": slippi_update.current_tag(),
        "path": str(path),
    })


def cmd_play(args) -> int:
    shared = slippi_settings.defaults()
    login_ok, _detail = settings_lib.install_login(shared)
    if not login_ok and args.require_login:
        return emit({"ok": False, "error": "not signed in"}, 1)

    try:
        appimage = slippi_update.ensure(log=lambda message: print(message, file=sys.stderr))
    except slippi_update.UpdateError as exc:
        return emit({"ok": False, "error": str(exc)}, 1)

    summary, settings = _settings_summary()
    if not settings or not summary.get("iso_valid"):
        return emit({"ok": False, "error": summary.get("iso_reason", "no game image")}, 1)

    iso = str(settings_lib.iso_path(settings["game"]["iso"]))
    environment = os.environ.copy()
    runtime = Path(f"/run/user/{os.getuid()}")
    if runtime.is_dir():
        environment.setdefault("XDG_RUNTIME_DIR", str(runtime))

    command = [
        RUNNER, str(appimage),
        "-u", str(settings_lib.DOLPHIN_DIR),
        "-C", "Dolphin.Display.Fullscreen=True",
        "-C", "Graphics.Hardware.VSync=False",
        "-b", "-e", iso,
    ]
    print(json.dumps({"ok": True, "event": "launching", "iso": iso}), flush=True)
    result = subprocess.run(command, env=environment)
    return emit({"ok": result.returncode == 0, "returncode": result.returncode})


def cmd_adapter(args) -> int:
    try:
        import slippi_adapter
        held = slippi_adapter.a_held(args.seconds)
    except Exception as exc:
        return emit({"ok": False, "error": str(exc)}, 1)
    return emit({"ok": True, "present": held is not None, "a_held": held})


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(prog="slippios-cli")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("status", help="report settings, login, Dolphin and adapter state")

    login = sub.add_parser("login", help="authenticate and install user.json")
    login.add_argument("--email")
    login.add_argument("--password")
    login.add_argument("--stdin", action="store_true", help="read {email,password} JSON from stdin")

    update = sub.add_parser("update", help="check or apply a Dolphin update")
    update.add_argument("--check", action="store_true")

    play = sub.add_parser("play", help="launch Dolphin with the configured game")
    play.add_argument("--require-login", action="store_true")

    adapter = sub.add_parser("adapter", help="report adapter presence and A button")
    adapter.add_argument("--seconds", type=float, default=1.0)

    return parser


def main() -> int:
    args = build_parser().parse_args()
    handlers = {
        "status": cmd_status,
        "login": cmd_login,
        "update": cmd_update,
        "play": cmd_play,
        "adapter": cmd_adapter,
    }
    return handlers[args.command](args)


if __name__ == "__main__":
    sys.exit(main())
