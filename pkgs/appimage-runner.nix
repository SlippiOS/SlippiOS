{
  appimageTools,
  buildFHSEnv,
  lib,
  pkgs,
}:

let
  exec = "${appimageTools.appimage-exec}/bin/appimage-exec.sh";

  # Slippi AppImages ship a linuxdeploy hook that prepends the FHS env's
  # /usr/lib to LD_LIBRARY_PATH. On NixOS that picks the host libcurl, which
  # lacks CURL_OPENSSL_4, instead of the AppImage's bundled libcurl. Patch the
  # hook to use the extracted AppImage's own usr/lib.
  runner = pkgs.writeShellScript "slippi-appimage-run" ''
    set -euo pipefail
    appimage=$1
    shift

    hash=$(sha256sum "$appimage" | awk '{print $1}')
    dir="''${XDG_CACHE_HOME:-$HOME/.cache}/appimage-run/$hash"
    if [ ! -x "$dir" ]; then
      rm -rf "$dir"
      mkdir -p "$(dirname "$dir")"
      ${exec} -x "$dir" "$appimage"
    fi

    # The AppImage's libcurl is the only library that must come from the
    # bundle; the rest are happier with the FHS environment's system libs.
    shim="$dir/shim"
    if [ ! -d "$shim" ]; then
      mkdir -p "$shim"
      ln -sf "$dir/usr/lib/libcurl.so.4" "$shim/libcurl.so.4"
      if command -v ldd >/dev/null 2>&1; then
        LD_LIBRARY_PATH="$dir/usr/lib" ldd "$dir/usr/lib/libcurl.so.4" 2>/dev/null |
          awk '/=>/ {print $3}' |
          while read -r lib; do
            case "$lib" in
              "$dir"/usr/lib/*) ln -sf "$lib" "$shim/$(basename "$lib")" ;;
            esac
          done
      fi
    fi

    hook="$dir/apprun-hooks/linux-env.sh"
    if [ -f "$hook" ]; then
      sed -i \
        's|export LD_LIBRARY_PATH="/usr/lib/:$LD_LIBRARY_PATH"|export LD_LIBRARY_PATH="''${SLIPPIOS_APPIMAGE_SHIM:-/usr/lib}:$LD_LIBRARY_PATH"|' \
        "$hook"
    fi

    export SLIPPIOS_APPIMAGE_SHIM="$shim"
    exec ${exec} -w "$dir" -- "$@"
  '';
in
buildFHSEnv (appimageTools.defaultFhsEnvArgs // {
  name = "slippi-appimage-run";

  targetPkgs = pkgs:
    [
      appimageTools.appimage-exec
      pkgs.gawk
      pkgs.gnused
      pkgs.libevdev
      pkgs.systemd
    ] ++ appimageTools.defaultFhsEnvArgs.targetPkgs pkgs;

  runScript = "${runner}";

  meta = {
    description = "Run Slippi Dolphin AppImages on NixOS";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
    mainProgram = "slippi-appimage-run";
  };
})
