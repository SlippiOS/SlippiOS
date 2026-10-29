extends "res://screens/screen.gd"


var launch_after := false


func present(message: String = "") -> void:
	clear()
	title("CONTROLLER", 44)
	var result := cli(["controllers", "list"])
	var pads: Array = array_value(result.get("pads", []))
	var ports: Array = array_value(result.get("ports", []))
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the controller settings.")), 0.9)
	elif pads.is_empty():
		label("No wired controller detected.", 0.8)
		label("Connect a controller, then retry.", 0.6)
		button("Retry", present.bind(""))
	else:
		var names := PackedStringArray()
		for pad in pads:
			names.append(str(pad))
		label("Detected: %s" % ", ".join(names), 0.8)
		var profile := _port1_profile(ports)
		if profile != "":
			label("Port 1: standard pad (%s)" % profile, 0.7)
		else:
			label("Port 1: GameCube adapter", 0.7)
	if message != "":
		label(message, 0.9)
	button("Continue", _continue)
	button("Controller settings", app.show_controllers)
	focus()


func _port1_profile(ports: Array) -> String:
	for item in ports:
		if int(item.get("port", 0)) == 1 and str(item.get("device", "")) == "pad":
			return str(item.get("profile", ""))
	return ""


func _continue() -> void:
	if launch_after and not app.needs_setup():
		app.launch_game()
		return
	app.skip_setup()


func back() -> void:
	app.show_setup()
