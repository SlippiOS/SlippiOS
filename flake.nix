{
  description = "SlippiOS: a console-style NixOS appliance for Slippi Melee";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

      python = pkgs.python3.withPackages (ps: [ ps.tomli-w ps.pyusb ps.pygame ]);

      # Sortable, unique, and short enough for a GPT partition label.
      version =
        let rev = self.rev or self.dirtyRev or null; in
        if rev != null then
          "${toString (self.lastModified or 0)}-${builtins.substring 0 7 rev}"
        else
          "dev-${toString (self.lastModified or 0)}";

      appSrc = pkgs.runCommand "slippios-apps" { } ''
        mkdir -p $out/session $out/cli
        cp ${./apps/session}/*.py $out/session/
        cp ${./apps/cli}/*.py $out/cli/
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

      nixosConfigurations.slippios-usb = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = { slippiosPackages = self.packages.${system}; };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          { system.image.version = version; }
        ];
      };

      packages.${system} = {
        appimage-runner = pkgs.callPackage ./pkgs/appimage-runner.nix { };

        session = pkgs.writeShellScriptBin "slippios-session" ''
          exec env PYTHONPATH=${appSrc}/session \
            ${python}/bin/python3 ${appSrc}/session/main.py "$@"
        '';

        cli = pkgs.writeShellScriptBin "slippios" ''
          exec env PYTHONPATH=${appSrc}/session:${appSrc}/cli \
            ${python}/bin/python3 ${appSrc}/cli/main.py "$@"
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
