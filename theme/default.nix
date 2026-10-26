# Single source of truth for SlippiOS branding. Nix consumes these values for
# Plymouth, LightDM and the X session; the image also installs them as
# /etc/slippios/theme.json, which the Godot shell reads at startup.
{
  background = "#0b0b10";
  # Godot's engine boot-splash clear color is read before scripts run, so it
  # must be baked into project.godot at build time.
  backgroundGodot = "Color(0.043137, 0.043137, 0.062745, 1)";
  surface = "#202633";
  accent = "#44a963";
  text = "#eef1f7";

  splash = {
    # Godot mark height as a fraction of the viewport.
    markFraction = 0.30;
    pulseSpeed = 5.2;
    pulseMin = 0.35;
    pulseMax = 1.0;
    scalePulse = 0.015;

    # Plymouth renders fixed-size frames; two-step cannot scale per display.
    plymouthMarkHeight = 324;
    plymouthFrames = 24;
  };
}
