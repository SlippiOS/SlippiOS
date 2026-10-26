{ config, pkgs, lib, slippiosPackages, ... }:

let
  cli = slippiosPackages.cli;
  shell = slippiosPackages.shell;

  # Temporary branding: the Slippi mark recoloured to Slippi green. Reused by
  # the Godot splash, the Plymouth theme, and the X root window so the handoff
  # between them has no blank frame.
  splashMark = ../apps/shell/assets/slippi-mark.png;
  splashBoot = ../apps/shell/assets/slippi-boot.png;

  slippiosPlymouth = pkgs.runCommand "slippios-plymouth" { } ''
    theme=$out/share/plymouth/themes/slippios
    mkdir -p "$theme"
    cp ${splashMark} "$theme/mark.png"
    cp ${../assets/plymouth/slippios.script} "$theme/slippios.script"
    cat > "$theme/slippios.plymouth" <<EOF
    [Plymouth Theme]
    Name=SlippiOS
    Description=SlippiOS boot splash
    ModuleName=script

    [script]
    ImageDir=$theme
    ScriptFile=$theme/slippios.script
    EOF
  '';

  # openbox-autostart only reads its own built-in global file and
  # $XDG_CONFIG_HOME/openbox/autostart. /etc/xdg/openbox/autostart is ignored.
  #
  # There is deliberately no automatic xterm fallback: when the shell fails,
  # the wrapper drops a flag and the recovery console takes the console
  # instead. xterm is only used by the expert-mode tools.
  autostart = pkgs.writeText "slippios-openbox-autostart" ''
    # Paint the same mark before Godot is ready, so the X start and Godot
    # startup never show a blank screen.
    ${pkgs.xorg.xsetroot}/bin/xsetroot -solid '#0b0b10'
    ${pkgs.xwallpaper}/bin/xwallpaper --center ${splashBoot}
    ${pkgs.xset}/bin/xset s off -dpms
    ${shell}/bin/slippios-shell
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
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  # The console is covered by the boot splash; keep systemd's service status
  # lines off it. Failures remain in the journal and the recovery console.
  boot.plymouth = {
    enable = true;
    theme = "slippios";
    themePackages = [ slippiosPlymouth ];
    logo = splashMark;
  };

  boot.kernelParams = [
    "quiet" "loglevel=3" "vt.global_cursor_default=0" "systemd.show_status=0"
  ];
  boot.kernelModules = [ "uinput" ];

  services.udev.extraRules = ''
    SUBSYSTEM=="usb", ATTR{idVendor}=="057e", ATTR{idProduct}=="0337", MODE="0666"
  '';

  networking.networkmanager.enable = true;
  networking.firewall.allowedTCPPorts = [ 8080 ];
  time.timeZone = "UTC";

  users.users.player = {
    isNormalUser = true;
    uid = 1000;
    extraGroups = [ "audio" "input" "video" "networkmanager" ];
  };

  services.xserver = {
    enable = true;
    windowManager.openbox.enable = true;
  };

  # Renamed from services.xserver.displayManager in NixOS 25.11+.
  services.displayManager = {
    autoLogin.enable = true;
    autoLogin.user = "player";
    defaultSession = "none+openbox";
  };

  environment.systemPackages = with pkgs; [
    bash
    coreutils
    openbox
    pulseaudio
    qrencode
    networkmanager
    slippiosPackages.appimage-runner
    slippiosPackages.session
    slippiosPackages.cli
    slippiosPackages.shell
    nano
    xterm
    libxcvt
    xrandr
    xset
  ];

  environment.etc."slippios/README".text = ''
    Settings live in /data/users/default/settings.toml.
    Game images live in /data/isos/.
  '';

  # SlippiOS stores user data on a partition labelled SLIPPIOS. Mount it when
  # present; otherwise /data lives on the (ephemeral) root filesystem so the
  # live ISO still works.
  systemd.services.slippi-data = {
    description = "Prepare SlippiOS data directory";
    wantedBy = [ "multi-user.target" ];
    before = [ "display-manager.service" ];
    after = [ "local-fs.target" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    script = ''
      ${pkgs.coreutils}/bin/mkdir -p /data
      if [ -b /dev/disk/by-label/SLIPPIOS ]; then
        ${pkgs.util-linux}/bin/mountpoint -q /data || \
          ${pkgs.util-linux}/bin/mount -L SLIPPIOS /data || true
      fi
      # A crash flag from the previous boot is stale: try the shell again.
      ${pkgs.coreutils}/bin/rm -f /data/system/shell-failed
      ${pkgs.coreutils}/bin/mkdir -p \
        /data/isos /data/system /data/users/default/dolphin \
        /data/users/default/replays /data/users/default/logs
      ${pkgs.coreutils}/bin/chown player:users \
        /data /data/isos /data/system /data/users /data/users/default \
        /data/users/default/dolphin /data/users/default/replays \
        /data/users/default/logs || true
      ${pkgs.coreutils}/bin/chmod 0775 /data || true

      # Seed community controller profiles without overwriting user edits.
      profile_dir=/data/users/default/dolphin/Config/Profiles/GCPad
      ${pkgs.coreutils}/bin/mkdir -p "$profile_dir"
      ${pkgs.coreutils}/bin/cp -n ${../assets/controller-profiles}/*.ini "$profile_dir"/ || true
      ${pkgs.coreutils}/bin/chown -R player:users "$profile_dir" || true
    '';
  };

  # The shell wrapper writes /data/system/shell-failed when Godot exits for
  # any reason other than Play (42) or Dolphin settings (43).
  systemd.services.slippios-recovery = {
    description = "SlippiOS recovery console";
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
    wantedBy = [ "multi-user.target" ];
    after = [ "data.mount" ];
    pathConfig.PathExists = "/data/system/shell-failed";
  };

  systemd.tmpfiles.rules = [
    "d /home/player 0755 player users -"
    "d /home/player/.config 0755 player users -"
    "d /home/player/.config/openbox 0755 player users -"
    "L+ /home/player/.config/openbox/autostart - - - - ${autostart}"
  ];

  system.stateVersion = "25.05";
}
