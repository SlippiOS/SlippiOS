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
	_build()


func _build() -> void:
	var grid := GridContainer.new()
	grid.columns = 10
	grid.add_theme_constant_override("h_separation", 6)
	grid.add_theme_constant_override("v_separation", 6)
	add_child(grid)

	for row in KEY_ROWS:
		for character in row:
			var key := Button.new()
			key.text = character
			key.custom_minimum_size = Vector2(56, 44)
			key.pressed.connect(_append.bind(character))
			grid.add_child(key)

	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 8)
	add_child(actions)

	var space := Button.new()
	space.text = "Space"
	space.custom_minimum_size = Vector2(300, 44)
	space.pressed.connect(_append.bind(" "))
	actions.add_child(space)

	var backspace := Button.new()
	backspace.text = "Delete"
	backspace.custom_minimum_size = Vector2(160, 44)
	backspace.pressed.connect(_delete)
	actions.add_child(backspace)

	var done := Button.new()
	done.text = "Done"
	done.custom_minimum_size = Vector2(160, 44)
	done.pressed.connect(_submit)
	actions.add_child(done)

	_refresh()


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
