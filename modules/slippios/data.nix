{ pkgs, ... }:

{
  systemd.services.slippi-data = {
    description = "Prepare SlippiOS data directory";
    wantedBy = [ "multi-user.target" ];
    before = [ "display-manager.service" ];
    after = [ "local-fs.target" ];
    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };
    script = ''
      ${pkgs.coreutils}/bin/mkdir -p /data
      if [ -b /dev/disk/by-label/SLIPPIOS ]; then
        ${pkgs.util-linux}/bin/mountpoint -q /data || \
          ${pkgs.util-linux}/bin/mount -L SLIPPIOS /data || true
      fi
      ${pkgs.coreutils}/bin/rm -f /data/system/shell-failed
      ${pkgs.coreutils}/bin/mkdir -p \
        /data/isos /data/system /data/users/default/dolphin \
        /data/users/default/replays /data/users/default/logs
      ${pkgs.coreutils}/bin/chown player:users \
        /data /data/isos /data/system /data/users /data/users/default \
        /data/users/default/dolphin /data/users/default/replays \
        /data/users/default/logs || true
      ${pkgs.coreutils}/bin/chmod 0775 /data || true

      config_dir=/data/users/default/dolphin/Config
      profile_dir=$config_dir/Profiles/GCPad
      ${pkgs.coreutils}/bin/install -d -o player -g users -m 0775 \
        "$config_dir" "$config_dir/Profiles" "$profile_dir"
      ${pkgs.coreutils}/bin/cp -n ${../../assets/controller-profiles}/*.ini "$profile_dir"/ || true
      ${pkgs.coreutils}/bin/chown -R player:users "$config_dir" || true
    '';
  };
}
