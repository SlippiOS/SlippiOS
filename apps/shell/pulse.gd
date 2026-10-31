extends Control


const Shine := preload("res://assets/slippi-mark-shine.png")

var _time := 0.0


func _ready() -> void:
	custom_minimum_size = Vector2(64, 64)
	size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
	set_process(true)


func _process(delta: float) -> void:
	_time += delta
	queue_redraw()


func _draw() -> void:
	if size.x <= 1 or size.y <= 1:
		return
	var center := size * 0.5
	var wave := 0.5 + 0.5 * sin(_time * SlippiTheme.pulse_speed)
	var pulse := SlippiTheme.pulse_min + (SlippiTheme.pulse_max - SlippiTheme.pulse_min) * wave
	var fit := minf(size.x / Shine.get_width(), size.y / Shine.get_height()) \
		* SlippiTheme.mark_fit
	var target := Shine.get_size() * fit
	var scale := 1.0 + SlippiTheme.scale_pulse * wave
	var rect := Rect2(center - target * scale * 0.5, target * scale)
	draw_texture_rect(Shine, rect, false, Color(1, 1, 1, pulse))
