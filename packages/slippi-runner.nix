{ buildFHSEnv, appimageTools, lib }:

buildFHSEnv {
  name = "slippi-appimage-run";

  # Extracts and runs an AppImage supplied at runtime. This lets SlippiOS
  # follow upstream releases without rebuilding the image.
  runScript = "${appimageTools.appimage-exec}/bin/appimage-exec.sh";

  targetPkgs = pkgs: with pkgs; [
    appimageTools.appimage-exec
    bashInteractive
    coreutils

    alsa-lib
    libpulseaudio

    libGL
    libglvnd
    vulkan-loader
    libdrm
    libgbm
    libxkbcommon

    libusb1
    systemd
    libevdev

    gtk3
    glib
    gdk-pixbuf
    freetype
    fontconfig
    zlib
    bzip2
    dbus

    xorg.libX11
    xorg.libXext
    xorg.libXrandr
    xorg.libXcursor
    xorg.libXi
    xorg.libXinerama
    xorg.libXrender
    xorg.libXfixes
    xorg.libXcomposite
    xorg.libXtst
    xorg.libXScrnSaver
    xorg.libXxf86vm
    xorg.libxcb
    xorg.xcbutil
    xorg.xcbutilimage
    xorg.xcbutilkeysyms
    xorg.xcbutilrenderutil
    xorg.xcbutilwm
    xorg.xkeyboardconfig
    xorg.xrandr

    iana-etc
    krb5
    gsettings-desktop-schemas
    hicolor-icon-theme
    zenity
    which
    perl
    xdg-utils
  ];

  meta = {
    description = "Run Slippi Dolphin AppImages on NixOS";
    license = lib.licenses.mit;
    platforms = [ "x86_64-linux" ];
    mainProgram = "slippi-appimage-run";
  };
}
