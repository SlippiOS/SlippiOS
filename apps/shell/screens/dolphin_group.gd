extends "res://screens/screen.gd"


var group := "gameplay"


func present() -> void:
	clear()
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	match group:
		"graphics":
			_graphics(dolphin)
		"audio":
			_audio(dolphin)
		"advanced":
			_advanced()
		_:
			_gameplay(dolphin)
	button("Back", app.show_settings)
	focus()


func _gameplay(dolphin: Dictionary) -> void:
	title("DOLPHIN GAMEPLAY", 40)
	button("Build: %s" % text_value(dolphin.get("build"), "stable"),
		_cycle.bind("build", ["stable", "beta"]))
	label("Stable is FM-Slippi. Beta is Mainline.", 0.6)
	button("Controller Settings", app.show_controllers)
	label("Assign each GameCube port to an adapter, a pad, or the keyboard.", 0.6)


func _graphics(dolphin: Dictionary) -> void:
	title("DOLPHIN GRAPHICS", 40)
	button("Renderer: %s" % text_value(dolphin.get("backend"), "Vulkan"),
		_cycle.bind("backend", ["Vulkan", "OGL"]))
	button("Resolution: %s" % _internal_resolution_label(dolphin),
		_cycle.bind("resolution", [1, 2, 3, 4, 5, 6]))
	button("Aspect: %s" % _aspect_label(dolphin),
		_cycle.bind("aspect", [3, 1, 2, 4]))
	button("VSync: %s" % _on_off(dolphin.get("vsync", false)),
		_toggle.bind("vsync"))
	button("Anti-aliasing: %s" % _msaa_label(dolphin),
		_cycle.bind("msaa", [1, 2, 4, 8]))
	button("Precompile shaders: %s" % _on_off(dolphin.get("precompile_shaders", false)),
		_toggle.bind("precompile_shaders"))
	button("Shader mode: %s" % _shader_label(dolphin),
		_cycle.bind("shader_mode", ["sync", "sync_uber", "async_uber", "async_skip"]))
	button("Netplay ping: %s" % _on_off(dolphin.get("show_netplay_ping", true)),
		_toggle.bind("show_netplay_ping"))
	button("Show FPS: %s" % _on_off(dolphin.get("show_fps", false)),
		_toggle.bind("show_fps"))


func _audio(dolphin: Dictionary) -> void:
	title("DOLPHIN AUDIO", 40)
	button("Audio: %s" % text_value(dolphin.get("audio_backend"), "Cubeb"),
		_cycle.bind("audio_backend", ["Cubeb", "ALSA", "NoSound"]))
	button("Music: %s" % _on_off(dolphin.get("enable_music", true)),
		_toggle.bind("enable_music"))


func _advanced() -> void:
	title("DOLPHIN ADVANCED", 40)
	button("Gecko codes", app.show_gecko)
	button("Reset Dolphin config", _reset)
	label("Reset removes Dolphin.ini, GFX.ini, and GCPadNew.ini.", 0.6)


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


func _shader_label(dolphin: Dictionary) -> String:
	match text_value(dolphin.get("shader_mode"), "async_uber"):
		"sync":
			return "Synchronous"
		"sync_uber":
			return "Synchronous (Uber)"
		"async_skip":
			return "Asynchronous (Skip Rendering)"
		_:
			return "Asynchronous (Uber)"


func _on_off(value: Variant) -> String:
	return "on" if bool(value) else "off"


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
		toast(str(saved.get("error", "Could not save the Dolphin setting.")), "error")
		app.show_dolphin_group(group)
		return
	app.show_dolphin_group(group)


func _toggle(key: String) -> void:
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var next := not bool(dolphin.get(key, false))
	var saved := cli(["settings", "set", "dolphin." + key, "true" if next else "false"])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save the Dolphin setting.")), "error")
		app.show_dolphin_group(group)
		return
	app.show_dolphin_group(group)


func _reset() -> void:
	var result := cli(["dolphin", "reset"])
	if not result.get("ok", false):
		toast(str(result.get("error", "Could not reset Dolphin.")), "error")
		app.show_dolphin_group(group)
		return
	toast("Dolphin configuration reset.", "success")
	app.show_dolphin_group(group)


func back() -> void:
	app.show_settings()
