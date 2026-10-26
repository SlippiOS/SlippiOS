extends RefCounted

## Base for shell screens. A screen builds controls into the app's content
## box and navigates by calling app.show_*().

var app


func _init(shell) -> void:
	app = shell


func present(_message: String = "") -> void:
	pass


func back() -> void:
	pass


func clear() -> void:
	app.clear()


func title(text: String, size: int = 56) -> Label:
	return app.add_title(text, size)


func label(text: String, alpha: float = 0.65) -> Label:
	return app.add_label(text, alpha)


func button(text: String, callback: Callable, height: int = 50) -> Button:
	return app.add_button(text, callback, height)


func add_content(node: Control) -> void:
	app.add_content(node)


func focus() -> void:
	app.focus_first()


func cli(args: Array, read_stderr: bool = true) -> Dictionary:
	return app.cli(args, read_stderr)


func text_value(value: Variant, fallback: String) -> String:
	return app.text_value(value, fallback)
