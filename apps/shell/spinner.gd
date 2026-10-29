extends Control


var _angle := 0.0


func _ready() -> void:
	custom_minimum_size = Vector2(48, 48)
	size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	set_process(true)


func _process(delta: float) -> void:
	_angle += delta * 4.5
	queue_redraw()


func _draw() -> void:
	var center := size * 0.5
	var radius := minf(size.x, size.y) * 0.32
	for index in range(8):
		var angle := _angle + index * TAU / 8.0
		var alpha := 0.15 + 0.85 * (float(index) / 8.0)
		draw_circle(center + Vector2(cos(angle), sin(angle)) * radius,
			3.0, Color(1.0, 1.0, 1.0, alpha))
