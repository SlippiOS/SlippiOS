#!/usr/bin/env python3
"""Small, dependency-free Initial Program Loader for SlippiOS.

The VM uses keyboard input so the boot/config state machine can be developed
without lying about GameCube adapter support. Hardware input will be added in
the adapter backend; the important contract is that backend must release the
USB interface before launching Dolphin.
"""

import os
import subprocess
import sys
import time


def clear():
    print("\033[2J\033[H", end="", flush=True)


def boot_animation():
    for frame in ("      ◇", "    ◇ ◇", "  ◇   ◇", "◇  SLIPPIOS  ◇"):
        clear()
        print("\n\n\n                 " + frame)
        print("\n                 Initializing console…")
        time.sleep(0.55)


def config_menu():
    while True:
        clear()
        print("\n                 SLIPPIOS CONFIGURATION\n")
        print("                 [1] Network (coming soon)")
        print("                 [2] Display/audio (coming soon)")
        print("                 [3] Install to internal disk (coming soon)")
        print("                 [Enter] Start Slippi")
        choice = input("\n                 Select: ").strip().lower()
        if choice in ("", "3", "enter"):
            return
        print("\n                 This option is not wired yet.")
        time.sleep(1)


def launch_slippi():
    clear()
    print("\n\n                 Launching Slippi Dolphin…\n")
    print("                 (Dolphin package is the next milestone.)")
    print("\n                 Press Ctrl+C to return to the shell.")
    try:
        while True:
            time.sleep(60)
    except KeyboardInterrupt:
        return


def main():
    boot_animation()
    # VM keyboard escape hatch. The physical implementation will replace this
    # with raw 057e:0337 polling and preserve the same boolean contract.
    if os.environ.get("SLIPPI_CONFIG") == "1":
        config_menu()
    elif os.environ.get("SLIPPI_VM"):
        print("\n                 Hold 'c' then Enter for config in this VM.")
        print("                 Press Enter to continue to Slippi.")
        choice = input("\n                 > ").strip().lower()
        if choice == "c":
            config_menu()
    launch_slippi()


if __name__ == "__main__":
    try:
        main()
    except EOFError:
        sys.exit(0)
