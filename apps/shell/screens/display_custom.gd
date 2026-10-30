extends "res://screens/screen.gd"


const Keyboard := preload("res://widgets/keyboard.gd")


func present() -> void:
	clear()
	title("CUSTOM MODE", 40)
	label("Enter WIDTHxHEIGHT or WIDTHxHEIGHT@HZ, then Done.", 0.8)
	var keyboard = Keyboard.new(false)
	keyboard.submitted.connect(_apply)
	add_content(keyboard)
	button("Back", app.show_display)
	focus()


func _apply(spec: String) -> void:
	var value := spec.strip_edges()
	if not value.contains("x"):
		toast("Use WIDTHxHEIGHT, for example 1920x1080.", "error")
		return
	var result := cli(["display", "custom", value])
	if not result.get("ok", false):
		toast(str(result.get("error", "The display rejected that mode.")), "error")
		return
	app.restart_shell()


func back() -> void:
	app.show_display()
