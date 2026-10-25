{ buildGoModule }:

buildGoModule {
  pname = "slippios";
  version = "0.2.0";

  src = ../backend;
  vendorHash = null;

  subPackages = [ "cmd/slippios" "cmd/slippios-session" ];

  meta = {
    description = "SlippiOS control CLI";
    mainProgram = "slippios";
  };
}
