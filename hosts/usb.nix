{ config, lib, pkgs, modulesPath, ... }:

let
  inherit (pkgs.stdenv.hostPlatform) efiArch;

  imageId = "slippios";
  version = config.system.image.version;
  ukiFile = config.system.boot.loader.ukiFile;

  # Keep every published partition image below GitHub's 2 GiB per-asset
  # limit. The squashfs is compressed, so this is also enough room for a
  # substantially larger uncompressed Nix store.
  storeSize = "1800M";
in {
  imports = [
    (modulesPath + "/image/repart.nix")
    (modulesPath + "/profiles/image-based-appliance.nix")
  ];

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
    "usb_storage"
    "uas"
    "xhci_pci"
    "sd_mod"
    "sr_mod"
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

  # Grow the data partition into the unused space on the stick. This runs
  # from the initrd, before the root and data filesystems are mounted.
  systemd.repart.partitions."50-data" = {
    Type = "linux-generic";
    Label = "SLIPPIOS";
    Format = "ext4";
    SizeMinBytes = "1G";
    SizeMaxBytes = "100%";
    GrowFileSystem = "yes";
  };
  boot.initrd.systemd.repart.enable = true;

  image.repart = {
    name = imageId;
    split = true;
    compression = {
      enable = true;
      algorithm = "xz";
      level = 3;
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
        stripNixStorePrefix = true;
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

      data.repartConfig = {
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

  # The update payload is built from the split image plus the matching UKI
  # by CI; the on-device updater stages those files in
  # /data/system/update before invoking systemd-sysupdate.

  systemd.sysupdate = {
    enable = true;

    transfers = {
      # The OS updater downloads and verifies the image into this
      # directory, then asks systemd-sysupdate to install it. Keeping the
      # source local is what lets us use zsync instead of downloading the
      # whole partition every time.
      "10-nix-store" = {
        Source = {
          Type = "regular-file";
          Path = "/data/system/update";
          MatchPattern = [ "${imageId}_@v.nix-store.raw" ];
        };
        Target = {
          Type = "partition";
          Path = "auto";
          MatchPattern = "nix-store_@v";
          MatchPartitionType = "linux-generic";
          ReadOnly = "yes";
          InstancesMax = 2;
        };
        Transfer.Verify = "no";
      };

      "20-boot-image" = {
        Source = {
          Type = "regular-file";
          Path = "/data/system/update";
          MatchPattern = [ "${imageId}_@v.efi" ];
        };
        Target = {
          Type = "regular-file";
          Path = "/EFI/Linux";
          PathRelativeTo = "boot";
          MatchPattern = [
            "${imageId}_@v+@l-@d.efi"
            "${imageId}_@v+@l.efi"
            "${imageId}_@v.efi"
          ];
          Mode = "0444";
          TriesLeft = 3;
          TriesDone = 0;
          InstancesMax = 2;
        };
        Transfer.Verify = "no";
      };
    };
  };

  # The shell asks for updates explicitly; don't install a leftover
  # payload at an arbitrary time.
  systemd.timers.systemd-sysupdate.wantedBy = lib.mkForce [ ];

  # Automatic boot assessment. A newly written UKI gets three attempts;
  # systemd-bless-boot clears the counter once the boot completes, and
  # systemd-boot falls back to the previous good UKI otherwise.
  systemd.additionalUpstreamSystemUnits = [
    "systemd-bless-boot.service"
    "systemd-boot-check-no-failures.service"
  ];
  systemd.services.systemd-boot-check-no-failures.wantedBy =
    [ "boot-complete.target" ];

  # The appliance auto-logs in as `player`; there is no interactive
  # password login. NixOS otherwise refuses to evaluate a system with no
  # password on root or a wheel user.
  users.allowNoPasswordLogin = true;

  # The image is a game console, not a desktop: skip the speech stack and
  # the full locale archive. Both are large and unused here.
  services.speechd.enable = false;
  i18n.supportedLocales = [ "C.UTF-8/UTF-8" "en_US.UTF-8/UTF-8" ];

  # NetworkManager owns Wi-Fi; systemd-networkd is not needed here.
  networking.useNetworkd = lib.mkForce false;

  # Firmware is needed for Wi-Fi and Bluetooth on real hardware.
  hardware.enableRedistributableFirmware = true;

  environment.systemPackages = with pkgs; [
    xz
    zsync
  ];

  environment.etc."slippios/version".text = "${version}\n";
}
