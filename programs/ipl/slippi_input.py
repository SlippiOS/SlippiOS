"""SDL gamepad input shared by the IPL (Glyph, Xbox, DualSense, ...)."""

from __future__ import annotations

import os

os.environ.setdefault("SDL_VIDEODRIVER", "dummy")
os.environ.setdefault("SDL_JOYSTICK_ALLOW_BACKGROUND_EVENTS", "1")

import pygame  # noqa: E402

SOUTH = 0  # SDL's A / Cross / south face on standard mappings


class Gamepads:
    def __init__(self) -> None:
        self._ready = False
        self._pads: list[pygame.joystick.Joystick] = []
        try:
            pygame.init()
            pygame.joystick.init()
            self._refresh()
            self._ready = True
        except pygame.error:
            self._ready = False

    def _refresh(self) -> None:
        self._pads = []
        for index in range(pygame.joystick.get_count()):
            try:
                pad = pygame.joystick.Joystick(index)
                pad.init()
                self._pads.append(pad)
            except pygame.error:
                continue

    def pump(self) -> None:
        if not self._ready:
            return
        for event in pygame.event.get():
            if event.type in (pygame.JOYDEVICEADDED, pygame.JOYDEVICEREMOVED):
                self._refresh()

    def any_south_held(self) -> bool:
        """True if any connected SDL controller reports its south face held."""
        if not self._ready:
            return False
        self.pump()
        for pad in self._pads:
            try:
                if pad.get_button(SOUTH):
                    return True
            except pygame.error:
                continue
        return False

    def close(self) -> None:
        if self._ready:
            pygame.joystick.quit()
            pygame.quit()
            self._ready = False
