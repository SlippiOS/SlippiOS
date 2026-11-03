extends "res://screens/screen.gd"


const CategoryMenu := preload("res://widgets/category_menu.gd")
const CATEGORIES := ["Gameplay", "Graphics", "Audio", "Advanced"]

static var selected := 0

var _menu
var _dolphin: Dictionary = {}


func present() -> void:
	clear()
	title("DOLPHIN SETTINGS", 44)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	_dolphin = settings.get("dolphin", {})
	_menu = CategoryMenu.new(app, CATEGORIES, _build_category, app.show_settings)
	_menu.build(selected)
	_menu.focus_selected()


func _build_category(index: int, menu) -> void:
	selected = index
	match index:
		0:
			_gameplay(menu)
		1:
			_graphics(menu)
		2:
			_audio(menu)
		3:
			_advanced(menu)


func _gameplay(menu) -> void:
	menu.pane_title("Gameplay")
	menu.pane_button("Build: %s" % text_value(_dolphin.get("build"), "stable"),
		_cycle.bind("build", ["stable", "beta"]))
	menu.pane_hint("Stable is FM-Slippi. Beta is Mainline.")
	menu.pane_button("Controller Settings", app.show_controllers)
	menu.pane_hint("Assign each GameCube port to an adapter, a pad, or the keyboard.")


func _graphics(menu) -> void:
	menu.pane_title("Graphics")
	menu.pane_button("Renderer: %s" % text_value(_dolphin.get("backend"), "Vulkan"),
		_cycle.bind("backend", ["Vulkan", "OGL"]))
	menu.pane_button("Resolution: %s" % _internal_resolution_label(_dolphin),
		_cycle.bind("resolution", [1, 2, 3, 4, 5, 6]))
	menu.pane_button("Aspect: %s" % _aspect_label(_dolphin),
		_cycle.bind("aspect", [3, 1, 2, 4]))
	menu.pane_button("VSync: %s" % _on_off(_dolphin.get("vsync", false)),
		_toggle.bind("vsync"))
	menu.pane_button("Anti-aliasing: %s" % _msaa_label(_dolphin),
		_cycle.bind("msaa", [1, 2, 4, 8]))
	menu.pane_button("Precompile shaders: %s" % _on_off(_dolphin.get("precompile_shaders", false)),
		_toggle.bind("precompile_shaders"))
	menu.pane_button("Shader mode: %s" % _shader_label(_dolphin),
		_cycle.bind("shader_mode", ["sync", "sync_uber", "async_uber", "async_skip"]))
	menu.pane_button("Netplay ping: %s" % _on_off(_dolphin.get("show_netplay_ping", true)),
		_toggle.bind("show_netplay_ping"))
	menu.pane_button("Show FPS: %s" % _on_off(_dolphin.get("show_fps", false)),
		_toggle.bind("show_fps"))


func _audio(menu) -> void:
	menu.pane_title("Audio")
	menu.pane_button("Audio: %s" % text_value(_dolphin.get("audio_backend"), "Cubeb"),
		_cycle.bind("audio_backend", ["Cubeb", "ALSA", "NoSound"]))
	menu.pane_button("Music: %s" % _on_off(_dolphin.get("enable_music", true)),
		_toggle.bind("enable_music"))
	menu.pane_hint("Music plays in Slippi menus and results.")


func _advanced(menu) -> void:
	menu.pane_title("Advanced")
	menu.pane_button("Gecko codes", app.show_gecko)
	menu.pane_button("Reset Dolphin config", _reset)
	menu.pane_hint("Reset removes Dolphin.ini, GFX.ini, and GCPadNew.ini.")


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
		app.show_dolphin()
		return
	app.show_dolphin()


func _toggle(key: String) -> void:
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var next := not bool(dolphin.get(key, false))
	var saved := cli(["settings", "set", "dolphin." + key, "true" if next else "false"])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save the Dolphin setting.")), "error")
		app.show_dolphin()
		return
	app.show_dolphin()


func _reset() -> void:
	var result := cli(["dolphin", "reset"])
	if not result.get("ok", false):
		toast(str(result.get("error", "Could not reset Dolphin.")), "error")
		app.show_dolphin()
		return
	toast("Dolphin configuration reset.", "success")
	app.show_dolphin()


func back() -> void:
	app.show_settings()
