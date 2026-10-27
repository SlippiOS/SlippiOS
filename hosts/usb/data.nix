{ config, pkgs, ... }:

let
  version = config.system.image.version;
in {
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

  # systemd-repart cannot cap a partition at 100% or repair the GPT backup
  # header. parted -f and resize2fs handle both.
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
        if [ $((pstart + psecs)) -ge $((dsecs - 2048)) ] \
          && [ $((partsz - fssz)) -lt $((partsz / 100)) ]; then
          exit 0
        fi
      fi
      ${pkgs.parted}/bin/parted -s -f "$disk" unit s resizepart "$partnum" 100% || true
      ${pkgs.util-linux}/bin/partx -u "$data" || true
      ${pkgs.e2fsprogs}/bin/e2fsck -f -y "$data" || true
      ${pkgs.e2fsprogs}/bin/resize2fs "$data" || true
    '';
  };

  # The GPT rewrite can race the generated /sysroot/nix/store mount.
  # Wait for the store label before allowing that mount.
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
