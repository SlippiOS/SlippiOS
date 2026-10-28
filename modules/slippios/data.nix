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
        /data/users/default/dolphin-stable \
        /data/users/default/replays /data/users/default/logs
      ${pkgs.coreutils}/bin/chown player:users \
        /data /data/isos /data/system /data/users /data/users/default \
        /data/users/default/dolphin /data/users/default/dolphin-stable \
        /data/users/default/replays /data/users/default/logs || true
      ${pkgs.coreutils}/bin/chmod 0775 /data || true

      main=/data/users/default/dolphin
      stable=/data/users/default/dolphin-stable
      ${pkgs.coreutils}/bin/install -d -o player -g users -m 0775 \
        "$main" "$main/Config" "$main/Config/Profiles" \
        "$main/Config/Profiles/GCPad" "$main/GameSettings" "$main/Slippi" \
        "$stable" "$stable/Config" "$stable/Slippi" \
        /data/system/apps /data/system/playback
      ${pkgs.coreutils}/bin/cp -n ${../../assets/controller-profiles}/*.ini \
        "$main/Config/Profiles/GCPad"/ || true
      ${pkgs.coreutils}/bin/chown -R player:users \
        "$main/Config" "$main/GameSettings" "$main/Slippi" || true

      # The beta/mainline and stable builds share pad profiles, Gecko codes,
      # and the Slippi login. Their Dolphin.ini, GFX.ini, and GCPadNew.ini
      # stay separate because the builds have different formats.
      if [ -e "$stable/Config/Profiles" ] && [ ! -L "$stable/Config/Profiles" ]; then
        if ${pkgs.coreutils}/bin/cp -an "$stable/Config/Profiles/." "$main/Config/Profiles/"; then
          ${pkgs.coreutils}/bin/rm -rf "$stable/Config/Profiles"
        fi
      fi
      ${pkgs.coreutils}/bin/ln -sfn ../../dolphin/Config/Profiles "$stable/Config/Profiles"

      if [ -e "$stable/GameSettings" ] && [ ! -L "$stable/GameSettings" ]; then
        if ${pkgs.coreutils}/bin/cp -an "$stable/GameSettings/." "$main/GameSettings/"; then
          ${pkgs.coreutils}/bin/rm -rf "$stable/GameSettings"
        fi
      fi
      ${pkgs.coreutils}/bin/ln -sfn ../dolphin/GameSettings "$stable/GameSettings"

      if [ -e "$stable/Slippi/user.json" ] && [ ! -L "$stable/Slippi/user.json" ]; then
        if ${pkgs.coreutils}/bin/cp -n "$stable/Slippi/user.json" "$main/Slippi/user.json"; then
          ${pkgs.coreutils}/bin/rm -f "$stable/Slippi/user.json"
        fi
      fi
      ${pkgs.coreutils}/bin/ln -sfn ../../dolphin/Slippi/user.json "$stable/Slippi/user.json"
      ${pkgs.coreutils}/bin/chown -R player:users "$stable" || true
    '';
  };
}
