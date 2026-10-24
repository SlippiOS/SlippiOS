"""GameCube adapter (057e:0337) boot-time button polling."""

from __future__ import annotations

import time

import usb.core
import usb.util

VID = 0x057E
PID = 0x0337
ENDPOINT_OUT = 0x02
ENDPOINT_IN = 0x81
PACKET_SIZE = 37
A_BUTTON = 0x01
CONNECTED = 0x30


def a_held(seconds: float = 2.0) -> bool | None:
    """Return True/False if an adapter is present, or None if none is found.

    The USB interface is always released before returning so Dolphin can claim
    the device with libusb.
    """
    device = usb.core.find(idVendor=VID, idProduct=PID)
    if device is None:
        return None

    detached = False
    try:
        try:
            if device.is_kernel_driver_active(0):
                device.detach_kernel_driver(0)
                detached = True
        except NotImplementedError:
            pass

        usb.util.claim_interface(device, 0)
        try:
            device.write(ENDPOINT_OUT, b"\x13")  # wake Wii U / Mayflash adapters
        except usb.core.USBError:
            pass

        deadline = time.monotonic() + seconds
        while time.monotonic() < deadline:
            try:
                data = device.read(ENDPOINT_IN, PACKET_SIZE, timeout=16)
            except usb.core.USBError:
                continue
            for port in range(4):
                offset = 1 + port * 9
                if len(data) > offset + 1:
                    status = data[offset]
                    buttons = data[offset + 1]
                    if (status & CONNECTED) and (buttons & A_BUTTON):
                        return True
        return False
    except usb.core.USBError:
        return False
    finally:
        try:
            usb.util.release_interface(device, 0)
        except Exception:
            pass
        try:
            usb.util.dispose_resources(device)
        except Exception:
            pass
