{ config, lib, pkgs, ... }:

let
  inherit (pkgs.stdenv.hostPlatform) efiArch;

  imageId = "slippios";
  version = config.system.image.version;
  ukiFile = config.system.boot.loader.ukiFile;

  # Keep every published partition image below GitHub's 2 GiB per-asset
  # limit. The squashfs is compressed, so this is also enough room for a
  # substantially larger uncompressed Nix store.
  storeSize = "1900M";
in {
  image.repart = {
    name = imageId;
    split = true;
    compression = {
      enable = true;
      algorithm = "xz";
      # Most of the outer image is either zeros (the empty slot and the
      # growable data partition) or an already-compressed squashfs. Level 1
      # is dramatically faster than level 3 and still keeps the image under
      # GitHub's 2 GiB asset limit.
      level = 1;
    };

    # The default mksquashfs compression is gzip. xz with the x86 BCJ
    # filter gives a noticeably smaller store image for this binary-heavy
    # closure, which keeps the update asset under the free-host size limit.
    mkfsOptions.squashfs = [ "-comp" "xz" "-Xbcj" "x86" ];

    partitions = {
      esp = {
        contents = {
          "/EFI/BOOT/BOOT${lib.toUpper efiArch}.EFI".source =
            "${pkgs.systemd}/lib/systemd/boot/efi/systemd-boot${efiArch}.efi";

          "/EFI/Linux/${ukiFile}".source =
            "${config.system.build.uki}/${ukiFile}";

          "/loader/loader.conf".source = builtins.toFile "loader.conf" ''
            timeout 1
            editor no
            auto-firmware no
          '';
        };
        repartConfig = {
          Type = "esp";
          Label = "boot";
          Format = "vfat";
          SizeMinBytes = "512M";
          SizeMaxBytes = "512M";
          SplitName = "-";
        };
      };

      # Slot A of the immutable system store. Slot B is the `empty`
      # partition, which systemd-sysupdate fills on the first update.
      nix-store = {
        storePaths = [ config.system.build.toplevel ];
        nixStorePrefix = "/";
        repartConfig = {
          Type = "linux-generic";
          Label = "nix-store_${version}";
          Minimize = "off";
          SizeMinBytes = storeSize;
          SizeMaxBytes = storeSize;
          Format = "squashfs";
          ReadOnly = "yes";
          SplitName = "nix-store";
        };
      };

      empty.repartConfig = {
        Type = "linux-generic";
        Label = "_empty";
        Minimize = "off";
        SizeMinBytes = storeSize;
        SizeMaxBytes = storeSize;
        SplitName = "-";
      };

      root.repartConfig = {
        Type = "root";
        Format = "ext4";
        Label = "root";
        Minimize = "off";
        SizeMinBytes = "1G";
        SizeMaxBytes = "1G";
        SplitName = "-";
      };

      # User-visible Melee image library. It is deliberately FAT32 so the
      # stick can be plugged into any PC and ISOs can be dragged in. The
      # running system mounts it over /data/isos.
      y-melee = {
        contents = {
          "/README.txt".source = pkgs.writeText "melee-roms-readme" ''
            SlippiOS Melee drive

            Copy Melee .iso images here, then eject the drive and boot the
            console. Use the console's Melee ROMs screen to choose which image
            starts at power-on.

            Keep only Melee images on this drive. Settings, login, Dolphin and
            replays are stored on the separate SLIPPIOS data partition.
          '';
        };
        repartConfig = {
          # Microsoft basic data so Windows and macOS auto-mount the FAT32
          # volume. A linux-generic type would be ignored by both.
          Type = "EBD0A0A2-B9E5-4433-87C0-68B6B72699C7";
          Format = "vfat";
          Label = "MELEE-ROMS";
          Minimize = "off";
          SizeMinBytes = "5G";
          SizeMaxBytes = "5G";
          SplitName = "-";
        };
      };

      z-data.repartConfig = {
        Type = "linux-generic";
        Format = "ext4";
        Label = "SLIPPIOS";
        Minimize = "off";
        SizeMinBytes = "1G";
        SizeMaxBytes = "1G";
        SplitName = "-";
      };
    };
  };
}
