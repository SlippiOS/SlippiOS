{
  description = "SlippiOS: a Melee console built on NixOS";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
  inputs.nix-cachyos-kernel.url = "github:xddxdd/nix-cachyos-kernel/release";

  outputs = inputs@{ self, nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

      cliPackage = pkgs.callPackage ./pkgs/slippios.nix { };
      shellPackage = pkgs.callPackage ./pkgs/slippios-shell.nix { };

      # Semver is the user-facing and update version. Dev builds use the next
      # patch plus the checkout timestamp so they sort above the release they
      # are based on.
      releaseVersion = nixpkgs.lib.strings.trim (builtins.readFile ./VERSION);
      buildRevision = self.rev or self.dirtyRev or "unknown";
      buildSuffix = builtins.substring 0 7 (builtins.baseNameOf (builtins.unsafeDiscardStringContext self.outPath));
      devVersion = "${releaseVersion}.${buildSuffix}";
      slippiosBuild = buildRevision;

      cachyosKernels = inputs.nix-cachyos-kernel.legacyPackages.${system};
    in {
      nixosConfigurations.slippios-usb = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = {
          inherit cachyosKernels slippiosBuild;
          slippiosPackages = self.packages.${system};
        };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          { system.image.version = releaseVersion; }
        ];
      };

      # Uncompressed image for faster local boots.
      nixosConfigurations.slippios-usb-dev = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = {
          inherit cachyosKernels slippiosBuild;
          slippiosPackages = self.packages.${system};
        };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          ./hosts/dev.nix
          {
            system.image.version = devVersion;
            image.repart.compression.enable = nixpkgs.lib.mkForce false;
          }
        ];
      };

      # Kernel comparison images. These share the dev layout but override the
      # CachyOS kernel selection from hosts/dev.nix.
      nixosConfigurations.slippios-usb-dev-default = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = {
          inherit cachyosKernels slippiosBuild;
          slippiosPackages = self.packages.${system};
        };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          ./hosts/dev.nix
          {
            system.image.version = devVersion;
            image.repart.compression.enable = nixpkgs.lib.mkForce false;
            slippios.kernel.enable = nixpkgs.lib.mkForce false;
          }
        ];
      };

      nixosConfigurations.slippios-usb-dev-bore = nixpkgs.lib.nixosSystem {
        inherit system;
        specialArgs = {
          inherit cachyosKernels slippiosBuild;
          slippiosPackages = self.packages.${system};
        };
        modules = [
          ./modules/slippios.nix
          ./hosts/usb.nix
          ./hosts/dev.nix
          {
            system.image.version = devVersion;
            image.repart.compression.enable = nixpkgs.lib.mkForce false;
            slippios.kernel.enable = nixpkgs.lib.mkForce true;
            slippios.kernel.variant = nixpkgs.lib.mkForce "bore";
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

          # Keep the newest session's stderr for field diagnosis.
          log=/data/system/shell.log
          if ! : >"$log" 2>/dev/null; then
            log=/tmp/slippios-shell.log
          fi

          # /data/dev is a development-only override. The marker is only
          # present in dev images, so a release image never runs binaries from
          # the writable data partition.
          dev=""
          if [ -e /etc/slippios/dev ]; then
            dev=/data/dev
            if [ -d "$dev" ]; then
              PATH="$dev:$PATH"
              export PATH
            fi
          fi
          shell_bin=${shellPackage}/bin/slippios-shell
          cli_bin=${cliPackage}/bin/slippios
          if [ -n "$dev" ] && [ -x "$dev/slippios-shell" ]; then
            shell_bin="$dev/slippios-shell"
          fi
          if [ -n "$dev" ] && [ -x "$dev/slippios" ]; then
            cli_bin="$dev/slippios"
          fi

          while true; do
            resolution=$(${pkgs.xrandr}/bin/xrandr --current 2>/dev/null |
              ${pkgs.gawk}/bin/awk '/ connected/ {
                for (i = 1; i <= NF; i++)
                  if ($i ~ /^[0-9]+x[0-9]+\+/) { print $i; exit }
              }' |
              ${pkgs.coreutils}/bin/cut -d+ -f1)
            shell_args=(--rendering-driver opengl3)
            if [ -n "$resolution" ]; then
              shell_args=(--resolution "$resolution" "''${shell_args[@]}")
            fi
            code=0
            SLIPPIOS_AUTOPLAY="$autoplay" \
            SLIPPIOS_SKIP_SPLASH="$splash" \
            SDL_JOYSTICK_HIDAPI_GAMECUBE=1 \
            SDL_HIDAPI_LIBUSB_GAMECUBE=1 \
              "$shell_bin" "''${shell_args[@]}" 2>>"$log" || code=$?

            # Godot must exit before Dolphin opens the adapter.
            if [ "$code" -eq 42 ]; then
              "$cli_bin" play >>"$log" 2>&1 || true
              autoplay=0
              splash=1
              continue
            fi
            if [ "$code" -eq 43 ]; then
              "$cli_bin" dolphin config >>"$log" 2>&1 || true
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
            if [ "$code" -eq 46 ]; then
              autoplay=0
              splash=1
              continue
            fi
            if [ "$code" -eq 47 ]; then
              "$cli_bin" replay play "$(cat /data/system/replay-request 2>/dev/null)" >>"$log" 2>&1 || true
              autoplay=0
              splash=1
              continue
            fi

            echo "slippios-shell exited with $code" >>"$log"
            exit "$code"
          done
        '';
      };
    };
}
