extends "res://screens/screen.gd"

## Expert custom display mode entry.

const Keyboard := preload("res://widgets/keyboard.gd")


func present(message: String = "") -> void:
	clear()
	title("CUSTOM MODE", 40)
	label("Enter WIDTHxHEIGHT or WIDTHxHEIGHT@HZ, then Done.", 0.8)
	if message != "":
		label(message, 0.9)
	var keyboard = Keyboard.new(false)
	keyboard.submitted.connect(_apply)
	add_content(keyboard)
	button("Back", app.show_display.bind(""))
	focus()


func _apply(spec: String) -> void:
	var value := spec.strip_edges()
	if not value.contains("x"):
		present("Use WIDTHxHEIGHT, for example 1920x1080.")
		return
	var result := cli(["display", "custom", value])
	if not result.get("ok", false):
		present(str(result.get("error", "The display rejected that mode.")))
		return
	app.show_settings("Custom mode applied: %s." % value)


func back() -> void:
	app.show_display()
