"""LAN pages for Slippi sign-in and ISO uploads, reached by QR code."""

from __future__ import annotations

import html
import json
import os
import socket
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import auth
import settings

MAX_UPLOAD = 8 * 1024 * 1024 * 1024


def local_address() -> str:
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
            sock.connect(("8.8.8.8", 80))
            return sock.getsockname()[0]
    except OSError:
        return "127.0.0.1"


def page(token: str, message: str = "", error: bool = False) -> str:
    tone = "err" if error else "ok"
    return f"""<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SlippiOS</title>
<style>
body{{font-family:system-ui,sans-serif;background:#111;color:#eee;max-width:34rem;margin:2rem auto;padding:0 1rem;line-height:1.5}}
h1{{font-size:1.5rem}}h2{{font-size:1.1rem;margin-top:2rem}}
input,button{{font:inherit}}input[type=text],input[type=email],input[type=password]{{width:100%;padding:.6rem;margin:.3rem 0;border-radius:.4rem;border:1px solid #333;background:#1b1b1b;color:#eee}}
button{{background:#2b6cb0;color:#fff;border:0;border-radius:.4rem;padding:.6rem 1rem;margin-top:.4rem}}
.ok{{color:#7bd88f}}.err{{color:#ff8b8b}}code{{background:#222;padding:.1rem .3rem;border-radius:.2rem}}
</style></head><body>
<h1>SlippiOS</h1>
<p>This page is only reachable on your local network.</p>
{f'<p class="{tone}">{html.escape(message)}</p>' if message else ''}
<h2>Sign in to Slippi</h2>
<form method="post" action="/{token}/auth">
  <input type="email" name="email" placeholder="Email" autocomplete="username" required>
  <input type="password" name="password" placeholder="Password" autocomplete="current-password" required>
  <button type="submit">Sign in</button>
</form>
<h2>Add a game image</h2>
<p>Select a Melee 1.02 ISO. It is copied to <code>/data/isos</code>.</p>
<input id="file" type="file" accept=".iso,application/x-iso9660-image">
<button onclick="upload()">Upload</button>
<pre id="status"></pre>
<script>
async function upload() {{
  const status = document.getElementById('status');
  const file = document.getElementById('file').files[0];
  if (!file) {{ status.textContent = 'Choose an ISO first.'; return; }}
  status.textContent = 'Uploading ' + file.name + '…';
  try {{
    const response = await fetch('/{token}/iso?name=' + encodeURIComponent(file.name), {{method: 'POST', body: file}});
    status.textContent = await response.text();
  }} catch (error) {{ status.textContent = String(error); }}
}}
</script></body></html>"""


def serve(token: str, port: int = 8080) -> None:
    class Handler(BaseHTTPRequestHandler):
        def _route(self) -> list[str] | None:
            path = urllib.parse.urlparse(self.path).path.strip("/")
            parts = path.split("/") if path else []
            if not parts or parts[0] != token:
                return None
            return parts[1:]

        def _send(self, code: int, body: str, content_type: str = "text/html") -> None:
            payload = body.encode()
            self.send_response(code)
            self.send_header("Content-Type", content_type + "; charset=utf-8")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        def do_GET(self) -> None:  # noqa: N802
            parts = self._route()
            if parts is None:
                self.send_error(404)
                return
            if parts == [] or parts == [""]:
                self._send(200, page(token))
            else:
                self.send_error(404)

        def do_POST(self) -> None:  # noqa: N802
            parts = self._route()
            if parts is None or not parts:
                self.send_error(404)
                return
            if parts[0] == "auth":
                self._authenticate()
            elif parts[0] == "iso":
                self._upload()
            else:
                self.send_error(404)

        def do_PUT(self) -> None:  # noqa: N802
            self._upload()

        def _authenticate(self) -> None:
            length = int(self.headers.get("Content-Length", "0") or 0)
            body = self.rfile.read(length).decode(errors="replace")
            form = urllib.parse.parse_qs(body)
            email = (form.get("email") or [""])[0].strip()
            password = (form.get("password") or [""])[0]

            try:
                user = auth.login(email, password)
            except auth.AuthError as exc:
                self._send(400, page(token, str(exc), error=True))
                return

            target = settings.PROFILE_DIR / settings.DEFAULT_USER_JSON
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(json.dumps(user, indent=2))
            target.chmod(0o600)
            installed, detail = settings.install_login(settings.defaults())
            if not installed:
                self._send(500, page(token, detail, error=True))
                return

            name = html.escape(user.get("displayName") or "player")
            code = html.escape(user.get("connectCode") or "")
            self._send(200, page(token, f"Signed in as {name} ({code}). You can return to the console."))

        def _upload(self) -> None:
            query = urllib.parse.parse_qs(urllib.parse.urlparse(self.path).query)
            name = Path((query.get("name") or ["upload.iso"])[0]).name
            if not name.lower().endswith(".iso"):
                self._send(400, "Only .iso files are accepted.", "text/plain")
                return

            length = int(self.headers.get("Content-Length", "0") or 0)
            if length <= 0 or length > MAX_UPLOAD:
                self._send(413, "Missing or oversized upload.", "text/plain")
                return

            settings.ISO_DIR.mkdir(parents=True, exist_ok=True)
            partial = settings.ISO_DIR / (name + ".part")
            destination = settings.ISO_DIR / name
            try:
                remaining = length
                with partial.open("wb") as output:
                    while remaining > 0:
                        chunk = self.rfile.read(min(1 << 20, remaining))
                        if not chunk:
                            break
                        output.write(chunk)
                        remaining -= len(chunk)
            except OSError as exc:
                partial.unlink(missing_ok=True)
                self._send(507, f"Could not store the image: {exc}", "text/plain")
                return

            ok, reason = settings.iso_status(partial.name)
            if not ok:
                partial.unlink(missing_ok=True)
                self._send(400, reason, "text/plain")
                return

            os.replace(partial, destination)
            destination.chmod(0o444)
            self._send(200, f"Added {name}. Return to the console.", "text/plain")

        def log_message(self, *_args) -> None:
            pass

    server = ThreadingHTTPServer(("0.0.0.0", port), Handler)
    print(json.dumps({"ok": True, "url": f"http://{local_address()}:{port}/{token}/"}), flush=True)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
