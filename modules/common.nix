{ config, pkgs, lib, ... }:

let
  ipl = pkgs.writeTextFile {
    name = "slippi-ipl.py";
    executable = true;
    destination = "/bin/slippi-ipl";
    text = builtins.readFile ../programs/ipl/slippi-ipl.py;
  };
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

  environment.etc."xdg/openbox/autostart".text = ''
    ${pkgs.xorg.xset}/bin/xset s off -dpms
    exec ${pkgs.xterm}/bin/xterm -fullscreen -fa Monospace -fs 18 -e ${ipl}/bin/slippi-ipl
  '';

  # Development-only placeholder until the real Dolphin package is selected.
  environment.etc."slippi-os/README".text = ''
    The IPL is active. The Slippi launcher is not packaged yet.
    The next hardware milestone is exclusive USB hand-off for 057e:0337.
  '';

  system.stateVersion = "25.05";
}
