{ pkgs, lib, ... }:

# Development-only additions. Never import this from the release host.
{
  # The diagnostics and firmware need more room than the release store.
  # Keep the update slot the same size, otherwise a dev-to-dev update cannot
  # fit into the inactive partition.
  image.repart.partitions.nix-store.repartConfig = {
    SizeMinBytes = lib.mkForce "2500M";
    SizeMaxBytes = lib.mkForce "2500M";
  };
  image.repart.partitions.empty.repartConfig = {
    SizeMinBytes = lib.mkForce "2500M";
    SizeMaxBytes = lib.mkForce "2500M";
  };

  services.openssh = {
    enable = true;
    settings = {
      PasswordAuthentication = true;
      PermitRootLogin = "no";
    };
  };

  users.users.player.initialPassword = "slippios";

  networking.firewall.allowedTCPPorts = [ 22 ];

  environment.systemPackages = with pkgs; [
    htop
    mesa-demos
    pciutils
    usbutils
    vulkan-tools
    (writeShellScriptBin "slippios-diag" ''
      #!/bin/sh
      export DISPLAY="''${DISPLAY:-:0}"
      export XAUTHORITY="''${XAUTHORITY:-/home/player/.Xauthority}"
      echo "== system =="
      uname -a
      echo "== gpu =="
      lspci -nnk | grep -A3 -iE 'vga|3d|display' || true
      echo "== kernel gpu =="
      sudo -n dmesg 2>/dev/null | grep -iE 'amdgpu|radeon|firmware|drm|gpu' | tail -80 || true
      echo "== xrandr =="
      xrandr --current 2>&1 || true
      echo "== glxinfo =="
      glxinfo -B 2>&1 | head -40 || true
      echo "== vulkan =="
      vulkaninfo --summary 2>&1 | head -40 || true
      echo "== top =="
      top -b -n1 | head -30 || true
      echo "== failed units =="
      systemctl --failed --no-pager 2>&1 || true
      echo "== display-manager =="
      sudo -n journalctl -b -u display-manager --no-pager 2>&1 | tail -80 || true
      echo "== shell log =="
      tail -80 /data/system/shell.log 2>/dev/null || true
      echo "== dolphin log =="
      tail -80 /data/users/default/logs/dolphin.log 2>/dev/null || true
    '')
  ];

  boot.initrd.kernelModules = [ "virtio_gpu" "bochs" ];

  security.sudo.wheelNeedsPassword = false;
  users.users.player.extraGroups = [ "wheel" ];

  security.sudo = {
    enable = true;
    extraRules = [{
      users = [ "player" ];
      commands = [{
        command = "/run/current-system/sw/bin/systemctl restart display-manager";
        options = [ "NOPASSWD" ];
      }];
    }];
  };

  systemd.services.slippios-dev-dir = {
    description = "Create the dev push directory.";
    wantedBy = [ "multi-user.target" ];
    after = [ "slippi-data.service" ];
    requires = [ "slippi-data.service" ];
    serviceConfig.Type = "oneshot";
    script = ''
      install -d -o player -g users -m 0775 /data/dev
    '';
  };
}
