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

      # Same image, but left uncompressed. Compression only exists to keep
      # release downloads under GitHub's asset limit, so local boot tests
      # skip it and save a few minutes per build.
      nixosConfigurations.slippios-usb-dev = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = { slippiosPackages = self.packages.${system}; };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          ./hosts/dev.nix
          {
            system.image.version = version;
            image.repart.compression.enable = nixpkgs.lib.mkForce false;
          }
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

          # Keep the newest session's stderr for field diagnosis. This is
          # truncated once per boot, not once per launch, so a crash after
          # Dolphin exits is still captured.
          log=/data/system/shell.log
          if ! : >"$log" 2>/dev/null; then
            log=/tmp/slippios-shell.log
          fi

          # Dev VMs can override the shell and CLI without rebuilding the
          # image: mise run dev:push copies them to /data/dev.
          dev=/data/dev
          if [ -d "$dev" ]; then
            PATH="$dev:$PATH"
            export PATH
          fi
          shell_bin=${shellPackage}/bin/slippios-shell
          cli_bin=${cliPackage}/bin/slippios
          if [ -x "$dev/slippios-shell" ]; then
            shell_bin="$dev/slippios-shell"
          fi
          if [ -x "$dev/slippios" ]; then
            cli_bin="$dev/slippios"
          fi

          while true; do
            code=0
            SLIPPIOS_AUTOPLAY="$autoplay" \
            SLIPPIOS_SKIP_SPLASH="$splash" \
            SDL_JOYSTICK_HIDAPI_GAMECUBE=1 \
            SDL_HIDAPI_LIBUSB_GAMECUBE=1 \
              "$shell_bin" \
                --fullscreen --rendering-driver opengl3 2>>"$log" || code=$?

            # Godot must be gone before Dolphin opens the adapter: USB access
            # is exclusive, and the shell's SDL driver claims it.
            if [ "$code" -eq 42 ]; then
              "$cli_bin" play || true
              autoplay=0
              splash=1
              continue
            fi
            if [ "$code" -eq 43 ]; then
              "$cli_bin" dolphin config || true
              autoplay=0
              splash=1
              continue
            fi
            if [ "$code" -eq 44 ]; then
              ${pkgs.xterm}/bin/xterm -fullscreen -fa Monospace -fs 16 \
                -e ${pkgs.bash}/bin/bash -l || true
              autoplay=0
              splash=1
              continue
            fi
            if [ "$code" -eq 45 ]; then
              ${pkgs.xterm}/bin/xterm -fullscreen -fa Monospace -fs 16 \
                -e ${pkgs.nano}/bin/nano /data/users/default/settings.toml || true
              autoplay=0
              splash=1
              continue
            fi

            # Anything else is a crash. Leave X and hand the console to the
            # recovery service instead of showing a desktop nobody can use.
            echo "slippios-shell exited with $code" >>"$log"
            : >/data/system/shell-failed 2>/dev/null || true
            exit "$code"
          done
        '';
      };
    };
}
