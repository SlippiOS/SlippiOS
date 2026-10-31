extends RefCounted


const Pulse := preload("res://pulse.gd")

const GAME_EXIT := 42
const DOLPHIN_EXIT := 43
const TERMINAL_EXIT := 44
const EDITOR_EXIT := 45
const REPLAY_EXIT := 47
const GAME_READY_EXIT := 48

const READY_PATH := "/data/system/game-ready"
const PID_PATH := "/data/system/game.pid"
const CANCEL_PATH := "/data/system/game-cancel"

var app
var _launch_pid := -1
var _launch_cancelled := false


func _init(shell) -> void:
	app = shell


func launching() -> bool:
	return _launch_pid > 0


func cancel_launch() -> void:
	if _launch_pid <= 0 or _launch_cancelled:
		return
	_launch_cancelled = true
	app.clear()
	app.add_title("LAUNCHING", 44)
	app.add_label("Cancelling…", 0.8)
	var file := FileAccess.open(CANCEL_PATH, FileAccess.WRITE)
	if file != null:
		file.store_string("cancel\n")
		file.close()
	# The CLI stops Dolphin's process group when the cancel file appears.
	# Wait for it so the menu is not shown behind a still-running game.
	var pid := _launch_pid
	var waited := 0.0
	while OS.is_process_running(pid) and waited < 10.0:
		await app.get_tree().create_timer(0.2).timeout
		waited += 0.2
	if OS.is_process_running(pid):
		OS.kill(pid)
	# _launch_in_shell observes the exit and presents the menu.


func launch_game() -> void:
	if app._busy:
		return
	if app.needs_setup():
		app.toast("No valid game image yet.")
		app.show_setup()
		return
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("PREPARING", 44)
	var status: Label = app.add_label("Checking for Slippi Dolphin…", 0.8)
	var bar := _progress_bar()
	await app.get_tree().process_frame
	var ready := await _ensure_dolphin(status, bar)
	if not ready:
		return
	if app.adapter_present():
		# The GameCube adapter is exclusive; Godot must exit before Dolphin
		# opens it. The wrapper relaunches the shell when Dolphin exits.
		bar.value = 100
		status.text = "Starting Melee…"
		await app.get_tree().create_timer(0.2).timeout
		app.get_tree().quit(GAME_EXIT)
		return
	await _launch_in_shell()


func open_dolphin_settings() -> void:
	await _prepare_dolphin("Opening Dolphin settings…", DOLPHIN_EXIT)


func _prepare_dolphin(starting: String, exit_code: int) -> void:
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("PREPARING", 44)
	var status: Label = app.add_label("Checking for Slippi Dolphin…", 0.8)
	var bar := _progress_bar()
	await app.get_tree().process_frame
	var ready := await _ensure_dolphin(status, bar)
	if not ready:
		return
	bar.value = 100
	status.text = starting
	await app.get_tree().create_timer(0.2).timeout
	app.get_tree().quit(exit_code)


func _progress_bar() -> ProgressBar:
	var bar := ProgressBar.new()
	bar.custom_minimum_size = Vector2(0, 24)
	bar.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	bar.max_value = 100
	bar.value = 0
	bar.show_percentage = false
	app.add_content(bar)
	return bar


func _ensure_dolphin(status: Label, bar: ProgressBar) -> bool:
	var progress_path := "/data/system/dolphin-progress.json"
	DirAccess.remove_absolute(progress_path)
	var pid := OS.create_process("slippios",
		PackedStringArray(["update", "--progress", progress_path]))
	if pid <= 0:
		_start_error("Could not start the Slippi Dolphin update.")
		return false

	while OS.is_process_running(pid):
		_show_progress(_read_progress(progress_path), status, bar)
		await app.get_tree().create_timer(0.15).timeout

	if OS.get_process_exit_code(pid) != 0:
		var data := _read_progress(progress_path)
		_start_error(str(data.get("error", "Could not prepare Slippi Dolphin.")))
		return false
	return true


