#!/usr/bin/env python3
"""slippios: human-first system control, JSON for the graphical shell."""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

import auth
import settings as settings_lib
import update

RUNNER = "slippi-appimage-run"


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


def cmd_status(_args) -> dict:
    summary, _settings = _settings_summary()
    login_ok, login_detail = settings_lib.login_status(settings_lib.DOLPHIN_USER_FILE)

    current = None
    try:
        if update.current_path():
            current = update.current_tag()
    except OSError:
        current = None

    try:
        import usb.core
        adapter = usb.core.find(
            idVendor=0x057E, idProduct=0x0337
        ) is not None
    except Exception:
        adapter = False

    return {
        "ok": True,
        "settings": summary,
        "login": {"signed_in": login_ok, "detail": login_detail},
        "dolphin": {"current": current},
        "adapter": {"present": adapter},
    }


def cmd_login(args) -> dict:
    email = args.email
    password = args.password
    if args.file:
        try:
            payload = json.loads(Path(args.file).read_text())
        except (OSError, ValueError) as exc:
            return {"ok": False, "error": f"cannot read credentials: {exc}"}
        email = payload.get("email", email)
        password = payload.get("password", password)
    elif args.stdin:
        try:
            payload = json.load(sys.stdin)
        except ValueError as exc:
            return {"ok": False, "error": f"invalid JSON: {exc}"}
        email = payload.get("email", email)
        password = payload.get("password", password)

    if not email or not password:
        return {"ok": False, "error": "email and password are required"}

    try:
        user = auth.login(email, password)
    except auth.AuthError as exc:
        return {"ok": False, "error": str(exc)}

    target = settings_lib.PROFILE_DIR / settings_lib.DEFAULT_USER_JSON
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(user, indent=2))
    target.chmod(0o600)
    installed, detail = settings_lib.install_login(settings_lib.defaults())

    return {
        "ok": installed,
        "connect_code": user.get("connectCode"),
        "display_name": user.get("displayName"),
        "detail": detail,
        **({} if installed else {"error": detail}),
    }


def cmd_update(args) -> dict:
    if args.check:
        try:
            tag, _asset = update._latest_release()
        except Exception as exc:
            return {"ok": False, "error": str(exc)}
        return {"ok": True, "check": True,
                "current": update.current_tag(), "latest": tag}

    try:
        path = update.ensure(log=lambda message: print(message, file=sys.stderr))
    except update.UpdateError as exc:
        return {"ok": False, "error": str(exc)}

    return {"ok": True, "check": False,
            "current": update.current_tag(), "path": str(path)}


def cmd_play(args) -> dict:
    login_ok, _detail = settings_lib.install_login(settings_lib.defaults())
    if not login_ok and args.require_login:
        return {"ok": False, "error": "not signed in"}

    try:
        appimage = update.ensure(
            log=lambda message: print(message, file=sys.stderr)
        )
    except update.UpdateError as exc:
        return {"ok": False, "error": str(exc)}

    summary, settings = _settings_summary()
    if not settings or not summary.get("iso_valid"):
        return {"ok": False, "error": summary.get("iso_reason", "no game image")}

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
    result = subprocess.run(command, env=environment)
    return {"ok": result.returncode == 0, "iso": iso,
            "returncode": result.returncode}


def cmd_isos(_args) -> dict:
    found = settings_lib.find_isos()
    entries = [
        {
            "path": str(path),
            "name": path.name,
            "valid": ok,
            "reason": reason,
        }
        for path, ok, reason in found
    ]
    return {"ok": True, "isos": entries}


def cmd_settings(args) -> dict:
    settings, error = settings_lib.load()
    if settings is None:
        if error and settings_lib.SETTINGS_FILE.is_file():
            return {"ok": False, "error": error}
        settings = settings_lib.defaults()

    if args.action == "show":
        return {"ok": True, "settings": settings}

    key, value = args.key, args.value
    section, _, name = key.partition(".")
    if not section or not name or section not in settings or not isinstance(settings[section], dict):
        return {"ok": False, "error": f"unknown setting {key}"}

    if name == "iso":
        path = settings_lib.iso_path(value)
        try:
            value = str(path.relative_to(settings_lib.ISO_DIR))
        except ValueError:
            value = str(path)

    if name == "refresh_hz":
        try:
            value = int(value)
        except ValueError:
            return {"ok": False, "error": "refresh_hz must be a number"}

    settings[section][name] = value
    settings_lib.save(settings)
    return {"ok": True, "settings": settings}


def cmd_net(args) -> dict:
    import net
    net.serve(args.token, args.port)
    return {"ok": True}


def cmd_adapter(args) -> dict:
    try:
        import adapter
        held = adapter.a_held(args.seconds)
    except Exception as exc:
        return {"ok": False, "error": str(exc)}
    return {"ok": True, "present": held is not None, "a_held": held}


