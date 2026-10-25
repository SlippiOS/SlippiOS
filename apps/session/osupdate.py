"""Stage and install SlippiOS system updates.

The running system is an A/B image: the Nix store and matching UKI live in
one slot, while the other slot is empty. systemd-sysupdate writes the staged
pair to the inactive slot. The next boot tries the new UKI three times and
automatically falls back to the previous one if it does not complete.

Downloads use zsync against the active nix-store partition, with a full
download fallback. The release manifest is the only source of versions and
hashes; no package manager is involved.
"""

from __future__ import annotations

import hashlib
import json
import os
import re
import shutil
import subprocess
import time
import urllib.request
from pathlib import Path

DATA = Path("/data")
STAGE = DATA / "system" / "update"
STATUS = STAGE / "status.json"
PROGRESS = STAGE / "progress.json"
LATEST = STAGE / "latest.json"
REQUEST = STAGE / "request"
REBOOT = STAGE / "reboot"
SOURCE = DATA / "system" / "update-source.json"
VERSION_FILE = Path("/etc/slippios/version")

DEFAULT_MANIFEST = (
    "https://github.com/SlippiOS/SlippiOS/releases/latest/download/manifest.json"
)
DOWNLOAD_BASE = "https://github.com/SlippiOS/SlippiOS/releases/download"
USER_AGENT = "SlippiOS/0.2"
SHA256 = re.compile(r"^[0-9a-f]{64}$")


class OSUpdateError(Exception):
    pass


def _read_json(path: Path, default: dict | None = None) -> dict:
    try:
        value = json.loads(path.read_text())
    except (OSError, ValueError):
        return default or {}
    return value if isinstance(value, dict) else (default or {})


def _write_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(value, indent=2) + "\n")
    os.replace(temporary, path)


def _request(url: str, timeout: int = 30):
    return urllib.request.urlopen(
        urllib.request.Request(url, headers={"User-Agent": USER_AGENT}),
        timeout=timeout,
    )


def _source() -> tuple[str | None, str]:
    """Return (asset base, manifest URL).

    A local base is useful for testing private release assets: write
    /data/system/update-source.json with {"base": "http://host:8000/"}.
    """
    base: str | None = None
    manifest = DEFAULT_MANIFEST
    config = _read_json(SOURCE)
    configured_base = config.get("base")
    if isinstance(configured_base, str) and configured_base:
        base = configured_base.rstrip("/") + "/"
        manifest = str(config.get("manifest") or (base + "manifest.json"))
    elif isinstance(config.get("manifest"), str):
        manifest = str(config["manifest"])
    return base, manifest


def _asset_url(base: str | None, tag: str | None, name: str) -> str:
    if base:
        return base + name
    if not tag:
        raise OSUpdateError("release manifest has no tag")
    return f"{DOWNLOAD_BASE}/{tag}/{name}"


def _manifest_from(data: dict) -> dict:
    if not isinstance(data, dict):
        raise OSUpdateError("invalid update manifest")
    version = data.get("version")
    if not isinstance(version, str) or not version:
        raise OSUpdateError("update manifest has no version")
    nix_store = data.get("nix_store")
    uki = data.get("uki")
    if not isinstance(nix_store, dict) or not isinstance(uki, dict):
        raise OSUpdateError("update manifest is missing nix_store/uki")
    for part, label in ((nix_store, "nix_store"), (uki, "uki")):
        name = part.get("name")
        digest = part.get("sha256")
        size = part.get("size")
        if not isinstance(name, str) or "/" in name or not name:
            raise OSUpdateError(f"update manifest has an invalid {label} name")
        if not isinstance(digest, str) or not SHA256.fullmatch(digest):
            raise OSUpdateError(f"update manifest has an invalid {label} sha256")
        if not isinstance(size, int) or size <= 0:
            raise OSUpdateError(f"update manifest has an invalid {label} size")
    tag = data.get("tag")
    if tag is not None and not isinstance(tag, str):
        raise OSUpdateError("update manifest has an invalid tag")
    return data


