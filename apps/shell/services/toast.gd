extends RefCounted


const MAX_VISIBLE := 3
const WIDTH := 420.0
const MARGIN := 24.0

var layer: VBoxContainer


func attach(parent: Control) -> void:
	layer = VBoxContainer.new()
	layer.name = "Toasts"
	layer.mouse_filter = Control.MOUSE_FILTER_IGNORE
	layer.set_anchors_preset(Control.PRESET_TOP_RIGHT)
	layer.offset_left = -WIDTH - MARGIN
	layer.offset_top = MARGIN
	layer.offset_right = -MARGIN
	layer.offset_bottom = MARGIN
	layer.grow_horizontal = Control.GROW_DIRECTION_BEGIN
	layer.grow_vertical = Control.GROW_DIRECTION_END
	layer.add_theme_constant_override("separation", 8)
	parent.add_child(layer)


func show(message: String, kind: String = "info") -> void:
	if layer == null or not is_instance_valid(layer):
		return
	var text := message.strip_edges()
	if text == "":
		return
	while layer.get_child_count() >= MAX_VISIBLE:
		var oldest := layer.get_child(0)
		layer.remove_child(oldest)
		oldest.queue_free()

	var panel := PanelContainer.new()
	panel.mouse_filter = Control.MOUSE_FILTER_IGNORE
	panel.custom_minimum_size.x = WIDTH
	panel.add_theme_stylebox_override("panel", _panel_style(kind))

	var label := Label.new()
	label.text = text
	label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	label.add_theme_color_override("font_color", SlippiTheme.text)
	label.add_theme_font_size_override("font_size", 16)
	panel.add_child(label)

	panel.modulate.a = 0.0
	layer.add_child(panel)

	var hold := clampf(2.6 + text.length() * 0.035, 3.0, 6.0)
	var tween := panel.create_tween()
	tween.tween_property(panel, "modulate:a", 1.0, 0.16)
	tween.tween_interval(hold)
	tween.tween_property(panel, "modulate:a", 0.0, 0.3)
	tween.tween_callback(func() -> void:
		if is_instance_valid(panel):
			panel.queue_free()
	)


func _panel_style(kind: String) -> StyleBoxFlat:
	var box := StyleBoxFlat.new()
	box.bg_color = SlippiTheme.surface
	box.border_width_left = 3
	box.border_color = _accent(kind)
	box.corner_radius_top_left = 6
	box.corner_radius_top_right = 6
	box.corner_radius_bottom_left = 6
	box.corner_radius_bottom_right = 6
	box.content_margin_left = 16
	box.content_margin_right = 16
	box.content_margin_top = 11
	box.content_margin_bottom = 11
	box.shadow_color = Color(0, 0, 0, 0.35)
	box.shadow_size = 8
	box.shadow_offset = Vector2(0, 2)
	return box


func _accent(kind: String) -> Color:
	match kind:
		"success":
			return SlippiTheme.accent
		"error":
			return SlippiTheme.danger
		_:
			return SlippiTheme.brand
