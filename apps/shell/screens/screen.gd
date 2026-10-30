extends RefCounted


var app


func _init(shell) -> void:
	app = shell


func present() -> void:
	pass


func back() -> void:
	pass


func clear() -> void:
	app.clear()


func title(text: String, size: int = 56) -> Label:
	return app.add_title(text, size)


func label(text: String, alpha: float = 0.65) -> Label:
	return app.add_label(text, alpha)


func section(text: String) -> void:
	app.add_section(text)


func button(text: String, callback: Callable, height: int = 46) -> Button:
	return app.add_button(text, callback, height)


func add_content(node: Control) -> void:
	app.add_content(node)


func toast(text: String, kind: String = "info") -> void:
	app.toast(text, kind)


func focus() -> void:
	app.focus_first()


func focus_or(fallback: Control) -> void:
	app.focus_or(fallback)


func cli(args: Array, read_stderr: bool = true) -> Dictionary:
	return app.cli(args, read_stderr)


func text_value(value: Variant, fallback: String) -> String:
	return app.text_value(value, fallback)


func array_value(value: Variant) -> Array:
	if value is Array:
		return value
	return []
