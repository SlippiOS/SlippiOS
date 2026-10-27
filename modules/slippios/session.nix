{ config, lib, pkgs, slippiosPackages, ... }:

let
  shell = slippiosPackages.shell;
  theme = import ../../theme/default.nix;
  splashBoot = ../../apps/shell/assets/slippi-boot.png;

  # Run the Godot shell directly as the X session. There is no window manager
  # in the normal path, so the shell owns the whole screen and draws no
  # decorations. If it crashes, Openbox gives us a usable desktop for
  # diagnosis; if the fallback also exits, the recovery console takes over.
  session = pkgs.writeShellScript "slippios-x-session" ''
    # A display-manager stop/restart terminates the whole session. Do not
    # mistake that for a crash and start Openbox on the way out.
    trap 'exit 0' TERM INT HUP

    ${pkgs.xsetroot}/bin/xsetroot -solid '${theme.background}' || true
    ${pkgs.xwallpaper}/bin/xwallpaper --zoom ${splashBoot} || true
    ${pkgs.xset}/bin/xset s off -dpms || true
    ${shell}/bin/slippios-shell
    # Crash/exit fallback: Openbox plus one terminal, so there is something
    # usable to diagnose with. The normal path never starts a window manager.
    ${pkgs.openbox}/bin/openbox &
    wm=$!
    sleep 1
    ${pkgs.xterm}/bin/xterm -fa Monospace -fs 14 &
    term=$!
    wait "$wm"
    kill "$term" 2>/dev/null || true
    # If Openbox also exited while the display manager is still running, hand
    # over to the console recovery menu. During shutdown it is already gone.
    if ${config.systemd.package}/bin/systemctl is-active --quiet display-manager.service; then
      : >/data/system/shell-failed 2>/dev/null || true
    fi
  '';

  recovery = pkgs.writeShellScript "slippios-recovery" ''
    # Shown on tty1 only after Openbox exits.
    log=/data/system/shell.log
    ${pkgs.kbd}/bin/chvt 1 2>/dev/null || true
    while true; do
      printf '\033[2J\033[H'
      ${pkgs.kbd}/bin/chvt 1 2>/dev/null || true
      echo "SlippiOS"
      echo
      echo "The graphical shell stopped."
      echo
      if [ -s "$log" ]; then
        tail -n 12 "$log" 2>/dev/null || true
        echo
      fi
      failed=$(${config.systemd.package}/bin/systemctl --failed --no-legend --plain --no-pager 2>/dev/null || true)
      if [ -n "$failed" ]; then
        echo "Failed units:"
        echo "$failed"
        echo
      fi
      echo "1) Try again"
      echo "2) Power off"
      echo "3) Reboot"
      printf "> "
      read -r choice || choice=2
      case "$choice" in
        1)
          rm -f /data/system/shell-failed
          ${config.systemd.package}/bin/systemctl --no-block start display-manager.service
          exit 0
          ;;
        2) ${config.systemd.package}/bin/systemctl poweroff ;;
        3) ${config.systemd.package}/bin/systemctl reboot ;;
      esac
    done
  '';
in {
  services.xserver = {
    enable = true;
    # No window manager on the normal path: the Godot shell is the session.
    # The session script starts Openbox only if the shell exits.
    displayManager.session = [{
      manage = "desktop";
      name = "slippios";
      start = "exec ${session}";
    }];
  };

  # Renamed from services.xserver.displayManager in NixOS 25.11+.
  services.displayManager = {
    autoLogin.enable = true;
    autoLogin.user = "player";
    defaultSession = "slippios";
  };

  # The stage-2 systemd status for plymouth-quit is still printed despite
  # ShowStatus=no. Clear tty1 after Plymouth has quit and before LightDM
  # starts X, so the last console lines never reach the user.
  systemd.services.display-manager.serviceConfig.ExecStartPre = lib.mkAfter [
    "${pkgs.bash}/bin/bash -c 'printf \\033[2J\\033[H > /dev/tty1'"
  ];

  # The session script starts Openbox when the shell crashes. If Openbox also
  # exits, it writes /data/system/shell-failed and this console takes over.
  systemd.services.slippios-recovery = {
    description = "SlippiOS recovery console.";
    conflicts = [ "display-manager.service" "getty@tty1.service" ];
    serviceConfig = {
      Type = "oneshot";
      StandardInput = "tty";
      StandardOutput = "tty";
      StandardError = "tty";
      TTYPath = "/dev/tty1";
      TTYReset = true;
      TTYVHangup = true;
      ExecStart = recovery;
    };
  };
  systemd.paths.slippios-recovery = {
    description = "Watch for the shell failure flag";
    wantedBy = [ "multi-user.target" ];
    after = [ "data.mount" ];
    pathConfig.PathExists = "/data/system/shell-failed";
  };
}
