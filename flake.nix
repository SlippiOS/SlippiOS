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

      packages.${system} = let
        python = pkgs.python3.withPackages (ps: [ ps.tomli-w ]);
        iplSrc = pkgs.runCommand "slippi-ipl-src" { } ''
          mkdir -p $out
          cp ${./programs/ipl/slippi-ipl.py} $out/slippi-ipl.py
          cp ${./programs/ipl/slippi_settings.py} $out/slippi_settings.py
        '';
      in {
        slippi-dolphin = pkgs.callPackage ./packages/slippi-dolphin.nix { };

        ipl = pkgs.writeShellScriptBin "slippi-ipl" ''
          exec ${python}/bin/python3 ${iplSrc}/slippi-ipl.py "$@"
        '';
      };
    };
}
