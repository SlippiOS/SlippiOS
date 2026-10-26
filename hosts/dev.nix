{ pkgs, ... }:

# Development-only additions. Never import this from the release host.
# SSH is password-protected with a throwaway password: this image is for local
# VMs, not for flashing or for a network you do not control.
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

  # Let the push script restart the graphical session without a password.
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

  # Pushed builds live here and take precedence in the shell wrapper.
  systemd.services.slippios-dev-dir = {
    description = "Create the dev push directory";
    wantedBy = [ "multi-user.target" ];
    after = [ "slippi-data.service" ];
    requires = [ "slippi-data.service" ];
    serviceConfig.Type = "oneshot";
    script = "install -d -o player -g users -m 0775 /data/dev";
  };
}
