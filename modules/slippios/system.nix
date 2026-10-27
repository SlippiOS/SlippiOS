{ pkgs, slippiosPackages, ... }:

{
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

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

  systemd.tmpfiles.rules = [
    "d /home/player 0755 player users -"
    "d /home/player/.config 0755 player users -"
  ];

  system.stateVersion = "25.05";
}
