{ appimageTools, fetchurl, lib }:

let
  version = "4.0.0-mainline-beta.19";
in appimageTools.wrapType2 {
  pname = "slippi-dolphin";
  inherit version;

  src = fetchurl {
    url = "https://github.com/project-slippi/dolphin/releases/download/v${version}/Slippi_Netplay_Mainline-x86_64.AppImage";
    hash = "sha256-2iG4+9CLO4r/xYkQSx+QEf22ieciKdj56rtgcjWhusM=";
  };

  meta = {
    description = "Slippi Mainline Dolphin (prebuilt release)";
    homepage = "https://github.com/project-slippi/dolphin";
    license = lib.licenses.gpl2Plus;
    platforms = [ "x86_64-linux" ];
    mainProgram = "slippi-dolphin";
  };
}