def human_status(payload: dict) -> None:
    settings = payload["settings"]
    state = "ok" if settings.get("exists") else "missing"
    print(f"Settings:  {settings['path']} ({state})")
    if settings.get("iso"):
        game = "ok" if settings.get("iso_valid") else settings.get("iso_reason", "invalid")
        print(f"Game:      {settings['iso_path']} ({game})")
        if settings.get("refresh_hz"):
            print(f"Display:   {settings['refresh_hz']} Hz")
    else:
        print("Game:      not configured")
    login = payload["login"]
    print(f"Login:     signed in ({login['detail']})" if login["signed_in"]
          else "Login:     not signed in")
    print(f"Dolphin:   {payload['dolphin']['current'] or 'not installed'}")
    print(f"Adapter:   {'connected' if payload['adapter']['present'] else 'not connected'}")


def human_update(payload: dict) -> None:
    if payload.get("check"):
        print(f"Current:   {payload.get('current') or 'not installed'}")
        print(f"Latest:    {payload.get('latest')}")
    else:
        print(f"Dolphin:   {payload.get('current')}")
        print(f"Path:      {payload.get('path')}")


def human_login(payload: dict) -> None:
    name = payload.get("display_name") or "player"
    print(f"Signed in as {name} ({payload.get('connect_code')})")


def human_play(payload: dict) -> None:
    if payload.get("ok"):
        print("Dolphin exited cleanly.")
    else:
        print(f"Dolphin exited with status {payload.get('returncode')}.")


def human_adapter(payload: dict) -> None:
    if not payload.get("present"):
        print("GameCube adapter: not connected")
        return
    held = payload.get("a_held")
    print(f"GameCube adapter: connected, A {'held' if held else 'not held'}")


def human_isos(payload: dict) -> None:
    if not payload["isos"]:
        print("No game images found in /data/isos.")
        return
    for entry in payload["isos"]:
        state = "ok" if entry["valid"] else entry["reason"]
        print(f"{entry['name']}: {state}")


def human_settings(payload: dict) -> None:
    settings = payload["settings"]
    print(f"Game:      {settings['game'].get('iso')}")
    print(f"Display:   {settings['display'].get('refresh_hz')} Hz, "
          f"{settings['display'].get('resolution')}")
    print(f"Audio:     {settings['audio'].get('sink')}")
    ssid = settings['network'].get('ssid') or "(none)"
    print(f"Network:   {ssid}")


def build_parser() -> argparse.ArgumentParser:
    common = argparse.ArgumentParser(add_help=False)
    common.add_argument("--json", action="store_true", default=argparse.SUPPRESS)
    common.add_argument("--human", action="store_true", default=argparse.SUPPRESS)

    parser = argparse.ArgumentParser(prog="slippios")
    parser.add_argument("--json", action="store_true",
                        help="force machine-readable output")
    parser.add_argument("--human", action="store_true",
                        help="force human-readable output")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("status", parents=[common],
                   help="settings, login, Dolphin and adapter state")

    login = sub.add_parser("login", parents=[common],
                           help="authenticate and install user.json")
    login.add_argument("--email")
    login.add_argument("--password")
    login.add_argument("--stdin", action="store_true",
                       help="read {email,password} JSON from stdin")
    login.add_argument("--file", help="read {email,password} JSON from a file")

    update = sub.add_parser("update", parents=[common],
                            help="check or apply a Dolphin update")
    update.add_argument("--check", action="store_true")

    play = sub.add_parser("play", parents=[common],
                          help="launch Dolphin with the configured game")
    play.add_argument("--require-login", action="store_true")

    adapter = sub.add_parser("adapter", parents=[common],
                             help="adapter presence and A button")
    adapter.add_argument("--seconds", type=float, default=1.0)

    sub.add_parser("isos", parents=[common], help="list discovered game images")

    net_parser = sub.add_parser("net", parents=[common],
                                help="serve LAN sign-in and ISO upload pages")
    net_parser.add_argument("--token", required=True)
    net_parser.add_argument("--port", type=int, default=8080)

    settings = sub.add_parser("settings", parents=[common],
                              help="show or change settings")
    settings_sub = settings.add_subparsers(dest="action", required=True)
    settings_sub.add_parser("show", parents=[common])
    setter = settings_sub.add_parser("set", parents=[common])
    setter.add_argument("key", help="section.key, e.g. game.iso")
    setter.add_argument("value")

    return parser


def main() -> int:
    args = build_parser().parse_args()
    handlers = {
        "status": cmd_status,
        "login": cmd_login,
        "update": cmd_update,
        "play": cmd_play,
        "adapter": cmd_adapter,
        "isos": cmd_isos,
        "settings": cmd_settings,
        "net": cmd_net,
    }
    renderers = {
        "status": human_status,
        "login": human_login,
        "update": human_update,
        "play": human_play,
        "adapter": human_adapter,
        "isos": human_isos,
        "settings": human_settings,
        "net": lambda payload: None,
    }

    payload = handlers[args.command](args)

    if args.json:
        json_mode = True
    elif args.human:
        json_mode = False
    else:
        json_mode = not sys.stdout.isatty()

    if json_mode:
        print(json.dumps(payload, sort_keys=True))
    elif payload.get("ok"):
        renderers[args.command](payload)
    else:
        print(f"error: {payload.get('error')}", file=sys.stderr)

    return 0 if payload.get("ok") else 1


if __name__ == "__main__":
    sys.exit(main())
