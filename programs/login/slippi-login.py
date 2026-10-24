#!/usr/bin/env python3
"""Fullscreen Slippi sign-in that captures user.json automatically."""

from __future__ import annotations

import json
import sys
from pathlib import Path

import gi

gi.require_version("Gtk", "3.0")
gi.require_version("WebKit2", "4.1")

from gi.repository import Gtk, WebKit2  # noqa: E402

DESTINATION = Path("/slippi/slippi-user.json")


class LoginWindow(Gtk.Window):
    def __init__(self) -> None:
        super().__init__(title="Slippi sign-in")
        self.set_default_size(1280, 800)
        self.fullscreen()
        self.succeeded = False

        self.webview = WebKit2.WebView()
        self.add(self.webview)
        self.webview.connect("download-started", self._on_download)
        self.connect("destroy", Gtk.main_quit)

        self.webview.load_uri("https://slippi.gg/online/enable")

    def _on_download(self, _view, download) -> None:
        DESTINATION.parent.mkdir(parents=True, exist_ok=True)
        DESTINATION.unlink(missing_ok=True)
        download.set_destination(DESTINATION.as_uri())
        download.connect("finished", self._on_finished)
        download.connect("failed", self._on_failed)

    def _on_finished(self, _download) -> None:
        try:
            data = json.loads(DESTINATION.read_text())
            self.succeeded = all(
                data.get(key) for key in ("uid", "playKey", "connectCode")
            )
        except (OSError, ValueError):
            self.succeeded = False
        self.destroy()

    def _on_failed(self, _download, _error) -> None:
        self.succeeded = False
        self.destroy()


def main() -> int:
    window = LoginWindow()
    window.show_all()
    Gtk.main()
    return 0 if window.succeeded else 1


if __name__ == "__main__":
    sys.exit(main())
