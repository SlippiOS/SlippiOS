extends "res://screens/screen.gd"


func present(message: String = "") -> void:
	clear()
	title("DOLPHIN SETTINGS", 40)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	if message != "":
		label(message, 0.9)
	button("Graphics: %s" % text_value(dolphin.get("backend"), "Vulkan"),
		_cycle.bind("backend", ["Vulkan", "OGL"]))
	button("Resolution: %s" % _internal_resolution_label(dolphin),
		_cycle.bind("resolution", [1, 2, 3, 4, 5, 6]))
	button("Aspect: %s" % _aspect_label(dolphin),
		_cycle.bind("aspect", [3, 1, 2, 4]))
	button("VSync: %s" % ("on" if dolphin.get("vsync", false) else "off"),
		_toggle.bind("vsync"))
	button("Anti-aliasing: %s" % _msaa_label(dolphin),
		_cycle.bind("msaa", [1, 2, 4, 8]))
	button("Audio: %s" % text_value(dolphin.get("audio_backend"), "Cubeb"),
		_cycle.bind("audio_backend", ["Cubeb", "ALSA", "NoSound"]))
	button("Gecko codes", app.show_gecko)
	button("Reset Dolphin config", _reset)
	button("Back", app.show_settings)
	focus()


func _internal_resolution_label(dolphin: Dictionary) -> String:
	var value := int(dolphin.get("resolution", 3))
	if value <= 0:
		return "auto"
	return "%dx native" % value


func _aspect_label(dolphin: Dictionary) -> String:
	match int(dolphin.get("aspect", 3)):
		1:
			return "widescreen"
		2:
			return "4:3"
		4:
			return "stretch"
		_:
			return "Melee (73:60)"


func _msaa_label(dolphin: Dictionary) -> String:
	var value := int(dolphin.get("msaa", 1))
	if value <= 1:
		return "off"
	return "%dx" % value


func _cycle(key: String, choices: Array) -> void:
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var current: Variant = dolphin.get(key)
	var index := -1
	for position in choices.size():
		var choice: Variant = choices[position]
		if typeof(choice) in [TYPE_INT, TYPE_FLOAT] and typeof(current) in [TYPE_INT, TYPE_FLOAT]:
			if float(choice) == float(current):
				index = position
				break
		elif str(choice) == str(current):
			index = position
			break
	if index < 0:
		index = 0
	var next: Variant = choices[(index + 1) % choices.size()]
	var saved := cli(["settings", "set", "dolphin." + key, str(next)])
	if not saved.get("ok", false):
		app.show_dolphin(str(saved.get("error", "Could not save the Dolphin setting.")))
		return
	app.show_dolphin()


func _toggle(key: String) -> void:
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var next := not bool(dolphin.get(key, false))
	var saved := cli(["settings", "set", "dolphin." + key, "true" if next else "false"])
	if not saved.get("ok", false):
		app.show_dolphin(str(saved.get("error", "Could not save the Dolphin setting.")))
		return
	app.show_dolphin()


func _reset() -> void:
	var result := cli(["dolphin", "reset"])
	if not result.get("ok", false):
		app.show_dolphin(str(result.get("error", "Could not reset Dolphin.")))
		return
	app.show_dolphin("Dolphin configuration reset.")


func back() -> void:
	app.show_settings()
