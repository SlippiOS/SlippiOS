extends Control


const Shine := preload("res://assets/slippi-mark-shine.png")
const S := preload("res://assets/slippi-mark-s.png")

var _time := 0.0


func _ready() -> void:
	# Scale from the display so the mark is consistent from 720p to 4K.
	size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	# The mark is downscaled heavily on 720p/1080p displays; mipmaps keep the
	# thin S stroke and hexagon edges smooth instead of stair-stepped.
	texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
	_resize()
	get_viewport().size_changed.connect(_resize)
	set_process(true)


func _resize() -> void:
	var screen := get_viewport_rect().size
	var height := screen.y * SlippiTheme.mark_fraction
	var width := height * (float(Shine.get_width()) / float(Shine.get_height()))
	custom_minimum_size = Vector2(width, height)


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
	var rect := Rect2(center - target * 0.5, target)

	# Only the shine pulses. The S stays fixed and fully opaque.
	var shine_scale := 1.0 + SlippiTheme.scale_pulse * wave
	var shine_rect := Rect2(center - target * shine_scale * 0.5, target * shine_scale)
	draw_texture_rect(Shine, shine_rect, false, Color(1, 1, 1, pulse))
	draw_texture_rect(S, rect, false, Color(1, 1, 1, 1))
