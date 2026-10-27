{ buildGoModule }:

buildGoModule {
  pname = "slippios";
  version = "0.2.1";

  src = ../backend;
  vendorHash = null;

  # nix-portable runs unsandboxed; give Go a disposable HOME.
  preBuild = ''
    export HOME="$TMPDIR/home"
    mkdir -p "$HOME"
  '';

  subPackages = [ "cmd/slippios" "cmd/slippios-session" ];

  meta = {
    description = "SlippiOS control CLI";
    mainProgram = "slippios";
  };
}
