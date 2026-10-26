extends "res://screens/screen.gd"

## Wi-Fi password entry.

const Keyboard := preload("res://widgets/keyboard.gd")

var ssid := ""


func present(message: String = "") -> void:
	clear()
	title("WI-FI PASSWORD", 40)
	label(ssid, 0.8)
	if message != "":
		label(message, 0.9)
	var keyboard = Keyboard.new(true)
	keyboard.submitted.connect(_submit)
	add_content(keyboard)
	button("Back", app.show_wifi.bind(""))
	focus()


func _submit(password: String) -> void:
	app.connect_wifi(ssid, password)


func back() -> void:
	app.show_wifi()
