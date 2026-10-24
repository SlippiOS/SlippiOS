{
  description = "SlippiOS: a console-style NixOS appliance for Slippi Melee";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };
      common = ./modules/common.nix;
    in {
      nixosConfigurations = {
        slippi-iso = nixpkgs.lib.nixosSystem {
          inherit system;
          modules = [
            ({ modulesPath, ... }: {
              imports = [ (modulesPath + "/installer/cd-dvd/installation-cd-minimal.nix") ];
            })
            common
            ./hosts/slippi-iso.nix
          ];
        };
      };

      packages.${system}.ipl = pkgs.python3Packages.buildPythonApplication {
        pname = "slippi-ipl";
        version = "0.1.0";
        pyproject = false;
        src = ./programs/ipl;
        installPhase = ''
          install -Dm755 slippi-ipl.py $out/bin/slippi-ipl
        '';
      };
    };
}
