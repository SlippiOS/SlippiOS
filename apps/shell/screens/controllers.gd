extends "res://screens/screen.gd"


func present(message: String = "") -> void:
	clear()
	title("CONTROLLER SETTINGS", 36)
	label("Assign an input to each GameCube port.", 0.7)
	if message != "":
		label(message, 0.9)

	var result := cli(["controllers", "list"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the controller settings.")), 0.9)
		button("Back", app.show_settings)
		focus()
		return

	for port in result.get("ports", []):
		var number := int(port.get("port", 0))
		if number < 1:
			continue
		button("Port %d: %s" % [number, _device_label(port)],
			app.show_controller_port.bind(number))

	label("A GameCube adapter provides all four ports. Standard pads use a Dolphin profile.", 0.6)
	button("Back", app.show_settings)
	focus()


func _device_label(port: Dictionary) -> String:
	match text_value(port.get("device"), "adapter"):
		"pad":
			var profile := text_value(port.get("profile"), "custom")
			return "standard pad (%s)" % profile
		"keyboard":
			return "keyboard"
		"none":
			return "none"
		_:
			return "GameCube adapter"


func back() -> void:
	app.show_settings()
