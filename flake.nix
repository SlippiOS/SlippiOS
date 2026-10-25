{
  description = "SlippiOS: a console-style NixOS appliance for Slippi Melee";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

      python = pkgs.python3.withPackages (ps: [ ps.tomli-w ps.pyusb ps.pygame ]);

      appSrc = pkgs.runCommand "slippios-apps" { } ''
        mkdir -p $out
        cp ${./apps/session}/*.py $out/
        cp ${./apps/ctl}/main.py $out/ctl.py
      '';
    in {
      nixosConfigurations.slippios-live = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = { slippiosPackages = self.packages.${system}; };
        modules = [
          ({ modulesPath, ... }: {
            imports = [ (modulesPath + "/installer/cd-dvd/installation-cd-minimal.nix") ];
          })
          ./modules/slippios.nix
          ./hosts/live.nix
        ];
      };

      packages.${system} = {
        appimage-runner = pkgs.callPackage ./pkgs/appimage-runner.nix { };

        session = pkgs.writeShellScriptBin "slippios-session" ''
          exec ${python}/bin/python3 ${appSrc}/main.py "$@"
        '';

        ctl = pkgs.writeShellScriptBin "slippios-ctl" ''
          exec ${python}/bin/python3 ${appSrc}/ctl.py "$@"
        '';

        shell = pkgs.writeShellScriptBin "slippios-shell" ''
          runtime=$(${pkgs.coreutils}/bin/mktemp -d)
          ${pkgs.coreutils}/bin/cp -r ${./apps/shell}/. "$runtime"/
          exec ${pkgs.godot_4}/bin/godot --path "$runtime" \
            --fullscreen --rendering-driver opengl3
        '';
      };
    };
}
