{ config, lib, pkgs, slippiosBuild, ... }:

let
  version = config.system.image.version;
in {
  # No interactive password login; NixOS requires this for evaluation.
  users.allowNoPasswordLogin = true;

  services.speechd.enable = false;
  i18n.supportedLocales = [ "C.UTF-8/UTF-8" "en_US.UTF-8/UTF-8" ];

  networking.useNetworkd = lib.mkForce false;

  hardware.enableRedistributableFirmware = true;

  environment.systemPackages = with pkgs; [
    xz
    zsync
  ];

  environment.etc."slippios/version".text = "${version}\n";
  environment.etc."slippios/build".text = "${slippiosBuild}\n";
}
