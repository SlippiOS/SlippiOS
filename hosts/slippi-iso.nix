{ lib, ... }:

{
  # The ISO must boot on varied hardware. Keep GPU selection generic and let
  # Mesa's modesetting driver discover the host GPU at boot.
  hardware.graphics.enable = true;
  hardware.graphics.enable32Bit = true;
  services.xserver.videoDrivers = [ "modesetting" ];

  # installation-cd-minimal.nix enables the legacy wireless module. SlippiOS
  # uses NetworkManager instead, and NixOS rejects both being enabled at once.
  networking.wireless.enable = lib.mkForce false;

  # Avoid waiting for a network connection during the live boot.
  boot.initrd.network.enable = false;
}
