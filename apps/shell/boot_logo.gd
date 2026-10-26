extends Control

## Boot mark. Uses the shared Slippi mark (temporary branding) and pulses it
## with a sine wave while the shell loads.

const Mark := preload("res://assets/slippi-mark.png")

var _time := 0.0


func _ready() -> void:
	# Size the mark from the display, not a fixed pixel size: the fraction
	# comes from theme/default.nix and keeps it consistent from 720p to 4K.
	var screen := get_viewport_rect().size
	var height := screen.y * SlippiTheme.mark_fraction
	var width := height * (float(Mark.get_width()) / float(Mark.get_height()))
	custom_minimum_size = Vector2(width, height)
	size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	set_process(true)


func _process(delta: float) -> void:
	_time += delta
	queue_redraw()


func _draw() -> void:
	if size.x <= 1 or size.y <= 1:
		return
	var center := size * 0.5
	var pulse := SlippiTheme.pulse_min + (SlippiTheme.pulse_max - SlippiTheme.pulse_min) \
		* (0.5 + 0.5 * sin(_time * SlippiTheme.pulse_speed))
	var fit := (minf(size.x / Mark.get_width(), size.y / Mark.get_height())
		* (0.92 + SlippiTheme.scale_pulse * sin(_time * SlippiTheme.pulse_speed)))
	var target := Mark.get_size() * fit
	var rect := Rect2(center - target * 0.5, target)

	# Soft glow so the pulse reads on a dark television.
	for ring in range(3):
		var radius := target.length() * (0.26 + ring * 0.05)
		draw_circle(center, radius, Color(0.267, 0.663, 0.388, 0.045 * pulse))
	draw_texture_rect(Mark, rect, false, Color(1, 1, 1, pulse))
