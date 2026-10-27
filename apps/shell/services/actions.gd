extends RefCounted


const GAME_EXIT := 42
const DOLPHIN_EXIT := 43
const TERMINAL_EXIT := 44
const EDITOR_EXIT := 45

var app


func _init(shell) -> void:
	app = shell


func launch_game() -> void:
	if app._busy:
		return
	if app.needs_setup():
		app.show_setup("No valid game image yet.")
		return
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("LAUNCHING", 44)
	app.add_label("Preparing Slippi Dolphin…", 0.8)
	await app.get_tree().process_frame

	# Godot must exit before Dolphin opens the adapter.
	var result: Dictionary = app.cli(["update"], false)
	if not result.get("ok", false):
		app._busy = false
		app.clear()
		app.add_title("COULD NOT START", 44)
		app.add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		app.add_button("Back", app.show_menu)
		app.focus_first()
		return
	app.get_tree().quit(GAME_EXIT)


func open_dolphin_settings() -> void:
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("LAUNCHING", 44)
	app.add_label("Preparing Slippi Dolphin…", 0.8)
	await app.get_tree().process_frame

	var result: Dictionary = app.cli(["update"], false)
	if not result.get("ok", false):
		app._busy = false
		app.clear()
		app.add_title("COULD NOT START", 44)
		app.add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		app.add_button("Back", app.show_menu)
		app.focus_first()
		return
	app.get_tree().quit(DOLPHIN_EXIT)


func open_terminal() -> void:
	app._screen_node = null
	app.clear()
	app.add_title("TERMINAL", 44)
	app.add_label("Opening a terminal…", 0.8)
	await app.get_tree().process_frame
	app.get_tree().quit(TERMINAL_EXIT)


func open_editor() -> void:
	app._screen_node = null
	app.clear()
	app.add_title("TEXT EDITOR", 44)
	app.add_label("Opening settings.toml…", 0.8)
	await app.get_tree().process_frame
	app.get_tree().quit(EDITOR_EXIT)


func connect_wifi(ssid: String, password: String) -> void:
	var result: Dictionary = app.cli(["wifi", "connect", "--ssid", ssid, "--password", password])
	if result.get("ok", false):
		app.refresh_status()
		if app.setup_flow:
			app.show_setup("Wi-Fi connected.")
		else:
			app.show_wifi("Connected to %s" % ssid)
	elif password == "":
		app.show_wifi("Could not connect: %s" % str(result.get("error", "unknown error")))
	else:
		app.show_wifi_password(ssid, str(result.get("error", "Could not connect.")))
