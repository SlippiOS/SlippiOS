extends Control

## Small procedural boot mark. It needs no external asset or license, and can
## be replaced by a video/sprite later without changing the shell flow.

var _time := 0.0


func _ready() -> void:
	custom_minimum_size = Vector2(190, 190)
	size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	set_process(true)


func _process(delta: float) -> void:
	_time += delta
	queue_redraw()


func _draw() -> void:
	if size.x <= 1 or size.y <= 1:
		return
	var center := size * 0.5
	var radius := minf(size.x, size.y) * 0.29
	var breathe := sin(_time * 2.1) * 3.0

	for ring in range(3):
		var glow_radius := radius + 13.0 + ring * 11.0 + breathe
		var alpha := 0.10 - ring * 0.025
		draw_arc(center, glow_radius, 0.0, TAU, 72,
			Color(0.30, 0.55, 1.0, alpha), 2.0, true)

	draw_arc(center, radius, -0.35 + sin(_time * 0.9) * 0.12, 2.15,
		72, Color(0.30, 0.56, 1.0, 0.95), 3.0, true)
	draw_arc(center, radius, 2.35, TAU - 0.15,
		72, Color(0.55, 0.82, 1.0, 0.45), 2.0, true)

	for dot in range(4):
		var angle := _time * 1.15 + float(dot) * TAU / 4.0
		var position := center + Vector2(cos(angle), sin(angle)) * radius
		draw_circle(position, 4.5, Color(0.72, 0.90, 1.0, 0.9))
		draw_circle(position, 9.0, Color(0.30, 0.55, 1.0, 0.12))

	var diamond := radius * 0.48
	var points := PackedVector2Array([
		center + Vector2(0.0, -diamond),
		center + Vector2(diamond, 0.0),
		center + Vector2(0.0, diamond),
		center + Vector2(-diamond, 0.0),
	])
	draw_colored_polygon(points, Color(0.20, 0.43, 0.88, 0.85))
	draw_polyline(PackedVector2Array([
		points[0], points[1], points[2], points[3], points[0],
	]), Color(0.68, 0.86, 1.0, 0.8), 1.5, true)
