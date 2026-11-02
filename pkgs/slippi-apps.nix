{ lib, appimageTools, runCommand, fetchurl, unzip }:

let
  stableZip = fetchurl {
    url = "https://github.com/project-slippi/Ishiiruka/releases/download/v3.6.4/FM-Slippi-3.6.4-Linux.zip";
    hash = "sha256-yraqKM7lltWuvN1MSCcKceUytvHoSkPr1/qL9S4rH8M=";
  };

  playbackZip = fetchurl {
    url = "https://github.com/project-slippi/Ishiiruka-Playback/releases/download/v3.5.2/playback-3.5.2-Linux.zip";
    hash = "sha256-x/y6C2EU7qtaogNn2Jpmna7r1jndxDqplpiDsYxJzxw=";
  };

  betaImage = fetchurl {
    url = "https://github.com/project-slippi/dolphin/releases/download/v4.0.0-mainline-beta.19/Slippi_Netplay_Mainline-x86_64.AppImage";
    hash = "sha256-2iG4+9CLO4r/xYkQSx+QEf22ieciKdj56rtgcjWhusM=";
  };

  extract = name: src: appimage: version: runCommand name {
    meta.license = lib.licenses.gpl2Plus;
  } ''
    mkdir -p $out
    ${unzip}/bin/unzip -q ${src} -d $out
    rm -f $out/*.zsync
    chmod +x $out/${appimage}
    echo ${version} > $out/.version
  '';

  stable = extract "slippios-stable-app-3.6.4" stableZip "Slippi_Online-x86_64.AppImage" "3.6.4";

  # The AppImage runner extracts the AppImage into the profile cache on first
  # launch. Seed that cache in the image so the first launch does not spend
  # minutes unpacking it while the launching screen is visible.
  stableCache = runCommand "slippios-stable-appimage-cache" {
    meta.license = lib.licenses.gpl2Plus;
  } ''
    appimage=${stable}/Slippi_Online-x86_64.AppImage
    hash=$(sha256sum "$appimage" | cut -d' ' -f1)
    mkdir -p $out
    ${appimageTools.appimage-exec}/bin/appimage-exec.sh -x "$out/$hash" "$appimage"
    # The AppImage stores its files read-only. The runner patches its hook and
    # creates a shim inside the cache, so leave the owner write bit set.
    chmod -R u+w "$out/$hash"
  '';
in {
  inherit stable stableCache;
  playback = extract "slippios-playback-app-3.5.2" playbackZip "Slippi_Playback-x86_64.AppImage" "3.5.2";
  beta = runCommand "slippios-beta-app-4.0.0-mainline-beta.19" {
    meta.license = lib.licenses.gpl2Plus;
  } ''
    mkdir -p $out
    cp ${betaImage} $out/Slippi_Netplay_Mainline-x86_64.AppImage
    chmod +x $out/Slippi_Netplay_Mainline-x86_64.AppImage
  '';
}
