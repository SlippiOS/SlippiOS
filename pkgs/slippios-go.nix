{ lib, buildGoModule }:

buildGoModule {
  pname = "slippios-go";
  version = "0.1.0";

  src = ../backend;
  vendorHash = null;

  subPackages = [ "cmd/slippios" ];

  postInstall = ''
    mv $out/bin/slippios $out/bin/slippios-go
  '';

  meta = {
    description = "SlippiOS control CLI (Go port)";
    mainProgram = "slippios-go";
  };
}
