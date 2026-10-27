{ config, lib, ... }:

let
  imageId = "slippios";
  version = config.system.image.version;
in {
  system.image.id = imageId;
  system.image.version = lib.mkDefault "dev";
  system.nixos.distroName = "SlippiOS";

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
