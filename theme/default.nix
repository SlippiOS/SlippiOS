{
  background = "#0b0b10";
  backgroundGodot = "Color(0.043137, 0.043137, 0.062745, 1)";
  surface = "#202633";
  accent = "#44a963";
  text = "#eef1f7";
  brand = "#4aa8ff";
  progress = "#4aa8ff";
  danger = "#e06c75";
  logoInk = "#f7f8fa";
  logoInkLight = "#0b0b10";

  splash = {
    markFraction = 0.30;
    # The splash mark is drawn at markFraction * markFit of the screen height.
    # The wallpaper generator uses the same value so the handoff does not jump.
    markFit = 0.92;
    pulseSpeed = 5.2;
    pulseMin = 0.35;
    pulseMax = 1.0;
    scalePulse = 0.015;

    plymouthMarkHeight = 324;
    plymouthFrames = 24;
    # 24 frames = 3 multishine pulses of 8 frames each.
    plymouthPulseFrames = 8;
  };
}
