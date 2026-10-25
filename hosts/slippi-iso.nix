{ lib, pkgs, ... }:

{
  # The ISO must boot on varied hardware. Keep GPU selection generic and let
  # Mesa's modesetting driver discover the host GPU at boot.
  hardware.graphics.enable = true;
  hardware.graphics.enable32Bit = true;
  services.xserver.videoDrivers = [ "modesetting" ];

  # installation-cd-minimal.nix disables fontconfig to save space, but xterm's
  # Xft font needs it. Without this, xterm exits before the IPL can start.
  fonts.fontconfig.enable = lib.mkForce true;
  fonts.packages = [ pkgs.dejavu_fonts ];

  # installation-cd-minimal.nix enables the legacy wireless module. SlippiOS
  # uses NetworkManager so the config menu can manage Wi-Fi, and NixOS rejects
  # both being enabled at once.
  networking.wireless.enable = lib.mkForce false;

  # The manual's installer profile is self-contained. Live media cannot install
  # packages at runtime, so keep the closure minimal instead of adding more.
  isoImage.squashfsCompression = "zstd -Xcompression-level 6";

  # Boot straight into the kiosk without showing the boot menu. The installer
  # module sets its own timeout, so this has to win.
  boot.loader.timeout = lib.mkForce 0;

  # Avoid waiting for a network connection during the live boot.
  boot.initrd.network.enable = false;
}
