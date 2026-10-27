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
}
