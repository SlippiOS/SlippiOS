{ lib, pkgs, slippiosPackages, ... }:

let
  amdgpuFirmware = pkgs.runCommand "amdgpu-firmware" {
    meta.license = lib.licenses.unfreeRedistributableFirmware;
  } ''
    mkdir -p $out/lib/firmware
    cp -r ${pkgs.linux-firmware}/lib/firmware/amdgpu $out/lib/firmware/
  '';

  licenseBundle = pkgs.runCommand "slippios-licenses" { } ''
    mkdir -p $out
    cp ${../../LICENSE} $out/LICENSE
    cp ${../../NOTICE} $out/NOTICE
    cp ${../../THIRD_PARTY_LICENSES.md} $out/THIRD_PARTY_LICENSES.md
    cp ${../../SOURCE} $out/SOURCE
    cp ${../../README.md} $out/README.md
    cp -r ${../../licenses}/. $out/
  '';
in
{
  nix.settings.experimental-features = [ "nix-command" "flakes" ];

  hardware.firmware = [ amdgpuFirmware ];

  boot.kernelParams = [
    "quiet" "loglevel=3" "vt.global_cursor_default=0" "systemd.show_status=0"
    # Avoid watchdog wakeups adding jitter to frame pacing.
    "nowatchdog"
  ];
  # Keep in sync with loglevel; 0 hides late text between Plymouth and the shell.
  boot.consoleLogLevel = 0;
  # Disable status lines explicitly; the initrd manager ignores show_status.
  boot.initrd.systemd.settings.Manager.ShowStatus = "no";
  systemd.settings.Manager.ShowStatus = "no";
  boot.kernelModules = [ "uinput" ];

  services.udev.extraRules = ''
    SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="057e", ATTRS{idProduct}=="0337", MODE="0666", TAG+="uaccess"
    SUBSYSTEM=="usb", ENV{DEVTYPE}=="usb_device", ATTRS{idVendor}=="057e", ATTRS{idProduct}=="0337", ATTR{power/control}="on"
  '';

  networking.networkmanager.enable = true;
  networking.firewall.allowedTCPPorts = [ 8080 ];
  time.timeZone = "UTC";

  # PipeWire otherwise restores a per-application stream volume (Dolphin can
  # come back at -20 dB). The console has no volume UI, so keep streams at 0 dB.
  security.rtkit.enable = true;
  systemd.user.services.slippios-audio = {
    description = "Keep SlippiOS audio streams at full volume";
    wantedBy = [ "default.target" ];
    after = [ "wireplumber.service" ];
    serviceConfig = {
      Type = "simple";
      Restart = "always";
      RestartSec = 2;
      ExecStart = "${pkgs.writeShellScript "slippios-audio" ''
        #!/bin/sh
        ${pkgs.pulseaudio}/bin/pactl set-sink-volume @DEFAULT_SINK@ 100% || true
        ${pkgs.pulseaudio}/bin/pactl subscribe | while read -r event; do
          case "$event" in
            *"new' on sink-input"*)
              ${pkgs.pulseaudio}/bin/pactl list short sink-inputs |
                while read -r id _; do
                  ${pkgs.pulseaudio}/bin/pactl set-sink-input-volume "$id" 100% || true
                done
              ;;
            *"new' on sink"*)
              ${pkgs.pulseaudio}/bin/pactl set-sink-volume @DEFAULT_SINK@ 100% || true
              ;;
          esac
        done
      ''}";
    };
  };

  users.users.player = {
    isNormalUser = true;
    uid = 1000;
    extraGroups = [ "audio" "input" "video" "networkmanager" ];
  };

  environment.systemPackages = with pkgs; [
    bash
    coreutils
    openbox
    pulseaudio
    qrencode
    networkmanager
    slippiosPackages.appimage-runner
    slippiosPackages.session
    slippiosPackages.cli
    slippiosPackages.shell
    nano
    xterm
    libxcvt
    xdotool
    xrandr
    xset
  ];

  environment.etc."slippios/README".text = ''
    Settings live in /data/users/default/settings.toml.
    Game images live in /data/isos/.
  '';
  environment.etc."slippios/licenses".source = licenseBundle;

  systemd.tmpfiles.rules = [
    "d /home/player 0755 player users -"
    "d /home/player/.config 0755 player users -"
  ];

  system.stateVersion = "25.05";
}
