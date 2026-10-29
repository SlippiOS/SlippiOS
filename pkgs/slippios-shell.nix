{
  lib,
  stdenv,
  godot_4,
}:

let
  theme = import ../theme/default.nix;
in

stdenv.mkDerivation {
  pname = "slippios-shell";
  version = "0.1.0";

  src = ../apps/shell;
  dontUnpack = true;

  nativeBuildInputs = [ godot_4 ];

  buildPhase = ''
    runHook preBuild

    cp -r "$src" project
    chmod -R u+w project
    sed -i "s|^boot_splash/bg_color=.*|boot_splash/bg_color=${theme.backgroundGodot}|" \
      project/project.godot
    cd project

    export HOME="$TMPDIR/home"
    export XDG_DATA_HOME="$TMPDIR/data"
    mkdir -p "$HOME" "$XDG_DATA_HOME/godot/export_templates"
    # The prebuilt upstream template cannot load the NixOS graphics stack.
    ln -s ${godot_4.export-template}/share/godot/export_templates/* \
      "$XDG_DATA_HOME/godot/export_templates/"

    godot --headless --path . --import
    godot --headless --path . --export-release "Linux" "$TMPDIR/slippios-shell"

    runHook postBuild
  '';

  installPhase = ''
    runHook preInstall
    install -Dm755 "$TMPDIR/slippios-shell" "$out/bin/slippios-shell"
    runHook postInstall
  '';

  meta = {
    description = "Compiled SlippiOS Godot shell";
    mainProgram = "slippios-shell";
    platforms = [ "x86_64-linux" ];
    license = lib.licenses.gpl3Plus;
  };
}
