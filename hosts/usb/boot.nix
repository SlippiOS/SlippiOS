{ config, lib, ... }:

let
  imageId = "slippios";
  version = config.system.image.version;
in {
  system.image.id = imageId;
  system.image.version = lib.mkDefault "dev";
  system.nixos.distroName = "SlippiOS";

  # UEFI only. The removable-media path means the image boots without
  # writing firmware variables, which is what makes it usable from a USB
  # stick on arbitrary machines.
  boot.loader.grub.enable = false;
  boot.uki.name = imageId;

  boot.initrd.systemd.enable = true;
  boot.supportedFilesystems = [ "squashfs" "ext4" "vfat" ];
  boot.initrd.availableKernelModules = [
    "squashfs"
    "ext4"
    "vfat"
    "fat"
    "nls_cp437"
    "nls_iso8859_1"
    "usb_storage"
    "uas"
    "xhci_pci"
    "ahci"
    "ata_piix"
    "sd_mod"
    "sr_mod"
    "nvme"
    "virtio"
    "virtio_pci"
    "virtio_blk"
    "virtio_scsi"
    "virtio_net"
    # Display drivers for the boot splash in VMs. Real hardware uses the
    # firmware framebuffer through the built-in simpledrm until the native
    # driver loads in stage 2; including native GPU drivers here would pull
    # hundreds of megabytes of firmware into the 512M ESP.
    "virtio_gpu"
    "bochs"
    "loop"
  ];

  fileSystems = {
    "/" = {
      device = "/dev/disk/by-partlabel/root";
      fsType = "ext4";
    };
    "/boot" = {
      device = "/dev/disk/by-partlabel/boot";
      fsType = "vfat";
    };
    # The label includes the image version. The matching UKI embeds this
    # label, which is how an A/B boot picks its own read-only store.
    "/nix/store" = {
      device = "/dev/disk/by-partlabel/nix-store_${version}";
      fsType = "squashfs";
      neededForBoot = true;
    };
    "/data" = {
      device = "/dev/disk/by-label/SLIPPIOS";
      fsType = "ext4";
      options = [ "nofail" "x-systemd.device-timeout=10" ];
    };
  };
}
