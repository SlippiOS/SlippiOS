"""Settings and game-image helpers for SlippiOS."""

from __future__ import annotations

import re
import tomllib
from pathlib import Path

import tomli_w

SLIPPI_DIR = Path("/slippi")
SETTINGS_FILE = SLIPPI_DIR / "settings.toml"
ISO_DIR = SLIPPI_DIR / "isos"
DOLPHIN_DIR = SLIPPI_DIR / "dolphin"

SCHEMA_VERSION = 1

# GameCube game IDs for Melee: USA (GALE01), Japan (GALJ01), PAL (GALP01).
MELEE_ID = re.compile(rb"GAL[A-Z0-9]01")


def defaults() -> dict:
    return {
        "schema_version": SCHEMA_VERSION,
        "game": {"iso": str(ISO_DIR / "melee.iso")},
        "display": {"refresh_hz": 120, "resolution": "auto"},
        "audio": {"sink": "auto"},
        "network": {"ssid": "", "psk": ""},
    }


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
    iso = Path(path)
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
