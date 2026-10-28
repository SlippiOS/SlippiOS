{ config, lib, pkgs, ... }:

let
  inherit (pkgs.stdenv.hostPlatform) efiArch;

  imageId = "slippios";
  version = config.system.image.version;
  ukiFile = config.system.boot.loader.ukiFile;
  slippiApps = pkgs.callPackage ../../pkgs/slippi-apps.nix { };

  # Keep the squashfs below GitHub's 2 GiB asset limit.
  storeSize = "1900M";
in {
  image.repart = {
    name = imageId;
    split = true;
    compression = {
      enable = true;
      algorithm = "xz";
      level = 1;
    };

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

      y-melee = {
        contents = {
          "/README.txt".source = pkgs.writeText "melee-roms-readme" ''
            SlippiOS Melee drive

            Copy Melee .iso images here, then eject the drive and boot the
            console. Use the console's Melee ROMs screen to choose which image
            starts at power-on.

            Keep only Melee images on this drive. Settings, login, Dolphin and
            replays are stored on the separate SlippiOS data partition.
          '';
        };
        repartConfig = {
          Type = "EBD0A0A2-B9E5-4433-87C0-68B6B72699C7";
          Format = "vfat";
          Label = "MELEE-ROMS";
          Minimize = "off";
          SizeMinBytes = "5G";
          SizeMaxBytes = "5G";
          SplitName = "-";
        };
      };

      z-data = {
        contents = {
          # Seed the pinned apps on a fresh flash. The data partition survives
          # updates, so existing consoles keep the copy they already have.
          "/system/apps/stable".source = slippiApps.stable;
          "/system/playback/3.5.2".source = slippiApps.playback;
          "/system/playback/.version".source =
            pkgs.writeText "slippios-playback-version" "3.5.2\n";
        };
        repartConfig = {
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
  };
}
