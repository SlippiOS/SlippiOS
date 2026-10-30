extends "res://screens/screen.gd"

const Keyboard := preload("res://widgets/keyboard.gd")

var ssid := ""


func present() -> void:
	clear()
	title("WI-FI PASSWORD", 40)
	label(ssid, 0.8)
	var keyboard = Keyboard.new(true)
	keyboard.submitted.connect(_submit)
	add_content(keyboard)
	button("Back", app.show_wifi)
	focus()


func _submit(password: String) -> void:
	app.connect_wifi(ssid, password)


func back() -> void:
	app.show_wifi()
