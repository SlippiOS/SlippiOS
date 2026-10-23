{ ... }:

{
  # The ISO must boot on varied hardware. Keep GPU selection generic and let
  # Mesa's modesetting driver discover the host GPU at boot.
  hardware.graphics.enable = true;
  hardware.graphics.enable32Bit = true;
  services.xserver.videoDrivers = [ "modesetting" ];

  # Avoid waiting for a network connection during the live boot.
  boot.initrd.network.enable = false;
}
