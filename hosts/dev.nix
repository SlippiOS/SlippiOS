{ pkgs, ... }:

# Development-only additions. Never import this from the release host.
{
  services.openssh = {
    enable = true;
    settings = {
      PasswordAuthentication = true;
      PermitRootLogin = "no";
    };
  };

  users.users.player.initialPassword = "slippios";

  networking.firewall.allowedTCPPorts = [ 22 ];

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
