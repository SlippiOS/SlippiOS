{ lib, pkgs, ... }:

let
  theme = import ../../theme/default.nix;
  logo = ../../assets/marks/icon.svg;
  splitLayers = ../../assets/marks/split-layers.py;
  markHeight = theme.splash.plymouthMarkHeight;

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
    inherit (theme) background surface accent text brand progress danger;
    splash = {
      mark_fraction = theme.splash.markFraction;
      mark_fit = theme.splash.markFit;
      pulse_speed = theme.splash.pulseSpeed;
      pulse_min = theme.splash.pulseMin;
      pulse_max = theme.splash.pulseMax;
      scale_pulse = theme.splash.scalePulse;
    };
  });

  # The two-step theme shows the throbber animation; a transparent logo
  # prevents a static copy of the shine from sitting behind the pulse.
  transparentLogo = pkgs.runCommand "slippios-plymouth-logo" { } ''
    ${pkgs.imagemagick}/bin/magick -size 1x1 xc:none PNG32:$out
  '';

  slippiosPlymouth = pkgs.runCommand "slippios-plymouth" {
    nativeBuildInputs = [ pkgs.librsvg pkgs.imagemagick pkgs.python3 ];
  } ''
    theme=$out/share/plymouth/themes/slippios
    mkdir -p "$theme"

    # Two-step aborts without these dialog assets, even in boot mode.
    cp ${pkgs.plymouth}/share/plymouth/themes/spinner/{lock,entry,keyboard,capslock,keymap-render,bullet}.png "$theme"/

    # Split the full mark into the pulsing shine and the static S.
    python3 ${splitLayers} "$TMPDIR/shine.svg" "$TMPDIR/s.svg" ${logo}

    rsvg-convert -w ${toString markHeight} -h ${toString markHeight} \
      -o "$TMPDIR/shine.png" "$TMPDIR/shine.svg"
    rsvg-convert -w ${toString markHeight} -h ${toString markHeight} \
      -o "$TMPDIR/s.png" "$TMPDIR/s.svg"

    frames=${toString theme.splash.plymouthFrames}
    pulse_frames=${toString theme.splash.plymouthPulseFrames}
    for frame in $(seq 0 $((frames - 1))); do
      phase=$((frame % pulse_frames))
      read -r pulse opacity scale flash glow < <(awk -v p="$phase" -v n="$pulse_frames" 'BEGIN {
        pi = atan2(0, -1);
        pulse = sin(pi * p / (n - 1)) ^ 3;
        printf "%.6f %.6f %.6f %.6f %.6f\n",
          pulse, 0.22 + 0.78 * pulse, 1.0 + 0.04 * pulse,
          0.70 * pulse, 0.50 * pulse;
      }')
      scale_pct=$(awk -v s="$scale" 'BEGIN { printf "%.2f", s * 100 }')

      magick "$TMPDIR/shine.png" -resize "$scale_pct%" \
        -channel A -evaluate multiply "$opacity" +channel \
        "$TMPDIR/shine-frame.png"
      magick "$TMPDIR/shine.png" -resize "$scale_pct%" \
        -fill white -colorize 100 -channel A -evaluate multiply "$flash" +channel \
        "$TMPDIR/flash.png"
      magick "$TMPDIR/shine.png" -resize "$scale_pct%" \
        -blur 0x9 -channel A -evaluate multiply "$glow" +channel \
        "$TMPDIR/glow.png"

      magick -size ${toString markHeight}x${toString markHeight} xc:none \
        "$TMPDIR/glow.png" -gravity center -composite \
        "$TMPDIR/flash.png" -gravity center -composite \
        "$TMPDIR/shine-frame.png" -gravity center -composite \
        "$TMPDIR/s.png" -gravity center -composite \
        PNG32:"$theme/throbber-$(printf %04d $((frame + 1))).png"
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
    ProgressBarForegroundColor=0x${lib.removePrefix "#" theme.brand}
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
  boot.plymouth = {
    enable = true;
    theme = "slippios";
    themePackages = [ slippiosPlymouth ];
    logo = transparentLogo;
  };

  # NixOS hardcodes DeviceTimeout=8 and duplicate keys keep the first value,
  # so replace the generated config outright.
  boot.initrd.systemd.contents."/etc/plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };
  environment.etc."plymouth/plymouthd.conf" =
    lib.mkForce { text = plymouthConf; };

  services.xserver.displayManager.lightdm.background = themeBackground;

  environment.etc."slippios/theme.json".source = themeJson;
}