func _launch_in_shell() -> void:
	app.clear()
	app.add_title("LAUNCHING", 44)
	app.add_label("Super Smash Bros. Melee", 0.9)
	var pulse := Pulse.new()
	app.add_content(pulse)
	var status: Label = app.add_label("Starting Dolphin…", 0.6)
	await app.get_tree().process_frame

	DirAccess.remove_absolute(READY_PATH)
	DirAccess.remove_absolute(PID_PATH)
	DirAccess.remove_absolute(CANCEL_PATH)
	_launch_cancelled = false
	var pid := OS.create_process("slippios",
		PackedStringArray(["play", "--ready-file", READY_PATH,
			"--cancel-file", CANCEL_PATH]))
	if pid <= 0:
		_start_error("Could not start Slippi Dolphin.")
		return
	_launch_pid = pid
	var pid_file := FileAccess.open(PID_PATH, FileAccess.WRITE)
	if pid_file != null:
		pid_file.store_string(str(pid))
		pid_file.close()

	while OS.is_process_running(pid):
		if FileAccess.file_exists(READY_PATH):
			_focus_game_window()
			_launch_pid = -1
			DirAccess.remove_absolute(CANCEL_PATH)
			app.get_tree().quit(GAME_READY_EXIT)
			return
		await app.get_tree().create_timer(0.2).timeout
	var code := OS.get_process_exit_code(pid)
	_launch_pid = -1
	DirAccess.remove_absolute(CANCEL_PATH)
	app._busy = false
	if _launch_cancelled:
		app.show_menu()
		return
	if FileAccess.file_exists(READY_PATH):
		# The game window opened, then Dolphin exited before the shell could
		# hand off. Treat that as a normal return to the menu.
		app.show_menu()
		return
	if code == 0:
		# A clean Dolphin exit can beat the ready-file poll on a slow start.
		app.show_menu()
		return
	_start_error("Dolphin exited before the game window opened.")


func _focus_game_window() -> void:
	var output: Array = []
	if OS.execute("xdotool",
			["search", "--onlyvisible", "--name", "Faster Melee|Dolphin"],
			output, true) != 0:
		return
	for entry in output:
		for id in str(entry).strip_edges().split("\n"):
			if id == "":
				continue
			var props: Array = []
			var state := ""
			if OS.execute("xprop", ["-id", id, "_NET_WM_STATE"], props, true) == 0 and not props.is_empty():
				state = str(props[0])
			var names: Array = []
			var name := ""
			if OS.execute("xdotool", ["getwindowname", id], names, true) == 0 and not names.is_empty():
				name = str(names[0])
			if state.contains("_NET_WM_STATE_FULLSCREEN"):
				OS.execute("xdotool", ["windowactivate", id], [], false)
			elif name.contains("Faster Melee"):
				# Dolphin keeps its small launcher frame mapped behind the
				# fullscreen render window. Hide it so the game fills the screen.
				OS.execute("xdotool", ["windowunmap", id], [], false)


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


func play_replay(path: String) -> void:
	if app._busy:
		return
	app._busy = true
	app._screen_node = null
	app.clear()
	app.add_title("STARTING REPLAY", 40)
	app.add_label("Preparing Slippi playback…", 0.8)
	await app.get_tree().process_frame

	var file := FileAccess.open("/data/system/replay-request", FileAccess.WRITE)
	if file == null:
		app._busy = false
		app.clear()
		app.add_title("COULD NOT START", 44)
		app.add_label("Could not write the replay request.", 0.9)
		app.add_button("Back", app.show_menu)
		app.focus_first()
		return
	file.store_string(path)
	file.close()
	app.get_tree().quit(REPLAY_EXIT)


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
			app.toast("Wi-Fi connected.", "success")
			app.show_setup()
		else:
			app.toast("Connected to %s." % ssid, "success")
			app.show_wifi()
	elif password == "":
		app.toast("Could not connect: %s" % str(result.get("error", "unknown error")), "error")
		app.show_wifi()
	else:
		app.toast(str(result.get("error", "Could not connect.")), "error")
		app.show_wifi_password(ssid)