def fetch_manifest() -> dict:
    base, url = _source()
    try:
        with _request(url) as response:
            data = json.load(response)
    except Exception as exc:
        raise OSUpdateError(f"cannot fetch update manifest: {exc}") from exc
    manifest = _manifest_from(data)
    manifest["_base"] = base
    manifest["_manifest_url"] = url
    return manifest


def current_version() -> str:
    try:
        value = VERSION_FILE.read_text().strip()
    except OSError:
        return "unknown"
    return value or "unknown"


def version_key(version: str) -> int:
    match = re.match(r"(\d+)", version or "")
    return int(match.group(1)) if match else 0


def check() -> dict:
    current = current_version()
    manifest = fetch_manifest()
    latest = manifest["version"]
    available = latest != current and version_key(latest) >= version_key(current)
    STAGE.mkdir(parents=True, exist_ok=True)
    _write_json(
        LATEST,
        {
            "checked_at": int(time.time()),
            "current": current,
            "latest": latest,
            "update_available": available,
            "manifest": {
                key: value for key, value in manifest.items()
                if not key.startswith("_")
            },
            "base": manifest.get("_base"),
        },
    )
    return {
        "ok": True,
        "current": current,
        "latest": latest,
        "update_available": available,
        "manifest": manifest,
    }


def status() -> dict:
    current = current_version()
    cached = _read_json(LATEST)
    service = _read_json(STATUS)
    progress = _read_json(PROGRESS)
    latest = cached.get("latest")
    available = bool(cached.get("update_available"))
    phase = progress.get("phase", "idle")
    if service:
        phase = service.get("phase", phase)
    if (
        service.get("phase") == "installed"
        and isinstance(latest, str)
        and latest
        and current == latest
    ):
        # The new system is already running; the staged update is no longer
        # actionable.
        service = {}
        STATUS.unlink(missing_ok=True)
        PROGRESS.unlink(missing_ok=True)
        phase = "idle"
    return {
        "ok": True,
        "current": current,
        "latest": latest,
        "update_available": available,
        "phase": phase,
        "done": progress.get("done", 0),
        "total": progress.get("total", 0),
        "error": progress.get("error") or service.get("error"),
        "service_ok": service.get("ok"),
    }


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _valid(path: Path, expected: str, size: int | None = None) -> bool:
    try:
        if size is not None and path.stat().st_size != size:
            return False
        return _sha256(path) == expected
    except OSError:
        return False


def _progress(phase: str, done: int = 0, total: int = 0) -> None:
    _write_json(
        PROGRESS,
        {
            "phase": phase,
            "done": int(done),
            "total": int(total),
            "updated_at": int(time.time()),
        },
    )


def fail(message: str) -> None:
    _write_json(
        PROGRESS,
        {
            "phase": "failed",
            "error": message,
            "updated_at": int(time.time()),
        },
    )


def _download(
    url: str,
    destination: Path,
    expected: str,
    size: int,
    phase: str = "downloading",
) -> None:
    partial = destination.with_suffix(destination.suffix + ".part")
    partial.unlink(missing_ok=True)
    try:
        with _request(url, timeout=60) as source, partial.open("wb") as output:
            done = 0
            _progress(phase, 0, size)
            while chunk := source.read(1 << 20):
                output.write(chunk)
                done += len(chunk)
                _progress(phase, done, size)
    except Exception as exc:
        partial.unlink(missing_ok=True)
        raise OSUpdateError(f"download failed: {exc}") from exc
    if not _valid(partial, expected, size):
        partial.unlink(missing_ok=True)
        raise OSUpdateError("downloaded file failed checksum verification")
    os.replace(partial, destination)


