{ config, lib, pkgs, modulesPath, ... }:

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

  # The Melee image library is a plain FAT32 volume so it can be edited from
  # any PC. Mount it over the /data/isos directory created by slippi-data.
  systemd.mounts = [{
    what = "/dev/disk/by-label/MELEE-ROMS";
    where = "/data/isos";
    type = "vfat";
    options = "nofail,uid=1000,gid=100,fmask=0133,dmask=0022";
    wantedBy = [ "multi-user.target" ];
    unitConfig = {
      After = [ "slippi-data.service" ];
      Requires = [ "slippi-data.service" ];
      Before = [ "display-manager.service" ];
    };
  }];

  # Grow the data partition into the unused space on the stick before the
  # root is mounted. systemd-repart cannot parse a percentage as a maximum
  # size on this systemd version, and it does not repair the GPT backup
  # header after flashing to a larger stick. parted -f does both; resize2fs
  # then grows the ext4 filesystem inside the last partition.
  boot.initrd.systemd.storePaths = [
    pkgs.coreutils
    pkgs.e2fsprogs
    pkgs.parted
    pkgs.util-linux
    pkgs.gawk
  ];
  boot.initrd.systemd.services.slippios-grow-data = {
    description = "Grow the SlippiOS data partition";
    wantedBy = [ "initrd-root-fs.target" ];
    after = [ "sysroot.mount" ];
    before = [ "initrd-switch-root.target" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    script = ''
      set -eu
      # The GPT rewrite briefly removes the other partition nodes; the
      # sliippios-wait-store unit waits for them before the store mount.
      root=$(${pkgs.coreutils}/bin/readlink -f /dev/disk/by-partlabel/root) || exit 0
      data=$(${pkgs.coreutils}/bin/readlink -f /dev/disk/by-partlabel/SLIPPIOS) || exit 0
      disk=/dev/$(${pkgs.coreutils}/bin/basename "$(${pkgs.coreutils}/bin/readlink -f "/sys/class/block/$(${pkgs.coreutils}/bin/basename "$root")/..")")
      partnum=$(${pkgs.coreutils}/bin/cat "/sys/class/block/$(${pkgs.coreutils}/bin/basename "$data")/partition")
      diskname=$(${pkgs.coreutils}/bin/basename "$disk")
      pstart=$(${pkgs.coreutils}/bin/cat "/sys/class/block/$(${pkgs.coreutils}/bin/basename "$data")/start")
      psecs=$(${pkgs.coreutils}/bin/cat "/sys/class/block/$(${pkgs.coreutils}/bin/basename "$data")/size")
      dsecs=$(${pkgs.coreutils}/bin/cat "/sys/class/block/$diskname/size")
      partsz=$(${pkgs.util-linux}/bin/blockdev --getsize64 "$data")
      info=$(${pkgs.e2fsprogs}/bin/dumpe2fs -h "$data" 2>/dev/null || true)
      blocks=$(printf '%s\n' "$info" | ${pkgs.gawk}/bin/awk '/^Block count:/ {print $3}')
      bsize=$(printf '%s\n' "$info" | ${pkgs.gawk}/bin/awk '/^Block size:/ {print $3}')
      if [ -n "$blocks" ] && [ -n "$bsize" ]; then
        fssz=$((blocks * bsize))
        # Skip only when the partition already reaches the end of the disk and
        # the filesystem fills the partition.
        if [ $((pstart + psecs)) -ge $((dsecs - 2048)) ] \
          && [ $((partsz - fssz)) -lt $((partsz / 100)) ]; then
          exit 0
        fi
      fi
      ${pkgs.parted}/bin/parted -s -f "$disk" unit s resizepart "$partnum" 100% || true
      ${pkgs.util-linux}/bin/partx -u "$data" || true
      # resize2fs refuses to grow an unclean filesystem; this only runs when
      # the partition size and filesystem size disagree.
      ${pkgs.e2fsprogs}/bin/e2fsck -f -y "$data" || true
      ${pkgs.e2fsprogs}/bin/resize2fs "$data" || true
    '';
  };

  # Growing the data partition rewrites the GPT and briefly removes the other
  # partition nodes. The generated /sysroot/nix/store mount can race that and
  # fail into the initrd emergency shell. Wait for the store label to come
  # back before allowing that mount.
  boot.initrd.systemd.services.slippios-wait-store = {
    description = "Wait for the SlippiOS store partition";
    wantedBy = [ "initrd-root-fs.target" ];
    after = [ "slippios-grow-data.service" "systemd-udev-trigger.service" ];
    before = [ "sysroot-nix-store.mount" "initrd-switch-root.target" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    script = ''
      set -eu
      i=0
      while [ "$i" -lt 100 ]; do
        if [ -e /dev/disk/by-partlabel/nix-store_* ]; then
          ${config.systemd.package}/bin/udevadm settle --timeout=1 || true
          exit 0
        fi
        ${config.systemd.package}/bin/udevadm settle --timeout=1 || true
        ${pkgs.coreutils}/bin/sleep 0.1
        i=$((i + 1))
      done
      echo "slippios-wait-store: store partition did not appear" >&2
      exit 1
    '';
  };

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

  # A staged payload is installed by the updater writing
  # /data/system/update/request. This keeps the downloader (and the shell)
  # unprivileged; only this oneshot runs as root.
  systemd.services.slippios-os-update = {
    description = "Install a staged SlippiOS system update.";
    after = [ "slippi-data.service" ];
    requires = [ "slippi-data.service" ];
    serviceConfig = {
      Type = "oneshot";
      TimeoutStartSec = "30min";
      ExecStart = pkgs.writeShellScript "slippios-os-update" ''
        set -eu
        # Runs as root from the update request path unit.
        export PATH=${lib.makeBinPath [ pkgs.coreutils ]}
        rm -f /data/system/update/request
        mkdir -p /data/system/update
        if ${config.systemd.package}/bin/systemd-sysupdate update; then
          printf '{"ok":true,"phase":"installed"}\n' > /data/system/update/status.json
          rm -f /data/system/update/slippios_*.nix-store.raw \
                /data/system/update/slippios_*.nix-store.raw.part \
                /data/system/update/slippios_*.efi
        else
          printf '{"ok":false,"phase":"failed","error":"systemd-sysupdate failed"}\n' \
            > /data/system/update/status.json
        fi
        chown player:users /data/system/update/status.json || true
      '';
    };
  };
  systemd.paths.slippios-os-update = {
    description = "Watch for a staged update request";
    wantedBy = [ "paths.target" ];
    # Ordering against slippi-data.service here would close a loop:
    # basic.target -> paths.target -> this path unit -> slippi-data ->
    # basic.target. The mount unit is enough to know when /data is ready.
    after = [ "data.mount" ];
    pathConfig.PathExists = "/data/system/update/request";
  };

  systemd.services.slippios-os-reboot = {
    description = "Restart after a SlippiOS update.";
    serviceConfig.Type = "oneshot";
    serviceConfig.ExecStart = pkgs.writeShellScript "slippios-os-reboot" ''
      set -eu
      # Requested by the updater after staging.
      rm -f /data/system/update/reboot
      ${config.systemd.package}/bin/systemctl reboot --no-block
    '';
  };
  systemd.paths.slippios-os-reboot = {
    description = "Watch for the update reboot request";
    wantedBy = [ "paths.target" ];
    after = [ "data.mount" ];
    pathConfig.PathExists = "/data/system/update/reboot";
  };

  # Automatic boot assessment. A newly written UKI gets three attempts;
  # systemd-bless-boot clears the counter once the boot completes, and
  # systemd-boot falls back to the previous good UKI otherwise.
  systemd.additionalUpstreamSystemUnits = [
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
