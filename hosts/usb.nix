{ modulesPath, ... }:

{
  imports = [
    (modulesPath + "/image/repart.nix")
    (modulesPath + "/profiles/image-based-appliance.nix")
    ./usb/boot.nix
    ./usb/data.nix
    ./usb/image.nix
    ./usb/services.nix
    ./usb/update.nix
  ];

  # BORE was fastest in the clean singles and 4-player tests. Singles is the
  # priority, so release images use BORE. Dev configurations can override it.
  slippios.kernel = {
    enable = true;
    variant = "bore";
  };
}
