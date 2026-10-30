{ lib, ... }:

{
  imports = [ ./usb.nix ];

  # The minimal image ships no app payloads. The first boot downloads the
  # stable, beta, and playback builds into /data.
  image.repart.partitions.z-data = {
    contents = lib.mkForce { };
    repartConfig = {
      SizeMinBytes = lib.mkForce "256M";
      SizeMaxBytes = lib.mkForce "256M";
    };
  };
}
