{ config, pkgs, lib, ... }:

let
  python = pkgs.python3.withPackages (ps: [ ps.tomli-w ]);

  iplSrc = pkgs.runCommand "slippi-ipl-src" { } ''
    mkdir -p $out
    cp ${../programs/ipl/slippi-ipl.py} $out/slippi-ipl.py
    cp ${../programs/ipl/slippi_settings.py} $out/slippi_settings.py
    cp ${../programs/ipl/slippi_update.py} $out/slippi_update.py
    cp ${../programs/ipl/slippi_login_server.py} $out/slippi_login_server.py
  '';

  ipl = pkgs.writeShellScriptBin "slippi-ipl" ''
    exec ${python}/bin/python3 ${iplSrc}/slippi-ipl.py "$@"
  '';

  slippi-runner = pkgs.callPackage ../packages/slippi-runner.nix { };

  # openbox-autostart only reads its own built-in global file and
  # $XDG_CONFIG_HOME/openbox/autostart. /etc/xdg/openbox/autostart is ignored.
  autostart = pkgs.writeText "slippi-openbox-autostart" ''
    ${pkgs.xorg.xset}/bin/xset s off -dpms
    exec ${pkgs.xterm}/bin/xterm -hold -fullscreen -fa Monospace -fs 18 -e ${ipl}/bin/slippi-ipl
  '';
in {
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  boot.kernelParams = [ "quiet" "loglevel=3" "vt.global_cursor_default=0" ];
  boot.kernelModules = [ "uinput" ];

  networking.networkmanager.enable = true;
  networking.firewall.allowedTCPPorts = [ 8080 ];
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
    ipl
    openbox
    pulseaudio
    qrencode
    slippi-runner
    xterm
    xorg.xrandr
    xorg.xset
  ];

  environment.etc."slippi-os/README".text = ''
    Settings live in /slippi/settings.toml. Game images live in /slippi/isos/.
    The next milestone is the runtime Slippi updater.
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
      ${pkgs.coreutils}/bin/chown melee:users         /slippi /slippi/isos /slippi/dolphin /slippi/replays /slippi/logs || true
      ${pkgs.coreutils}/bin/chmod 0775 /slippi || true
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
