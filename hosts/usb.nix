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

  # The LTS CachyOS kernel was the fastest of the tested kernels and had the
  # most stable 4-player frame pacing. Dev configurations can override this.
  slippios.kernel = {
    enable = true;
    variant = "lts";
  };
}
