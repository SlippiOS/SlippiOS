extends RefCounted


var app
var categories: Array
var on_select: Callable
var on_back: Callable
var selected := 0
var pane: VBoxContainer

var _rail: VBoxContainer
var _pane: VBoxContainer


func _init(shell, names: Array, select_callback: Callable,
		back_callback: Callable) -> void:
	app = shell
	categories = names
	on_select = select_callback
	on_back = back_callback


func build(initial: int = 0) -> void:
	var columns := HBoxContainer.new()
	columns.add_theme_constant_override("separation", 28)
	app.add_content(columns)

	_rail = VBoxContainer.new()
	_rail.custom_minimum_size.x = 210
	_rail.add_theme_constant_override("separation", 4)
	columns.add_child(_rail)

	_pane = VBoxContainer.new()
	_pane.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_pane.add_theme_constant_override("separation", 10)
	columns.add_child(_pane)
	pane = _pane

	for index in categories.size():
		_rail.add_child(_category_button(str(categories[index]), index))

	var spacer := Control.new()
	spacer.custom_minimum_size = Vector2(0, 10)
	spacer.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_rail.add_child(spacer)
	var back_button := Button.new()
	back_button.text = "Back"
	back_button.alignment = HORIZONTAL_ALIGNMENT_LEFT
	back_button.custom_minimum_size = Vector2(0, 44)
	back_button.add_theme_font_size_override("font_size", 16)
	back_button.add_theme_stylebox_override("focus", _focus_style())
	back_button.pressed.connect(on_back)
	_rail.add_child(back_button)
	_paint_category(back_button, false)

	select(initial)


func select(index: int) -> void:
	selected = index
	for position in categories.size():
		_paint_category(_rail.get_child(position) as Button, position == index)
	for child in _pane.get_children():
		_pane.remove_child(child)
		child.queue_free()

	on_select.call(index, self)
	_wire_focus(index)


func focus_selected() -> void:
	app.focus_or(_rail.get_child(selected) as Control)


func pane_title(text: String) -> void:
	var node := Label.new()
	node.text = text
	if SlippiTheme.font_bold != null:
		node.add_theme_font_override("font", SlippiTheme.font_bold)
	node.add_theme_font_size_override("font_size", 24)
	node.add_theme_color_override("font_color", SlippiTheme.text)
	_pane.add_child(node)
	var spacer := Control.new()
	spacer.custom_minimum_size = Vector2(0, 4)
	spacer.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_pane.add_child(spacer)


func pane_button(text: String, callback: Callable) -> void:
	var node := Button.new()
	node.text = text
	node.custom_minimum_size = Vector2(0, 46)
	node.pressed.connect(callback)
	_pane.add_child(node)


func pane_hint(text: String) -> void:
	var node := Label.new()
	node.text = text
	node.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	node.add_theme_color_override("font_color", Color(SlippiTheme.text, 0.55))
	node.add_theme_font_size_override("font_size", 14)
	_pane.add_child(node)


func _category_button(text: String, index: int) -> Button:
	var node := Button.new()
	node.text = text
	node.alignment = HORIZONTAL_ALIGNMENT_LEFT
	node.custom_minimum_size = Vector2(0, 44)
	node.add_theme_font_size_override("font_size", 16)
	node.add_theme_stylebox_override("focus", _focus_style())
	node.focus_entered.connect(select.bind(index))
	node.pressed.connect(select.bind(index))
	return node


func _wire_focus(index: int) -> void:
	var active: Button = _rail.get_child(index) as Button
	var pane_buttons: Array[Button] = []
	for child in _pane.get_children():
		if child is Button:
			pane_buttons.append(child)

	for position in pane_buttons.size():
		var button := pane_buttons[position]
		var up := pane_buttons[position - 1] if position > 0 else button
		var down := pane_buttons[position + 1] \
			if position + 1 < pane_buttons.size() else button
		button.focus_neighbor_left = button.get_path_to(active)
		button.focus_neighbor_right = button.get_path_to(button)
		button.focus_neighbor_top = button.get_path_to(up)
		button.focus_neighbor_bottom = button.get_path_to(down)

	var first: Button = pane_buttons[0] if not pane_buttons.is_empty() else null
	var rail_buttons: Array[Button] = []
	for position in categories.size():
		rail_buttons.append(_rail.get_child(position) as Button)
	var back_button: Button = _rail.get_child(categories.size() + 1) as Button
	rail_buttons.append(back_button)
	for position in rail_buttons.size():
		var button := rail_buttons[position]
		var up := rail_buttons[position - 1] if position > 0 else button
		var down := rail_buttons[position + 1] \
			if position + 1 < rail_buttons.size() else button
		button.focus_neighbor_left = button.get_path_to(button)
		button.focus_neighbor_right = button.get_path_to(first) \
			if first != null else button.get_path_to(button)
		button.focus_neighbor_top = button.get_path_to(up)
		button.focus_neighbor_bottom = button.get_path_to(down)


func _paint_category(node: Button, active: bool) -> void:
	var color := SlippiTheme.text if active else Color(SlippiTheme.text, 0.65)
	node.add_theme_color_override("font_color", color)
	node.add_theme_color_override("font_hover_color", SlippiTheme.text)
	node.add_theme_color_override("font_pressed_color", SlippiTheme.text)
	node.add_theme_color_override("font_focus_color", SlippiTheme.text)
	node.add_theme_stylebox_override("normal", _rail_style(active, false))
	node.add_theme_stylebox_override("hover", _rail_style(active, true))
	node.add_theme_stylebox_override("pressed", _rail_style(active, true))


func _rail_style(active: bool, hover: bool) -> StyleBoxFlat:
	var box := StyleBoxFlat.new()
	box.bg_color = Color(0, 0, 0, 0)
	if active:
		box.bg_color = SlippiTheme.surface
	elif hover:
		box.bg_color = Color(SlippiTheme.surface, 0.45)
	box.corner_radius_top_left = 6
	box.corner_radius_top_right = 6
	box.corner_radius_bottom_left = 6
	box.corner_radius_bottom_right = 6
	box.content_margin_left = 14
	box.content_margin_right = 12
	box.content_margin_top = 9
	box.content_margin_bottom = 9
	return box


func _focus_style() -> StyleBoxFlat:
	var box := StyleBoxFlat.new()
	box.bg_color = Color(0, 0, 0, 0)
	box.border_width_left = 1
	box.border_width_top = 1
	box.border_width_right = 1
	box.border_width_bottom = 1
	box.border_color = Color(SlippiTheme.text, 0.6)
	box.corner_radius_top_left = 6
	box.corner_radius_top_right = 6
	box.corner_radius_bottom_left = 6
	box.corner_radius_bottom_right = 6
	return box
