{ config, lib, pkgs, slippiosPackages, ... }:

let
  shell = slippiosPackages.shell;
  theme = import ../../theme/default.nix;
  splashBoot = ../../apps/shell/assets/slippi-boot.png;

  session = pkgs.writeShellScript "slippios-x-session" ''
    # A display-manager stop is not a crash; do not start Openbox.
    trap 'exit 0' TERM INT HUP

    ${pkgs.xsetroot}/bin/xsetroot -solid '${theme.background}' || true
    ${pkgs.xwallpaper}/bin/xwallpaper --zoom ${splashBoot} || true
    ${pkgs.xset}/bin/xset s off -dpms || true
    ${shell}/bin/slippios-shell
    ${pkgs.openbox}/bin/openbox &
    wm=$!
    sleep 1
    ${pkgs.xterm}/bin/xterm -fa Monospace -fs 14 &
    term=$!
    wait "$wm"
    kill "$term" 2>/dev/null || true
    if ${config.systemd.package}/bin/systemctl is-active --quiet display-manager.service; then
      : >/data/system/shell-failed 2>/dev/null || true
    fi
  '';

  recovery = pkgs.writeShellScript "slippios-recovery" ''
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
    displayManager.session = [{
      manage = "desktop";
      name = "slippios";
      start = "exec ${session}";
    }];
  };

  services.displayManager = {
    autoLogin.enable = true;
    autoLogin.user = "player";
    defaultSession = "slippios";
  };

  # Clear tty1 after Plymouth quits and before LightDM starts X.
  systemd.services.display-manager.serviceConfig.ExecStartPre = lib.mkAfter [
    "${pkgs.bash}/bin/bash -c 'printf \\033[2J\\033[H > /dev/tty1'"
  ];

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
