{ config, lib, pkgs, ... }:

let
  imageId = "slippios";
in {
  systemd.sysupdate = {
    enable = true;

    transfers = {
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

  systemd.timers.systemd-sysupdate.wantedBy = lib.mkForce [ ];

  systemd.services.slippios-os-update = {
    description = "Install a staged SlippiOS system update.";
    after = [ "slippi-data.service" ];
    requires = [ "slippi-data.service" ];
    serviceConfig = {
      Type = "oneshot";
      TimeoutStartSec = "30min";
      ExecStart = pkgs.writeShellScript "slippios-os-update" ''
        set -eu
        export PATH=${lib.makeBinPath [ pkgs.coreutils ]}
        target=$(cat /data/system/update/request)
        boot_id=$(cat /proc/sys/kernel/random/boot_id)
        rm -f /data/system/update/request
        mkdir -p /data/system/update
        error=
        if ${config.systemd.package}/lib/systemd/systemd-sysupdate update; then
          ${config.systemd.package}/bin/udevadm settle || true
          # Older images installed NixOS generation entries. This image boots
          # UKIs, so those entries only add noise and can outrank the UKI.
          rm -f /boot/loader/entries/nixos-generation-*.conf
          # A sysupdate no-op exits 0 without writing anything. Require both
          # the target partition and a fresh boot image before reporting success.
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
    # Ordering against slippi-data.service would create a target loop.
    # The mount unit is enough to know when /data is ready.
    after = [ "data.mount" ];
    pathConfig.PathExists = "/data/system/update/request";
  };

  systemd.services.slippios-os-reboot = {
    description = "Restart after a SlippiOS update.";
    serviceConfig.Type = "oneshot";
    serviceConfig.ExecStart = pkgs.writeShellScript "slippios-os-reboot" ''
      set -eu
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

  systemd.additionalUpstreamSystemUnits = [
    "systemd-boot-check-no-failures.service"
  ];
  systemd.services.systemd-boot-check-no-failures.wantedBy =
    [ "boot-complete.target" ];
}
