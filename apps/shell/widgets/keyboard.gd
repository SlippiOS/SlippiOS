extends VBoxContainer


signal submitted(value: String)

const KEY_ROWS := ["1234567890", "qwertyuiop", "asdfghjkl", "zxcvbnm", "-_.@", "!@#$%^&*()", "+=[]{};:'\",.<>/?"]
const MASK := "•"

var masked := false
var value := ""

var _field: Label


func _init(is_masked: bool) -> void:
	masked = is_masked
	add_theme_constant_override("separation", 8)
	_field = Label.new()
	_field.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_field.custom_minimum_size = Vector2(0, 32)
	add_child(_field)
	var hint := Label.new()
	hint.text = "Keyboard or on-screen keys"
	hint.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	hint.modulate = Color(1, 1, 1, 0.45)
	add_child(hint)
	_build()


func _build() -> void:
	var grid := GridContainer.new()
	grid.columns = 10
	grid.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	grid.add_theme_constant_override("h_separation", 6)
	grid.add_theme_constant_override("v_separation", 6)
	add_child(grid)

	for row in KEY_ROWS:
		for character in row:
			var key := Button.new()
			key.text = character
			key.custom_minimum_size = Vector2(0, 40)
			key.size_flags_horizontal = Control.SIZE_EXPAND_FILL
			key.pressed.connect(_append.bind(character))
			grid.add_child(key)

	var actions := HBoxContainer.new()
	actions.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	actions.add_theme_constant_override("separation", 8)
	add_child(actions)

	var space := Button.new()
	space.text = "Space"
	space.custom_minimum_size = Vector2(0, 40)
	space.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	space.size_flags_stretch_ratio = 2.0
	space.pressed.connect(_append.bind(" "))
	actions.add_child(space)

	var backspace := Button.new()
	backspace.text = "Delete"
	backspace.custom_minimum_size = Vector2(0, 40)
	backspace.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	backspace.pressed.connect(_delete)
	actions.add_child(backspace)

	var done := Button.new()
	done.text = "Done"
	done.custom_minimum_size = Vector2(0, 40)
	done.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	done.pressed.connect(_submit)
	actions.add_child(done)

	_refresh()


func _input(event: InputEvent) -> void:
	if not is_visible_in_tree():
		return
	if not (event is InputEventKey) or not event.pressed or event.echo:
		return
	if event.keycode == KEY_BACKSPACE or event.keycode == KEY_DELETE:
		get_viewport().set_input_as_handled()
		_delete()
	elif event.keycode == KEY_ENTER or event.keycode == KEY_KP_ENTER:
		get_viewport().set_input_as_handled()
		_submit()
	elif event.unicode >= 32:
		get_viewport().set_input_as_handled()
		_append(char(event.unicode))


func _append(character: String) -> void:
	value += character
	_refresh()


func _delete() -> void:
	if value.length() > 0:
		value = value.substr(0, value.length() - 1)
	_refresh()


func _submit() -> void:
	submitted.emit(value)


func _refresh() -> void:
	_field.text = MASK.repeat(value.length()) if masked else value
