{ runCommand, fetchurl, unzip }:

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

  extract = name: src: appimage: runCommand name { } ''
    mkdir -p $out
    ${unzip}/bin/unzip -q ${src} -d $out
    rm -f $out/*.zsync
    chmod +x $out/${appimage}
  '';
in {
  stable = extract "slippios-stable-app-3.6.4" stableZip "Slippi_Online-x86_64.AppImage";
  playback = extract "slippios-playback-app-3.5.2" playbackZip "Slippi_Playback-x86_64.AppImage";
  beta = runCommand "slippios-beta-app-4.0.0-mainline-beta.19" { } ''
    mkdir -p $out
    cp ${betaImage} $out/Slippi_Netplay_Mainline-x86_64.AppImage
    chmod +x $out/Slippi_Netplay_Mainline-x86_64.AppImage
  '';
}
