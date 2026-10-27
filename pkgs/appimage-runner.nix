{
  appimageTools,
  buildFHSEnv,
  lib,
}:

# Runs runtime-downloaded Dolphin AppImages without rebuilding the image.
buildFHSEnv (appimageTools.defaultFhsEnvArgs // {
  name = "slippi-appimage-run";

  targetPkgs = pkgs:
    [
      appimageTools.appimage-exec
      pkgs.libevdev
      pkgs.systemd
    ] ++ appimageTools.defaultFhsEnvArgs.targetPkgs pkgs;

  runScript = "${appimageTools.appimage-exec}/bin/appimage-exec.sh";

  meta = {
    description = "Run Slippi Dolphin AppImages on NixOS";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
    mainProgram = "slippi-appimage-run";
  };
})
