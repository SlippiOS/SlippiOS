{ config, lib, pkgs, ... }:

let
  imageId = "slippios";
in {
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
        # Runs as root from the update request path unit. The request names
        # the staged version; record it with this boot's ID so the updater can
        # tell a completed reboot from a rollback.
        export PATH=${lib.makeBinPath [ pkgs.coreutils ]}
        target=$(cat /data/system/update/request)
        boot_id=$(cat /proc/sys/kernel/random/boot_id)
        rm -f /data/system/update/request
        mkdir -p /data/system/update
        error=
        if ${config.systemd.package}/lib/systemd/systemd-sysupdate update; then
          ${config.systemd.package}/bin/udevadm settle || true
          # A no-op exits successfully without writing anything. Only report
          # success after both the target partition and a fresh, uncounted
          # boot image exist; otherwise boot assessment has nothing to try.
          if [ ! -b "/dev/disk/by-partlabel/nix-store_''${target}" ]; then
            echo "slippios: target partition nix-store_''${target} is missing" >&2
            error="The update did not install; the new system partition is missing."
          elif [ ! -f "/boot/EFI/Linux/${imageId}_''${target}+3-0.efi" ]; then
            echo "slippios: boot image ${imageId}_''${target}+3-0.efi is missing" >&2
            error="The update did not install; the new boot image is missing."
          else
            printf '{"ok":true,"phase":"installed","version":"%s","boot_id":"%s"}\n' \
              "$target" "$boot_id" > /data/system/update/status.json
            rm -f /data/system/update/slippios_*.nix-store.raw \
                  /data/system/update/slippios_*.nix-store.raw.part \
                  /data/system/update/slippios_*.efi
          fi
        else
          error="systemd-sysupdate failed"
        fi
        if [ -n "$error" ]; then
          echo "slippios: $error" >&2
          printf '{"ok":false,"phase":"failed","version":"%s","boot_id":"%s","error":"%s"}\n' \
            "$target" "$boot_id" "$error" > /data/system/update/status.json
        fi
        chown player:users /data/system/update/status.json || true
        [ -z "$error" ]
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
}