def current_store_device() -> str | None:
    """Return the block device currently mounted as /nix/store, if any."""
    try:
        with open("/proc/mounts", encoding="utf-8") as handle:
            for line in handle:
                fields = line.split()
                if len(fields) >= 3 and fields[1] == "/nix/store":
                    device = os.path.realpath(fields[0])
                    return device if Path(device).is_block_device() else None
    except OSError:
        pass
    return None


def _zsync(
    seed: str,
    destination: Path,
    raw_url: str,
    zsync_url: str,
    size: int,
) -> bool:
    executable = shutil.which("zsync")
    if not executable or not seed:
        return False

    destination.unlink(missing_ok=True)
    log = STAGE / "zsync.log"
    try:
        with log.open("wb") as output:
            process = subprocess.Popen(
                [
                    executable,
                    "-i", seed,
                    "-o", str(destination),
                    "-u", raw_url,
                    zsync_url,
                ],
                stdout=output,
                stderr=subprocess.STDOUT,
            )
            while process.poll() is None:
                try:
                    done = destination.stat().st_size
                except OSError:
                    done = 0
                _progress("downloading", done, size)
                time.sleep(1)
    except OSError as exc:
        (STAGE / "zsync-error").write_text(str(exc) + "\n")
        return False
    return process.returncode == 0 and destination.is_file()


def _stage_file(
    name: str,
    expected: str,
    size: int,
    url: str,
    zsync_url: str | None = None,
    seed: str | None = None,
) -> Path:
    destination = STAGE / name
    if _valid(destination, expected, size):
        return destination

    destination.unlink(missing_ok=True)
    if zsync_url and seed:
        if _zsync(seed, destination, url, zsync_url, size) and _valid(
            destination, expected, size
        ):
            return destination
        destination.unlink(missing_ok=True)

    _download(url, destination, expected, size)
    return destination


def stage(log=print) -> dict:
    try:
        return _stage(log)
    except Exception as exc:
        fail(str(exc))
        raise OSUpdateError(str(exc)) from exc


def _stage(log=print) -> dict:
    STAGE.mkdir(parents=True, exist_ok=True)
    REQUEST.unlink(missing_ok=True)
    STATUS.unlink(missing_ok=True)
    PROGRESS.unlink(missing_ok=True)

    checked = check()
    manifest = checked["manifest"]
    if not checked["update_available"]:
        _progress("idle")
        return {
            "ok": True,
            "updated": False,
            "current": checked["current"],
            "latest": checked["latest"],
        }

    version = manifest["version"]
    base = manifest.get("_base")
    tag = manifest.get("tag")
    raw = manifest["nix_store"]
    uki = manifest["uki"]
    raw_name = raw["name"]
    uki_name = uki["name"]
    raw_url = _asset_url(base, tag, raw_name)
    uki_url = _asset_url(base, tag, uki_name)
    zsync_name = raw.get("zsync")
    zsync_url = (
        _asset_url(base, tag, zsync_name)
        if isinstance(zsync_name, str) and zsync_name
        else None
    )

    for old in STAGE.glob("slippios_*.nix-store.raw"):
        if old.name != raw_name:
            old.unlink(missing_ok=True)
    for old in STAGE.glob("slippios_*.efi"):
        if old.name != uki_name:
            old.unlink(missing_ok=True)

    log(f"Staging SlippiOS {version}")
    _progress("downloading", 0, int(raw["size"]))
    _stage_file(
        raw_name,
        raw["sha256"],
        int(raw["size"]),
        raw_url,
        zsync_url=zsync_url,
        seed=current_store_device(),
    )

    _progress("downloading", 0, int(uki["size"]))
    _stage_file(uki_name, uki["sha256"], int(uki["size"]), uki_url)

    _progress("installing")
    REQUEST.write_text(version + "\n")
    return {
        "ok": True,
        "updated": True,
        "current": checked["current"],
        "latest": version,
        "phase": "installing",
    }


def request_reboot() -> dict:
    STAGE.mkdir(parents=True, exist_ok=True)
    REBOOT.write_text(str(int(time.time())) + "\n")
    return {"ok": True}
