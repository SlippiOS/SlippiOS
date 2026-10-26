{
  appimageTools,
  buildFHSEnv,
  lib,
}:

# Runs Slippi Dolphin AppImages that are downloaded at runtime, so the image
# does not need rebuilding for every upstream Dolphin release. Start from
# nixpkgs' tested AppImage environment and add the input/udev libraries
# Dolphin needs for controllers.
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
