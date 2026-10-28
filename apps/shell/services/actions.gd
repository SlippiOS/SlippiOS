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
	await _prepare_dolphin("Starting Melee…", GAME_EXIT)


func open_dolphin_settings() -> void:
	await _prepare_dolphin("Opening Dolphin settings…", DOLPHIN_EXIT)


func _prepare_dolphin(starting: String, exit_code: int) -> void:
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("PREPARING", 44)
	var status: Label = app.add_label("Checking for Slippi Dolphin…", 0.8)
	var bar := ProgressBar.new()
	bar.custom_minimum_size = Vector2(0, 24)
	bar.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	bar.max_value = 100
	bar.value = 0
	bar.show_percentage = false
	app.add_content(bar)
	await app.get_tree().process_frame

	var progress_path := "/data/system/dolphin-progress.json"
	DirAccess.remove_absolute(progress_path)
	var pid := OS.create_process("slippios",
		PackedStringArray(["update", "--progress", progress_path]))
	if pid <= 0:
		_start_error("Could not start the Slippi Dolphin update.")
		return

	while OS.is_process_running(pid):
		_show_progress(_read_progress(progress_path), status, bar)
		await app.get_tree().create_timer(0.15).timeout

	if OS.get_process_exit_code(pid) != 0:
		var data := _read_progress(progress_path)
		_start_error(str(data.get("error", "Could not prepare Slippi Dolphin.")))
		return
	bar.value = 100
	status.text = starting
	await app.get_tree().create_timer(0.2).timeout
	app.get_tree().quit(exit_code)


func _read_progress(path: String) -> Dictionary:
	if not FileAccess.file_exists(path):
		return {}
	var file := FileAccess.open(path, FileAccess.READ)
	if file == null:
		return {}
	var parsed: Variant = JSON.parse_string(file.get_as_text())
	if typeof(parsed) != TYPE_DICTIONARY:
		return {}
	return parsed


func _show_progress(data: Dictionary, status: Label, bar: ProgressBar) -> void:
	match str(data.get("phase", "")):
		"downloading":
			var total := float(data.get("total", 0))
			var done := float(data.get("done", 0))
			if total > 0:
				bar.value = clampf(done / total * 100.0, 0.0, 100.0)
				status.text = "Downloading Slippi Dolphin… %d%%" % int(bar.value)
			else:
				status.text = "Downloading Slippi Dolphin…"
		"ready":
			bar.value = 100
			status.text = "Preparing…"
		"error":
			status.text = str(data.get("error", "Could not prepare Slippi Dolphin."))
		_:
			status.text = "Checking for Slippi Dolphin…"


func _start_error(message: String) -> void:
	app._busy = false
	app.clear()
	app.add_title("COULD NOT START", 44)
	app.add_label(message, 0.9)
	app.add_button("Back", app.show_menu)
	app.focus_first()


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
