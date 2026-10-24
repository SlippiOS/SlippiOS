"""Settings and game-image helpers for SlippiOS."""

from __future__ import annotations

import json
import os
import re
import shutil
import tomllib
from pathlib import Path

import tomli_w

SLIPPI_DIR = Path("/slippi")
SETTINGS_FILE = SLIPPI_DIR / "settings.toml"
ISO_DIR = SLIPPI_DIR / "isos"
DOLPHIN_DIR = SLIPPI_DIR / "dolphin"
DOLPHIN_USER_FILE = DOLPHIN_DIR / "Slippi" / "user.json"
DEFAULT_USER_JSON = "slippi-user.json"
HARDWARE_FILE = SLIPPI_DIR / "hardware-id"

_PLACEHOLDER_IDS = {
    "",
    "none",
    "not specified",
    "to be filled by o.e.m.",
    "03000200-0400-0500-0006-000700080009",
}


def hardware_id() -> str | None:
    """Best-effort stable machine identity (DMI product UUID)."""
    for path in ("/sys/class/dmi/id/product_uuid", "/sys/class/dmi/id/board_serial"):
        try:
            value = Path(path).read_text().strip()
        except OSError:
            continue
        if value.lower() not in _PLACEHOLDER_IDS:
            return value
    return None


def check_hardware_binding() -> bool:
    """Drop a persisted login when the cartridge moved to different hardware."""
    current = hardware_id()
    if current is None:
        return False
    try:
        stored = HARDWARE_FILE.read_text().strip()
    except OSError:
        stored = ""
    if stored and stored != current:
        (SLIPPI_DIR / DEFAULT_USER_JSON).unlink(missing_ok=True)
        DOLPHIN_USER_FILE.unlink(missing_ok=True)
        HARDWARE_FILE.unlink(missing_ok=True)
        return True
    if not stored:
        HARDWARE_FILE.write_text(current + "\n")
    return False

SCHEMA_VERSION = 1

# GameCube game IDs for Melee: USA (GALE01), Japan (GALJ01), PAL (GALP01).
MELEE_ID = re.compile(rb"GAL[A-Z0-9]01")


def defaults() -> dict:
    return {
        "schema_version": SCHEMA_VERSION,
        "game": {"iso": "melee.iso"},
        "display": {"refresh_hz": 120, "resolution": "auto"},
        "audio": {"sink": "auto"},
        "network": {"ssid": "", "psk": ""},
        "slippi": {"user_json": DEFAULT_USER_JSON},
    }


def iso_path(value: str | Path) -> Path:
    """Resolve an ISO reference; relative paths live in the isos directory."""
    path = Path(value)
    return path if path.is_absolute() else ISO_DIR / path


def resolve_path(value: str | Path) -> Path:
    path = Path(value)
    return path if path.is_absolute() else SLIPPI_DIR / path


def login_status(path: str | Path) -> tuple[bool, str]:
    """Check a Slippi user.json export and return its connect code."""
    candidate = resolve_path(path)
    if not candidate.is_file():
        return False, f"not found: {candidate}"
    try:
        data = json.loads(candidate.read_text())
    except (OSError, ValueError) as exc:
        return False, f"cannot read {candidate}: {exc}"
    if not isinstance(data, dict):
        return False, f"{candidate} is not a Slippi user file"
    missing = [key for key in ("uid", "playKey", "connectCode") if not data.get(key)]
    if missing:
        return False, f"{candidate} is missing {', '.join(missing)}"
    return True, str(data["connectCode"])


def install_login(settings: dict) -> tuple[bool, str]:
    """Copy an imported user.json into Dolphin's user directory."""
    configured = settings.get("slippi", {}).get("user_json", DEFAULT_USER_JSON)
    candidates = [resolve_path(configured), SLIPPI_DIR / "user.json"]
    for candidate in candidates:
        ok, detail = login_status(candidate)
        if not ok:
            continue
        DOLPHIN_USER_FILE.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(candidate, DOLPHIN_USER_FILE)
        os.chmod(DOLPHIN_USER_FILE, 0o600)
        current = hardware_id()
        if current:
            HARDWARE_FILE.write_text(current + "\n")
        return True, detail
    return False, f"no Slippi login file at {resolve_path(configured)}"


def load() -> tuple[dict | None, str | None]:
    """Load settings, returning (settings, error)."""
    if not SETTINGS_FILE.is_file():
        return None, f"missing {SETTINGS_FILE}"

    try:
        with SETTINGS_FILE.open("rb") as handle:
            data = tomllib.load(handle)
    except (OSError, tomllib.TOMLDecodeError) as exc:
        return None, f"cannot read {SETTINGS_FILE}: {exc}"

    version = data.get("schema_version")
    if version != SCHEMA_VERSION:
        return None, f"unsupported schema_version {version!r}"

    merged = defaults()
    for section, values in data.items():
        if isinstance(values, dict) and isinstance(merged.get(section), dict):
            merged[section].update(values)
        else:
            merged[section] = values
    return merged, None


def save(settings: dict) -> None:
    SLIPPI_DIR.mkdir(parents=True, exist_ok=True)
    SETTINGS_FILE.write_text(tomli_w.dumps(settings), encoding="utf-8")


def iso_status(path: str | Path) -> tuple[bool, str]:
    """Check that a path looks like a Melee disc image."""
    iso = iso_path(path)
    if not iso.is_file():
        return False, f"not found: {iso}"
    try:
        with iso.open("rb") as handle:
            header = handle.read(6)
    except OSError as exc:
        return False, f"cannot read {iso}: {exc}"
    if not MELEE_ID.match(header):
        return False, f"{iso} is not a Melee image (game id {header!r})"
    return True, "ok"


def search_dirs() -> list[Path]:
    dirs = [ISO_DIR]
    for base in (Path("/run/media"), Path("/media"), Path("/mnt")):
        if not base.is_dir():
            continue
        try:
            dirs.extend(entry for entry in base.iterdir() if entry.is_dir())
            dirs.extend(entry for entry in base.glob("*/*") if entry.is_dir())
        except OSError:
            continue
    unique = []
    for directory in dirs:
        if directory not in unique:
            unique.append(directory)
    return unique


def find_isos() -> list[tuple[Path, bool, str]]:
    """Return (path, valid, reason) for every *.iso in known locations."""
    found: dict[Path, tuple[bool, str]] = {}
    for directory in search_dirs():
        try:
            candidates = sorted(directory.rglob("*.iso"))
        except OSError:
            continue
        for candidate in candidates:
            if candidate in found:
                continue
            found[candidate] = iso_status(candidate)
    return [(path, ok, reason) for path, (ok, reason) in found.items()]


def any_valid_iso() -> bool:
    return any(ok for _, ok, _ in find_isos())
