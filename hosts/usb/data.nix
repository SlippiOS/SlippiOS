{ config, pkgs, ... }:

let
  version = config.system.image.version;
in {
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
      # Check the exact label. After an A/B update both store partitions
      # exist, and a glob would expand to two paths.
      store="/dev/disk/by-partlabel/nix-store_${version}"
      i=0
      while [ "$i" -lt 300 ]; do
        if [ -e "$store" ]; then
          exit 0
        fi
        ${pkgs.coreutils}/bin/sleep 0.1
        i=$((i + 1))
      done
      echo "slippios-wait-store: $store did not appear" >&2
      exit 1
    '';
  };
}
