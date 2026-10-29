{ config, lib, pkgs, slippiosPackages, ... }:

let
  shell = slippiosPackages.shell;
  theme = import ../../theme/default.nix;
  splashMark = ../../apps/shell/assets/slippi-mark.png;

  # The splash mark is markFraction * markFit of the screen height. Render a
  # wallpaper for every common desktop size so xwallpaper can use a 1:1 image
  # instead of scaling a 4K one down and aliasing the thin S stroke.
  markFraction = theme.splash.markFraction * theme.splash.markFit;
  wallpaperSizes = [
    "640x480"
    "800x600"
    "1024x768"
    "1280x720"
    "1280x800"
    "1366x768"
    "1440x900"
    "1600x900"
    "1680x1050"
    "1920x1080"
    "1920x1200"
    "2560x1440"
    "2560x1600"
    "3840x2160"
  ];
  wallpapers = pkgs.runCommand "slippios-wallpapers" {
    nativeBuildInputs = [ pkgs.imagemagick ];
  } ''
    mkdir -p "$out"
    ${lib.concatMapStringsSep "\n" (size:
      let
        parts = lib.splitString "x" size;
        height = lib.toInt (lib.elemAt parts 1);
        mark = builtins.floor (height * markFraction + 0.5);
      in ''
        magick -size "${size}" xc:'${theme.background}' \
          \( ${splashMark} -resize "x${toString mark}" \) \
          -gravity center -composite "$out/${size}.png"
      '') wallpaperSizes}
    for image in "$out"/*.png; do
      printf '%s %s\n' "$(basename "$image" .png)" "$image"
    done > "$out/list"
  '';

  wallpaper = pkgs.writeShellScriptBin "slippios-wallpaper" ''
    if [ -z "''${DISPLAY:-}" ]; then
      exit 0
    fi
    ${pkgs.xsetroot}/bin/xsetroot -solid '${theme.background}' || true

    resolution=$(${pkgs.xrandr}/bin/xrandr --current 2>/dev/null |
      ${pkgs.gawk}/bin/awk '/ connected/ {
        for (i = 1; i <= NF; i++)
          if ($i ~ /^[0-9]+x[0-9]+\+/) { print $i; exit }
      }' | ${pkgs.coreutils}/bin/cut -d+ -f1)

    image=${wallpapers}/''${resolution:-1280x800}.png
    if [ ! -f "$image" ]; then
      image=$(${pkgs.gawk}/bin/awk -v w="''${resolution%x*}" -v h="''${resolution#*x}" '
        {
          split($1, a, "x");
          area = a[1] * a[2];
          if (area > largest_area) { largest = $2; largest_area = area }
          if (a[1] >= w && a[2] >= h && (best == "" || area < best_area)) {
            best = $2;
            best_area = area;
          }
        }
        END { print (best != "" ? best : largest) }
      ' ${wallpapers}/list)
    fi
    ${pkgs.xwallpaper}/bin/xwallpaper --maximize "$image" || true
  '';

  session = pkgs.writeShellScript "slippios-x-session" ''
    # A display-manager stop is not a crash.
    trap 'exit 0' TERM INT HUP

    ${wallpaper}/bin/slippios-wallpaper
    ${pkgs.xset}/bin/xset s off -dpms || true
    ${pkgs.openbox}/bin/openbox --config-file /etc/xdg/openbox/rc.xml &
    wm=$!
    ${pkgs.coreutils}/bin/sleep 0.5
    ${slippiosPackages.cli}/bin/slippios display apply || true
    # Set the splash after the display mode is applied so it scales to the
    # final resolution instead of being cropped by a later mode change.
    ${wallpaper}/bin/slippios-wallpaper
    ${shell}/bin/slippios-shell
    # The shell wrapper handles games and tools. If it exits unexpectedly,
    # leave Openbox running and open a terminal for diagnosis.
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
  environment.systemPackages = [ wallpaper ];

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
