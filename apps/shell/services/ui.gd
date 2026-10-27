extends RefCounted


var content: VBoxContainer
var scroll: ScrollContainer


func clear() -> void:
	if content == null:
		return
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


func add_button(text: String, callback: Callable, height: int = 50) -> Button:
	var node := Button.new()
	node.text = text
	node.custom_minimum_size = Vector2(0, height)
	node.pressed.connect(callback)
	content.add_child(node)
	return node


func add_content(node: Control) -> void:
	content.add_child(node)


func focus_first() -> void:
	var first := _find_button(content)
	if first != null:
		first.grab_focus()


func text_value(value: Variant, fallback: String) -> String:
	if value == null:
		return fallback
	var text := str(value)
	return fallback if text == "" else text


func _find_button(node: Node) -> Button:
	if node is Button:
		return node
	for child in node.get_children():
		var found := _find_button(child)
		if found != null:
			return found
	return null
