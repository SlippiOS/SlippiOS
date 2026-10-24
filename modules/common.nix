{ config, pkgs, lib, ... }:

let
  iplScript = pkgs.writeText "slippi-ipl.py"
    (builtins.readFile ../programs/ipl/slippi-ipl.py);

  # NixOS has no /usr/bin/python3, so a bare `#!/usr/bin/env python3` shebang
  # fails. Always invoke the interpreter by store path.
  ipl = pkgs.writeShellScriptBin "slippi-ipl" ''
    exec ${pkgs.python3}/bin/python3 ${iplScript} "$@"
  '';

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
    xterm
    xorg.xset
  ];

  environment.etc."slippi-os/README".text = ''
    The IPL is active. The Slippi launcher is not packaged yet.
    The next hardware milestone is exclusive USB hand-off for 057e:0337.
  '';

  systemd.tmpfiles.rules = [
    "d /home/melee 0755 melee melee -"
    "d /home/melee/.config 0755 melee melee -"
    "d /home/melee/.config/openbox 0755 melee melee -"
    "L+ /home/melee/.config/openbox/autostart - - - - ${autostart}"
  ];

  system.stateVersion = "25.05";
}
