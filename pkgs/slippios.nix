{ buildGoModule }:

buildGoModule {
  pname = "slippios";
  version = "0.2.1";

  src = ../backend;
  vendorHash = null;

  # nix-portable runs unsandboxed, so the Go toolchain can create $HOME/.config
  # in the shared build root. Give it a disposable HOME instead.
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
