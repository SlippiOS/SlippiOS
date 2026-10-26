{ config, pkgs, lib, slippiosPackages, ... }:

let
  cli = slippiosPackages.cli;
  shell = slippiosPackages.shell;

  # Temporary branding: the Slippi mark recoloured to Slippi green. Reused by
  # the Godot splash, the Plymouth theme, and the X root window so the handoff
  # between them has no blank frame.
  splashMark = ../apps/shell/assets/slippi-mark.png;
  splashBoot = ../apps/shell/assets/slippi-boot.png;
  theme = import ../theme/default.nix;
  plymouthConf = ''
    [Daemon]
    ShowDelay=0
    DeviceTimeout=1
    Theme=slippios
  '';
  themeBackground = pkgs.runCommand "slippios-theme-background.png" { } ''
    ${pkgs.imagemagick}/bin/magick -size 1920x1080 xc:'${theme.background}' "$out"
  '';
  themeJson = pkgs.writeText "slippios-theme.json" (builtins.toJSON {
    inherit (theme) background surface accent text;
    splash = {
      mark_fraction = theme.splash.markFraction;
      pulse_speed = theme.splash.pulseSpeed;
      pulse_min = theme.splash.pulseMin;
      pulse_max = theme.splash.pulseMax;
      scale_pulse = theme.splash.scalePulse;
    };
  });

  # Charge-style two-step theme: the mark is pre-rendered as throbber frames
  # with a sine opacity, the same C-side animation Fedora's charge theme uses.
  slippiosPlymouth = pkgs.runCommand "slippios-plymouth" { } ''
    theme=$out/share/plymouth/themes/slippios
    mkdir -p "$theme"
    # The two-step plugin loads these dialog assets during show_splash_screen,
    # even in boot mode; without them it aborts and Plymouth falls back to text.
    cp ${pkgs.plymouth}/share/plymouth/themes/spinner/{lock,entry,keyboard,capslock,keymap-render,bullet}.png "$theme"/
    for frame in $(seq 1 ${toString theme.splash.plymouthFrames}); do
      alpha=$(awk -v i="$frame" 'BEGIN { pi = atan2(0, -1); print ${toString theme.splash.pulseMin} + (${toString theme.splash.pulseMax} - ${toString theme.splash.pulseMin}) * (0.5 + 0.5 * sin(2 * pi * i / ${toString theme.splash.plymouthFrames})) }')
      ${pkgs.imagemagick}/bin/magick ${splashMark} -resize x${toString theme.splash.plymouthMarkHeight} \
        -channel A -evaluate multiply "$alpha" +channel PNG32:"$theme/throbber-$(printf %04d "$frame").png"
    done
    cat > "$theme/slippios.plymouth" <<EOF
    [Plymouth Theme]
    Name=SlippiOS
    Description=SlippiOS boot splash
    ModuleName=two-step

    [two-step]
    Font=DejaVu Sans 12
    TitleFont=DejaVu Sans 30
    ImageDir=$theme
    HorizontalAlignment=.5
    VerticalAlignment=.5
    BackgroundStartColor=0x${lib.removePrefix "#" theme.background}
    BackgroundEndColor=0x${lib.removePrefix "#" theme.background}
    Transition=none
    TransitionDuration=0.0
    ProgressBarBackgroundColor=0x${lib.removePrefix "#" theme.surface}
    ProgressBarForegroundColor=0x${lib.removePrefix "#" theme.accent}
    MessageBelowAnimation=true

    [boot-up]
    UseEndAnimation=false

    [shutdown]
    UseEndAnimation=false

    [reboot]
    UseEndAnimation=false
    EOF
  '';

  # Run the Godot shell directly as the X session. There is no window manager
  # in the normal path, so the shell owns the whole screen and draws no
  # decorations. If it crashes, Openbox gives us a usable desktop for
  # diagnosis; if the fallback also exits, the recovery console takes over.
  session = pkgs.writeShellScript "slippios-x-session" ''
    # A display-manager stop/restart terminates the whole session. Do not
    # mistake that for a crash and start Openbox on the way out.
    trap 'exit 0' TERM INT HUP

    ${pkgs.xorg.xsetroot}/bin/xsetroot -solid '${theme.background}' || true
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
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  # The console is covered by the boot splash; keep systemd's service status
  # lines off it. Failures remain in the journal and the recovery console.
  boot.plymouth = {
    enable = true;
    theme = "slippios";
    themePackages = [ slippiosPlymouth ];
    logo = splashMark;
  };

  # NixOS hardcodes DeviceTimeout=8 before extraConfig; duplicate keys keep the
  # first value, so replace the generated config outright. Without this the
  # console sits on screen until the 8s device wait expires.
  boot.initrd.systemd.contents."/etc/plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };
  environment.etc."plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };

  boot.kernelParams = [
    "quiet" "loglevel=3" "vt.global_cursor_default=0" "systemd.show_status=0"
  ];
  # NixOS appends loglevel=<consoleLogLevel> after our params, so keep this in
  # sync: 0 suppresses late kernel/systemd text between Plymouth and the shell.
  boot.consoleLogLevel = 0;
  # The initrd systemd manager ignores the kernel show_status setting on this
  # path; disable status lines explicitly in both stages. Otherwise the last
  # lines reappear when Plymouth quits.
  boot.initrd.systemd.settings.Manager.ShowStatus = "no";
  systemd.settings.Manager.ShowStatus = "no";
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
    # No window manager on the normal path: the Godot shell is the session.
    # The session script starts Openbox only if the shell exits.
    displayManager.session = [{
      manage = "desktop";
      name = "slippios";
      start = "exec ${session}";
    }];
    # LightDM's greeter background, used during autologin before the session
    # paints the root window. Same value as the Plymouth/shell background.
    displayManager.lightdm.background = themeBackground;
  };

  # The stage-2 systemd status for plymouth-quit is still printed despite
  # ShowStatus=no. Clear tty1 after Plymouth has quit and before LightDM
  # starts X, so the last console lines never reach the user.
  systemd.services.display-manager.serviceConfig.ExecStartPre = lib.mkAfter [
    "${pkgs.bash}/bin/bash -c 'printf \\033[2J\\033[H > /dev/tty1'"
  ];

  # Renamed from services.xserver.displayManager in NixOS 25.11+.
  services.displayManager = {
    autoLogin.enable = true;
    autoLogin.user = "player";
    defaultSession = "slippios";
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

  # Runtime copy of theme/default.nix for the Godot shell.
  environment.etc."slippios/theme.json".source = themeJson;

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

  systemd.tmpfiles.rules = [
    "d /home/player 0755 player users -"
    "d /home/player/.config 0755 player users -"
  ];

  system.stateVersion = "25.05";
}
