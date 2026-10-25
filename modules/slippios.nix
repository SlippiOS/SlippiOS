{ config, pkgs, lib, slippiosPackages, ... }:

let
  session = slippiosPackages.session;
  ctl = slippiosPackages.ctl;
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
  time.timeZone = "UTC";

  users.users.melee = {
    isNormalUser = true;
    extraGroups = [ "audio" "input" "video" "networkmanager" ];
  };

  services.xserver = {
    enable = true;
    windowManager.openbox.enable = true;
    displayManager = {
      autoLogin.enable = true;
      autoLogin.user = "melee";
      defaultSession = "none+openbox";
    };
  };

  environment.systemPackages = with pkgs; [
    bash
    coreutils
    openbox
    pulseaudio
    slippiosPackages.appimage-runner
    slippiosPackages.session
    slippiosPackages.ctl
    slippiosPackages.shell
    xterm
    xorg.xrandr
    xorg.xset
  ];

  environment.etc."slippios/README".text = ''
    Settings live in /slippi/settings.toml. Game images live in /slippi/isos/.
  '';

  # SlippiOS stores user data on a partition labelled SLIPPI. Mount it when
  # present; otherwise /slippi lives on the (ephemeral) root filesystem so the
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
      ${pkgs.coreutils}/bin/mkdir -p /slippi
      if [ -b /dev/disk/by-label/SLIPPI ]; then
        ${pkgs.util-linux}/bin/mountpoint -q /slippi || \
          ${pkgs.util-linux}/bin/mount -L SLIPPI /slippi || true
      fi
      ${pkgs.coreutils}/bin/mkdir -p \
        /slippi/isos /slippi/dolphin /slippi/replays /slippi/logs
      ${pkgs.coreutils}/bin/chown melee:users \
        /slippi /slippi/isos /slippi/dolphin /slippi/replays /slippi/logs || true
      ${pkgs.coreutils}/bin/chmod 0775 /slippi || true

      # Seed community controller profiles without overwriting user edits.
      profile_dir=/slippi/dolphin/Config/Profiles/GCPad
      ${pkgs.coreutils}/bin/mkdir -p "$profile_dir"
      ${pkgs.coreutils}/bin/cp -n ${../assets/controller-profiles}/*.ini "$profile_dir"/ || true
      ${pkgs.coreutils}/bin/chown -R melee:users "$profile_dir" || true
    '';
  };

  systemd.tmpfiles.rules = [
    "d /home/melee 0755 melee users -"
    "d /home/melee/.config 0755 melee users -"
    "d /home/melee/.config/openbox 0755 melee users -"
    "L+ /home/melee/.config/openbox/autostart - - - - ${autostart}"
  ];

  system.stateVersion = "25.05";
}
