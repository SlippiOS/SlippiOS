"""NetworkManager helpers for the console Wi-Fi setup screen."""

from __future__ import annotations

import os
import shutil
import subprocess
from typing import Any

import settings


class WifiError(Exception):
    pass


def _nmcli(arguments: list[str], timeout: int = 30) -> str:
    executable = shutil.which("nmcli")
    if not executable:
        raise WifiError("NetworkManager CLI is not available")
    environment = os.environ.copy()
    environment["LC_ALL"] = "C"
    try:
        result = subprocess.run(
            [executable, *arguments],
            capture_output=True,
            text=True,
            timeout=timeout,
            env=environment,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise WifiError(str(exc)) from exc
    if result.returncode != 0:
        message = (result.stderr or result.stdout).strip()
        raise WifiError(message or "NetworkManager command failed")
    return result.stdout


def _split_t(line: str) -> list[str]:
    """Split nmcli's escaped terse output."""
    fields: list[str] = []
    current: list[str] = []
    escaped = False
    for character in line:
        if escaped:
            current.append(character)
            escaped = False
        elif character == "\\":
            escaped = True
        elif character == ":":
            fields.append("".join(current))
            current = []
        else:
            current.append(character)
    fields.append("".join(current))
    return fields


def scan() -> list[dict[str, Any]]:
    output = _nmcli(
        ["-t", "-f", "IN-USE,SSID,SIGNAL,SECURITY", "device", "wifi", "list",
         "--rescan", "yes"],
        timeout=45,
    )
    networks: dict[str, dict[str, Any]] = {}
    for line in output.splitlines():
        fields = _split_t(line)
        if len(fields) < 4:
            continue
        active, ssid, signal, security = fields[:4]
        if not ssid:
            continue
        try:
            strength = int(signal)
        except ValueError:
            strength = 0
        candidate = {
            "ssid": ssid,
            "signal": strength,
            "security": "" if security.strip() == "--" else security.strip(),
            "active": active.strip() == "*",
        }
        existing = networks.get(ssid)
        if (
            existing is None
            or candidate["signal"] > existing["signal"]
            or (candidate["active"] and not existing["active"])
        ):
            networks[ssid] = candidate
    return sorted(
        networks.values(),
        key=lambda entry: (not entry["active"], -entry["signal"], entry["ssid"].lower()),
    )


def status() -> dict[str, Any]:
    try:
        output = _nmcli(
            ["-t", "-f", "IN-USE,SSID", "device", "wifi", "list"],
            timeout=15,
        )
    except WifiError:
        return {"connected": False, "ssid": None}
    for line in output.splitlines():
        fields = _split_t(line)
        if len(fields) >= 2 and fields[0].strip() == "*" and fields[1]:
            return {"connected": True, "ssid": fields[1]}
    return {"connected": False, "ssid": None}


def connect(ssid: str, password: str = "") -> dict[str, Any]:
    if not ssid:
        raise WifiError("no network selected")
    arguments = ["device", "wifi", "connect", ssid]
    if password:
        arguments.extend(["password", password])
    _nmcli(arguments, timeout=90)

    stored, _error = settings.load()
    if stored is None:
        stored = settings.defaults()
    stored["network"]["ssid"] = ssid
    settings.save(stored)
    return {"ssid": ssid}
