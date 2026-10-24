"""Tiny LAN page that accepts a Slippi login export.

The user opens the page on a phone, follows the official sign-in link, then
uploads the resulting user.json. The IPL copies it into Dolphin's user dir.
"""

from __future__ import annotations

import html
import json
import socket
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import slippi_settings as settings_lib

PORT = 8080

PAGE = """<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SlippiOS sign-in</title>
<style>
body{font-family:system-ui,sans-serif;background:#111;color:#eee;max-width:34rem;margin:3rem auto;padding:0 1rem;line-height:1.5}
a{color:#7cc4ff}h1{font-size:1.4rem}code{background:#222;padding:.1rem .3rem;border-radius:.2rem}
button{background:#2b6cb0;color:#fff;border:0;border-radius:.4rem;padding:.6rem 1rem;font-size:1rem}
.step{margin:1.2rem 0}.ok{color:#7bd88f}.err{color:#ff8b8b}
</style></head><body>
<h1>SlippiOS sign-in</h1>
<p class="step"><strong>1.</strong> On this device, open
<a href="https://slippi.gg/online/enable" target="_blank" rel="noreferrer">slippi.gg/online/enable</a>
and sign in with your Slippi account.</p>
<p class="step"><strong>2.</strong> Tap <em>Download</em> to save <code>user.json</code>.</p>
<p class="step"><strong>3.</strong> Choose that file here, then return to the console.</p>
<p class="step"><input id="file" type="file" accept="application/json,.json"></p>
<p class="step"><button id="send">Upload</button></p>
<p id="status"></p>
<script>
document.getElementById('send').onclick = async () => {
  const status = document.getElementById('status');
  const file = document.getElementById('file').files[0];
  if (!file) { status.className = 'err'; status.textContent = 'Choose user.json first.'; return; }
  status.className = ''; status.textContent = 'Uploading…';
  try {
    const response = await fetch('/upload', {method: 'POST', body: await file.text()});
    const message = await response.text();
    status.className = response.ok ? 'ok' : 'err';
    status.textContent = message;
  } catch (error) {
    status.className = 'err'; status.textContent = String(error);
  }
};
</script></body></html>
"""


def local_address() -> str:
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as sock:
            sock.connect(("8.8.8.8", 80))
            return sock.getsockname()[0]
    except OSError:
        return "127.0.0.1"


class _Handler(BaseHTTPRequestHandler):
    def _respond(self, code: int, body: str, content_type: str = "text/plain") -> None:
        payload = body.encode()
        self.send_response(code)
        self.send_header("Content-Type", content_type + "; charset=utf-8")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self) -> None:  # noqa: N802
        if self.path not in ("/", "/index.html"):
            self.send_error(404)
            return
        self._respond(200, PAGE, "text/html")

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/upload":
            self.send_error(404)
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            length = 0
        if length <= 0 or length > 64 * 1024:
            self._respond(413, "File is empty or too large.")
            return

        try:
            data = json.loads(self.rfile.read(length))
        except ValueError:
            self._respond(400, "That file is not valid JSON.")
            return

        missing = [key for key in ("uid", "playKey", "connectCode") if not data.get(key)]
        if missing:
            self._respond(400, "Missing: " + ", ".join(missing))
            return

        destination = settings_lib.SLIPPI_DIR / settings_lib.DEFAULT_USER_JSON
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_text(json.dumps(data, indent=2))
        destination.chmod(0o600)

        name = html.escape(str(data.get("displayName") or "player"))
        code = html.escape(str(data["connectCode"]))
        self._respond(200, f"Signed in as {name} ({code}). You can return to the console.")

    def log_message(self, *_args) -> None:
        pass


def start() -> ThreadingHTTPServer:
    """Start the sign-in page in the background."""
    server = ThreadingHTTPServer(("0.0.0.0", PORT), _Handler)
    threading.Thread(target=server.serve_forever, daemon=True).start()
    return server


def stop(server: ThreadingHTTPServer) -> None:
    server.shutdown()
    server.server_close()


def url() -> str:
    return f"http://{local_address()}:{PORT}/"
