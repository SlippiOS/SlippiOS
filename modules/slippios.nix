{ config, pkgs, lib, slippiosPackages, ... }:

let
  session = slippiosPackages.session;
  cli = slippiosPackages.cli;
  shell = slippiosPackages.shell;

  # openbox-autostart only reads its own built-in global file and
  # $XDG_CONFIG_HOME/openbox/autostart. /etc/xdg/openbox/autostart is ignored.
  autostart = pkgs.writeText "slippios-openbox-autostart" ''
    ${pkgs.xorg.xset}/bin/xset s off -dpms
    ${shell}/bin/slippios-shell || \
      exec ${pkgs.xterm}/bin/xterm -hold -fullscreen -fa Monospace -fs 18 -e ${session}/bin/slippios-session
  '';
in {
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  boot.kernelParams = [ "quiet" "loglevel=3" "vt.global_cursor_default=0" ];
  boot.kernelModules = [ "uinput" ];

  services.udev.extraRules = ''
    SUBSYSTEM=="usb", ATTR{idVendor}=="057e", ATTR{idProduct}=="0337", MODE="0666"
  '';

  networking.networkmanager.enable = true;
  networking.firewall.allowedTCPPorts = [ 8080 ];
  time.timeZone = "UTC";

  users.users.player = {
    isNormalUser = true;
    extraGroups = [ "audio" "input" "video" "networkmanager" ];
  };

  services.xserver = {
    enable = true;
    windowManager.openbox.enable = true;
    displayManager = {
      autoLogin.enable = true;
      autoLogin.user = "player";
      defaultSession = "none+openbox";
    };
  };

  environment.systemPackages = with pkgs; [
    bash
    coreutils
    openbox
    pulseaudio
    qrencode
    slippiosPackages.appimage-runner
    slippiosPackages.session
    slippiosPackages.cli
    slippiosPackages.shell
    xterm
    xorg.xrandr
    xorg.xset
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

  systemd.tmpfiles.rules = [
    "d /home/player 0755 player users -"
    "d /home/player/.config 0755 player users -"
    "d /home/player/.config/openbox 0755 player users -"
    "L+ /home/player/.config/openbox/autostart - - - - ${autostart}"
  ];

  system.stateVersion = "25.05";
}
