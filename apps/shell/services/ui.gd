extends RefCounted


var content: VBoxContainer
var scroll: ScrollContainer
var preserve_focus := true

var _focus_key := ""


func clear() -> void:
	if content == null:
		return
	if preserve_focus:
		_remember_focus()
	else:
		_focus_key = ""
	if scroll != null:
		scroll.scroll_vertical = 0
	for child in content.get_children():
		content.remove_child(child)
		child.queue_free()


func add_title(text: String, size: int = 56) -> Label:
	var node := Label.new()
	node.text = text
	node.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	node.add_theme_font_size_override("font_size", size)
	content.add_child(node)
	return node


func add_label(text: String, alpha: float = 0.65) -> Label:
	var node := Label.new()
	node.text = text
	node.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	node.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	node.modulate = Color(1, 1, 1, alpha)
	content.add_child(node)
	return node


func add_section(text: String) -> void:
	if content == null:
		return
	var spacer := Control.new()
	spacer.custom_minimum_size = Vector2(0, 6)
	spacer.mouse_filter = Control.MOUSE_FILTER_IGNORE
	content.add_child(spacer)

	var row := HBoxContainer.new()
	row.mouse_filter = Control.MOUSE_FILTER_IGNORE
	row.add_theme_constant_override("separation", 8)
	var bar := ColorRect.new()
	bar.color = SlippiTheme.brand
	bar.custom_minimum_size = Vector2(3, 12)
	bar.size_flags_vertical = Control.SIZE_SHRINK_CENTER
	bar.mouse_filter = Control.MOUSE_FILTER_IGNORE
	row.add_child(bar)
	var node := Label.new()
	node.text = text.to_upper()
	node.mouse_filter = Control.MOUSE_FILTER_IGNORE
	node.add_theme_font_size_override("font_size", 13)
	node.add_theme_color_override("font_color", SlippiTheme.brand)
	row.add_child(node)
	content.add_child(row)


func add_button(text: String, callback: Callable, height: int = 46) -> Button:
	var node := Button.new()
	node.text = text
	node.custom_minimum_size = Vector2(0, height)
	node.pressed.connect(callback)
	content.add_child(node)
	return node


func add_content(node: Control) -> void:
	content.add_child(node)


func focus_first() -> void:
	if _focus_key != "":
		var previous := _find_button_by_key(content, _focus_key)
		_focus_key = ""
		if previous != null:
			previous.grab_focus()
			return
	var first := _find_button(content)
	if first != null:
		first.grab_focus()


func text_value(value: Variant, fallback: String) -> String:
	if value == null:
		return fallback
	var text := str(value)
	return fallback if text == "" else text


func _remember_focus() -> void:
	_focus_key = ""
	if content == null:
		return
	var focused := content.get_viewport().gui_get_focus_owner()
	if focused is Button and content.is_ancestor_of(focused):
		_focus_key = _button_key(focused.text)


func _button_key(text: String) -> String:
	var key := text
	if key.begins_with("[x] ") or key.begins_with("[ ] "):
		key = key.substr(4)
	var colon := key.find(":")
	if colon > 0:
		key = key.substr(0, colon)
	var dash := key.find(" — ")
	if dash > 0:
		key = key.substr(0, dash)
	return key.strip_edges()


func _find_button(node: Node) -> Button:
	if node is Button:
		return node
	for child in node.get_children():
		var found := _find_button(child)
		if found != null:
			return found
	return null


func _find_button_by_key(node: Node, key: String) -> Button:
	if node is Button and _button_key(node.text) == key:
		return node
	for child in node.get_children():
		var found := _find_button_by_key(child, key)
		if found != null:
			return found
	return null
