"""Fetch and verify the newest Slippi Mainline Dolphin release.

The appliance always follows upstream, because netplay requires matching
Dolphin versions. A failed check or download leaves the current build usable.
"""

from __future__ import annotations

import fcntl
import hashlib
import json
import os
import re
import time
import urllib.request
from pathlib import Path

BASE = Path("/slippi/dolphin-app")
VERSIONS = BASE / "versions"
CURRENT = BASE / "current"
STATE = BASE / "state.json"
LOCK = BASE / "lock"

API = "https://api.github.com/repos/project-slippi/dolphin/releases?per_page=10"
ASSET_NAME = "Slippi_Netplay_Mainline-x86_64.AppImage"
USER_AGENT = "SlippiOS/0.1"
CHECK_INTERVAL = 900  # seconds; avoid hammering the GitHub API
TAG = re.compile(r"[A-Za-z0-9._-]+")


class UpdateError(Exception):
    pass


def _read_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text())
    except (OSError, ValueError):
        return {}


def _write_json(path: Path, data: dict) -> None:
    tmp = path.with_suffix(".tmp")
    tmp.write_text(json.dumps(data, indent=2))
    os.replace(tmp, path)


def _request(url: str, timeout: int = 20):
    request = urllib.request.Request(
        url,
        headers={
            "User-Agent": USER_AGENT,
            "Accept": "application/vnd.github+json",
        },
    )
    return urllib.request.urlopen(request, timeout=timeout)


def _latest_release() -> tuple[str, dict]:
    with _request(API) as response:
        releases = json.load(response)
    for release in releases:
        if release.get("draft"):
            continue
        for asset in release.get("assets", []):
            if asset.get("name") == ASSET_NAME:
                tag = release.get("tag_name", "")
                if not TAG.fullmatch(tag):
                    raise UpdateError(f"unexpected release tag {tag!r}")
                return tag, asset
    raise UpdateError("no Slippi release with a Linux AppImage found")


def current_path() -> Path | None:
    try:
        target = Path(os.readlink(CURRENT))
    except OSError:
        return None
    return target if target.is_file() else None


def current_tag() -> str | None:
    path = current_path()
    return path.parent.name if path else None


def _digest_ok(path: Path, digest: str | None) -> bool:
    if not digest or not digest.startswith("sha256:"):
        return False
    expected = digest.split(":", 1)[1].lower()
    hasher = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            hasher.update(chunk)
    return hasher.hexdigest() == expected


def _download(asset: dict, destination: Path, log) -> None:
    total = int(asset.get("size") or 0)
    megabytes = total // (1024 * 1024)
    log(f"Downloading {asset['name']} ({megabytes} MB)…")

    partial = destination.with_name(destination.name + ".part")
    request = urllib.request.Request(
        asset["browser_download_url"], headers={"User-Agent": USER_AGENT}
    )
    try:
        with _request(asset["browser_download_url"], timeout=60) as source, \
                partial.open("wb") as output:
            done = 0
            reported = -1
            while chunk := source.read(1 << 20):
                output.write(chunk)
                done += len(chunk)
                current_mb = done >> 20
                if current_mb >= reported + 16:
                    reported = current_mb
                    suffix = f" ({done * 100 // total}%)" if total else ""
                    log(f"  {current_mb} MB{suffix}")
    except Exception:
        partial.unlink(missing_ok=True)
        raise
    os.replace(partial, destination)


def _switch(tag: str, path: Path) -> None:
    temporary = CURRENT.with_name("current.tmp")
    temporary.unlink(missing_ok=True)
    os.symlink(path, temporary)
    os.replace(temporary, CURRENT)

    state = _read_json(STATE)
    state.update({"tag": tag, "checked_at": int(time.time())})
    _write_json(STATE, state)


def ensure(log=print, force: bool = False) -> Path:
    """Return a verified AppImage, downloading the newest release if needed."""
    BASE.mkdir(parents=True, exist_ok=True)
    VERSIONS.mkdir(exist_ok=True)

    with LOCK.open("w") as lock_handle:
        fcntl.flock(lock_handle, fcntl.LOCK_EX)

        existing = current_path()
        state = _read_json(STATE)
        fresh = state.get("checked_at", 0) > time.time() - CHECK_INTERVAL

        if existing and fresh and not force:
            return existing

        try:
            tag, asset = _latest_release()
        except Exception as exc:
            if existing:
                log(f"Update check failed, using {current_tag()}: {exc}")
                return existing
            raise UpdateError(f"cannot check for Slippi updates: {exc}") from exc

        if existing and not force and current_tag() == tag:
            state["checked_at"] = int(time.time())
            _write_json(STATE, state)
            return existing

        directory = VERSIONS / tag
        directory.mkdir(parents=True, exist_ok=True)
        destination = directory / ASSET_NAME

        digest = asset.get("digest")
        if destination.is_file() and _digest_ok(destination, digest):
            log(f"Using cached {tag}.")
        else:
            _download(asset, destination, log)
            if not _digest_ok(destination, digest):
                destination.unlink(missing_ok=True)
                raise UpdateError("downloaded AppImage failed checksum verification")

        destination.chmod(0o755)
        _switch(tag, destination)
        return destination
