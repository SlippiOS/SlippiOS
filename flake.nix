{
  description = "SlippiOS: a console-style NixOS appliance for Slippi Melee";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

      cliPackage = pkgs.callPackage ./pkgs/slippios.nix { };
      shellPackage = pkgs.callPackage ./pkgs/slippios-shell.nix { };

      # Sortable, unique, and short enough for a GPT partition label.
      version =
        let rev = self.rev or self.dirtyRev or null; in
        if rev != null then
          "${toString (self.lastModified or 0)}-${builtins.substring 0 7 rev}"
        else
          "dev-${toString (self.lastModified or 0)}";
    in {
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
        slippios = cliPackage;
        session = cliPackage;
        cli = cliPackage;
        shell-export = shellPackage;

        shell = pkgs.writeShellScriptBin "slippios-shell" ''
          autoplay=1
          splash=0
          while true; do
            code=0
            SLIPPIOS_AUTOPLAY="$autoplay" \
            SLIPPIOS_SKIP_SPLASH="$splash" \
            SDL_JOYSTICK_HIDAPI_GAMECUBE=1 \
            SDL_HIDAPI_LIBUSB_GAMECUBE=1 \
              ${shellPackage}/bin/slippios-shell \
                --fullscreen --rendering-driver opengl3 || code=$?
            if [ "$code" -ne 42 ]; then
              exit "$code"
            fi

            # Godot must be gone before Dolphin opens the adapter: USB access
            # is exclusive, and the shell's SDL driver claims it.
            ${cliPackage}/bin/slippios play || true
            autoplay=0
            splash=1
          done
        '';
      };
    };
}
