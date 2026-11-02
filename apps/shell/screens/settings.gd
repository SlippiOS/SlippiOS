extends "res://screens/screen.gd"


const CATEGORIES := ["Display & Audio", "Gameplay", "Network", "System", "Advanced"]

static var selected := 0

var _rail: VBoxContainer
var _pane: VBoxContainer
var _settings: Dictionary = {}


func present() -> void:
	clear()
	title("SETTINGS", 44)
	_settings = cli(["settings", "show"]).get("settings", {})

	var columns := HBoxContainer.new()
	columns.add_theme_constant_override("separation", 28)
	add_content(columns)

	_rail = VBoxContainer.new()
	_rail.custom_minimum_size.x = 210
	_rail.add_theme_constant_override("separation", 4)
	columns.add_child(_rail)

	_pane = VBoxContainer.new()
	_pane.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	_pane.add_theme_constant_override("separation", 10)
	columns.add_child(_pane)

	for index in CATEGORIES.size():
		_rail.add_child(_category_button(CATEGORIES[index], index))

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
	back_button.pressed.connect(app.show_menu)
	_rail.add_child(back_button)
	_paint_category(back_button, false)

	_select(selected)
	focus_or(_rail.get_child(selected) as Control)


func _category_button(text: String, index: int) -> Button:
	var node := Button.new()
	node.text = text
	node.alignment = HORIZONTAL_ALIGNMENT_LEFT
	node.custom_minimum_size = Vector2(0, 44)
	node.add_theme_font_size_override("font_size", 16)
	node.add_theme_stylebox_override("focus", _focus_style())
	node.focus_entered.connect(_select.bind(index))
	node.pressed.connect(_select.bind(index))
	return node


func _select(index: int) -> void:
	selected = index
	for position in CATEGORIES.size():
		_paint_category(_rail.get_child(position) as Button, position == index)
	for child in _pane.get_children():
		_pane.remove_child(child)
		child.queue_free()

	match index:
		0:
			_display_audio()
		1:
			_gameplay()
		2:
			_network()
		3:
			_system()
		4:
			_advanced()

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
	for position in CATEGORIES.size():
		rail_buttons.append(_rail.get_child(position) as Button)
	var back_button: Button = _rail.get_child(CATEGORIES.size() + 1) as Button
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


func _display_audio() -> void:
	var display: Dictionary = _settings.get("display", {})
	var audio: Dictionary = _settings.get("audio", {})
	_pane_title("Display & Audio")
	_pane_button("Display: %s" % _display_label(display), app.show_display)
	_pane_button("Audio: %d%%" % int(audio.get("volume", 100)), app.show_audio)
	_pane_hint("Changes apply immediately and are saved for the next boot.")


func _gameplay() -> void:
	_pane_title("Gameplay")
	_pane_button("Dolphin Settings", app.show_dolphin)
	_pane_hint("Graphics, audio, and the stable or beta build.")
	_pane_button("Controller Settings", app.show_controllers)
	_pane_hint("Assign each GameCube port to an adapter, a pad, or the keyboard.")


func _network() -> void:
	var network: Dictionary = _settings.get("network", {})
	_pane_title("Network")
	_pane_button("Wi-Fi: %s" % text_value(network.get("ssid"), "not connected"), app.show_wifi)
	_pane_hint("Ethernet works automatically when a cable is connected.")


func _system() -> void:
	_pane_title("System")
	_pane_button("Update", app.show_update)
	_pane_hint("Install system updates and check the bundled Dolphin build.")


func _advanced() -> void:
	var expert: Dictionary = _settings.get("expert", {})
	var expert_on := bool(expert.get("enabled", false))
	_pane_title("Advanced")
	_pane_button("Expert mode: %s" % ("on" if expert_on else "off"), _toggle_expert)
	_pane_hint("Unlocks the terminal, text editor, and Dolphin GUI.")
	if expert_on:
		_pane_button("Run setup again", _run_setup)
		_pane_button("Expert tools", app.show_expert)


func _pane_title(text: String) -> void:
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


func _pane_button(text: String, callback: Callable) -> void:
	var node := Button.new()
	node.text = text
	node.custom_minimum_size = Vector2(0, 46)
	node.pressed.connect(callback)
	_pane.add_child(node)


func _pane_hint(text: String) -> void:
	var node := Label.new()
	node.text = text
	node.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	node.add_theme_color_override("font_color", Color(SlippiTheme.text, 0.55))
	node.add_theme_font_size_override("font_size", 14)
	_pane.add_child(node)


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


func _display_label(display: Dictionary) -> String:
	var custom := text_value(display.get("custom"), "")
	if custom != "":
		return "custom %s" % custom
	var resolution := text_value(display.get("resolution"), "auto")
	if resolution == "" or resolution == "auto":
		return "auto"
	var rate := text_value(display.get("rate"), "")
	if rate == "":
		rate = text_value(display.get("refresh_hz"), "")
	if rate == "" or rate == "0":
		return resolution
	return "%s @ %s Hz" % [resolution, rate]


func _toggle_expert() -> void:
	var enabled: bool = app.expert_enabled()
	var saved := cli(["settings", "set", "expert.enabled", "false" if enabled else "true"])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save expert mode.")), "error")
		app.show_settings()
		return
	toast("Expert mode %s." % ("disabled" if enabled else "enabled"), "success")
	app.show_settings()


func _run_setup() -> void:
	app.show_setup()


func back() -> void:
	app.show_menu()
