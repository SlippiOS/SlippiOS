{ config, lib, pkgs, ... }:

let
  version = config.system.image.version;
in {
  # The system auto-logs in as `player`; there is no interactive password
  # login. NixOS otherwise refuses to evaluate a system with no password on
  # root or a wheel user.
  users.allowNoPasswordLogin = true;

  # The image is a game console, not a desktop: skip the speech stack and
  # the full locale archive. Both are large and unused here.
  services.speechd.enable = false;
  i18n.supportedLocales = [ "C.UTF-8/UTF-8" "en_US.UTF-8/UTF-8" ];

  # NetworkManager owns Wi-Fi; systemd-networkd is not needed here.
  networking.useNetworkd = lib.mkForce false;

  # Firmware is needed for Wi-Fi and Bluetooth on real hardware.
  hardware.enableRedistributableFirmware = true;

  environment.systemPackages = with pkgs; [
    xz
    zsync
  ];

  environment.etc."slippios/version".text = "${version}\n";
}
