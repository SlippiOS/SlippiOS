{ lib, pkgs, ... }:

let
  # Temporary branding: the Slippi mark recoloured to Slippi green. Reused by
  # the Godot splash, the Plymouth theme, and the X root window so the handoff
  # between them has no blank frame.
  splashMark = ../../apps/shell/assets/slippi-mark.png;
  theme = import ../../theme/default.nix;
  plymouthConf = ''
    [Daemon]
    ShowDelay=0
    DeviceTimeout=1
    Theme=slippios
  '';
  themeBackground = pkgs.runCommand "slippios-theme-background.png" { } ''
    ${pkgs.imagemagick}/bin/magick -size 1920x1080 xc:'${theme.background}' "$out"
  '';
  themeJson = pkgs.writeText "slippios-theme.json" (builtins.toJSON {
    inherit (theme) background surface accent text;
    splash = {
      mark_fraction = theme.splash.markFraction;
      pulse_speed = theme.splash.pulseSpeed;
      pulse_min = theme.splash.pulseMin;
      pulse_max = theme.splash.pulseMax;
      scale_pulse = theme.splash.scalePulse;
    };
  });

  # Charge-style two-step theme: the mark is pre-rendered as throbber frames
  # with a sine opacity, the same C-side animation Fedora's charge theme uses.
  slippiosPlymouth = pkgs.runCommand "slippios-plymouth" { } ''
    theme=$out/share/plymouth/themes/slippios
    mkdir -p "$theme"
    # The two-step plugin loads these dialog assets during show_splash_screen,
    # even in boot mode; without them it aborts and Plymouth falls back to text.
    cp ${pkgs.plymouth}/share/plymouth/themes/spinner/{lock,entry,keyboard,capslock,keymap-render,bullet}.png "$theme"/
    for frame in $(seq 1 ${toString theme.splash.plymouthFrames}); do
      alpha=$(awk -v i="$frame" 'BEGIN { pi = atan2(0, -1); print ${toString theme.splash.pulseMin} + (${toString theme.splash.pulseMax} - ${toString theme.splash.pulseMin}) * (0.5 + 0.5 * sin(2 * pi * i / ${toString theme.splash.plymouthFrames})) }')
      ${pkgs.imagemagick}/bin/magick ${splashMark} -resize x${toString theme.splash.plymouthMarkHeight} \
        -channel A -evaluate multiply "$alpha" +channel PNG32:"$theme/throbber-$(printf %04d "$frame").png"
    done
    cat > "$theme/slippios.plymouth" <<EOF
    [Plymouth Theme]
    Name=SlippiOS
    Description=SlippiOS boot splash
    ModuleName=two-step

    [two-step]
    Font=DejaVu Sans 12
    TitleFont=DejaVu Sans 30
    ImageDir=$theme
    HorizontalAlignment=.5
    VerticalAlignment=.5
    BackgroundStartColor=0x${lib.removePrefix "#" theme.background}
    BackgroundEndColor=0x${lib.removePrefix "#" theme.background}
    Transition=none
    TransitionDuration=0.0
    ProgressBarBackgroundColor=0x${lib.removePrefix "#" theme.surface}
    ProgressBarForegroundColor=0x${lib.removePrefix "#" theme.accent}
    MessageBelowAnimation=true

    [boot-up]
    UseEndAnimation=false

    [shutdown]
    UseEndAnimation=false

    [reboot]
    UseEndAnimation=false
    EOF
  '';
in {
  # The console is covered by the boot splash; keep systemd's service status
  # lines off it. Failures remain in the journal and the recovery console.
  boot.plymouth = {
    enable = true;
    theme = "slippios";
    themePackages = [ slippiosPlymouth ];
    logo = splashMark;
  };

  # NixOS hardcodes DeviceTimeout=8 before extraConfig; duplicate keys keep the
  # first value, so replace the generated config outright. Without this the
  # console sits on screen until the 8s device wait expires.
  boot.initrd.systemd.contents."/etc/plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };
  environment.etc."plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };

  # LightDM's greeter background, used during autologin before the session
  # paints the root window. Same value as the Plymouth/shell background.
  services.xserver.displayManager.lightdm.background = themeBackground;

  # Runtime copy of theme/default.nix for the Godot shell.
  environment.etc."slippios/theme.json".source = themeJson;
}
