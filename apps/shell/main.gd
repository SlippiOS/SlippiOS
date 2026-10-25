extends Control

const CLI := "slippios"
const KEY_ROWS := ["1234567890", "qwertyuiop", "asdfghjkl", "zxcvbnm", "-_.@"]
const MASK := "•"

var _status: Dictionary = {}
var _screen := ""
var _content: VBoxContainer
var _title: Label
var _status_label: Label
var _play_pid := -1
var _busy := false

var _auth_stage := ""
var _auth_email := ""
var _auth_field := ""
var _auth_label: Label
var _auth_origin := "setup"
var _net_pid := -1
var _net_mode := ""
var _net_url := ""
var _net_timer := 0.0
var _net_status: Label
var _os_pid := -1
var _os_waiting := false
var _os_timer := 0.0
var _os_status: Label


func _ready() -> void:
	DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_FULLSCREEN)
	set_process(false)
	_build_base()
	_refresh_status()
	if _needs_setup():
		_show_setup()
	else:
		_show_menu()
	_animate_boot()


func _build_base() -> void:
	var background := ColorRect.new()
	background.color = Color("#0b0b10")
	background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	background.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(background)

	var center := CenterContainer.new()
	center.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	add_child(center)

	_content = VBoxContainer.new()
	_content.custom_minimum_size = Vector2(760, 0)
	_content.add_theme_constant_override("separation", 12)
	center.add_child(_content)


func _clear() -> void:
	for child in _content.get_children():
		_content.remove_child(child)
		child.queue_free()


func _add_title(text: String, size: int = 56) -> Label:
	var label := Label.new()
	label.text = text
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.add_theme_font_size_override("font_size", size)
	_content.add_child(label)
	return label


func _add_label(text: String, alpha: float = 0.65) -> Label:
	var label := Label.new()
	label.text = text
	label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	label.modulate = Color(1, 1, 1, alpha)
	_content.add_child(label)
	return label


func _add_button(text: String, callback: Callable, height: int = 50) -> Button:
	var button := Button.new()
	button.text = text
	button.custom_minimum_size = Vector2(0, height)
	button.pressed.connect(callback)
	_content.add_child(button)
	return button


func _focus_first() -> void:
	var first := _find_button(_content)
	if first != null:
		first.grab_focus()


func _find_button(node: Node) -> Button:
	if node is Button:
		return node
	for child in node.get_children():
		var found := _find_button(child)
		if found != null:
			return found
	return null


func _animate_boot() -> void:
	if _title == null:
		return
	_title.modulate.a = 0.0
	var tween := create_tween()
	tween.tween_property(_title, "modulate:a", 1.0, 0.7)


func _needs_setup() -> bool:
	var settings: Dictionary = _status.get("settings", {})
	return not settings.get("exists", false) or not settings.get("iso_valid", false)


func _refresh_status() -> void:
	_status = _cli(["status"])


func _status_text() -> String:
	var settings: Dictionary = _status.get("settings", {})
	var login: Dictionary = _status.get("login", {})
	var dolphin: Dictionary = _status.get("dolphin", {})
	var game: String = "ready" if settings.get("iso_valid", false) else str(settings.get("iso_reason", "not configured"))
	var account: String = "signed in" if login.get("signed_in", false) else "not signed in"
	var version: String = str(dolphin.get("current") or "not installed")
	return "Game: %s\nLogin: %s\nDolphin: %s" % [game, account, version]


func _cli(args: Array) -> Dictionary:
	var output: Array = []
	var code := OS.execute(CLI, PackedStringArray(args), output, true)
	var text := ""
	for line in output:
		text += str(line)
	var parsed: Variant = JSON.parse_string(text)
	if typeof(parsed) != TYPE_DICTIONARY:
		return {"ok": false, "error": "invalid CLI output (exit %d)" % code}
	return parsed


func _show_menu() -> void:
	_stop_net()
	_refresh_status()
	_screen = "menu"
	_clear()
	_title = _add_title("SLIPPIOS", 64)
	_add_label("Super Smash Bros. Melee", 0.55)
	_add_button("Play", _on_play)
	_add_button("Update", _show_update)
	_add_button("Account", _show_account)
	_add_button("Settings", _show_settings)
	_add_button("Quit", _on_quit)
	_status_label = _add_label(_status_text(), 0.6)
	_focus_first()


func _show_setup(message: String = "") -> void:
	_stop_net()
	_screen = "setup"
	_clear()
	_title = _add_title("SETUP", 48)
	_add_label("Choose your Melee image (NTSC 1.02).", 0.7)
	if message != "":
		_add_label(message, 0.9)

	var result := _cli(["isos"])
	var found := false
	var isos: Array = result.get("isos", [])
	for entry in isos:
		if entry.get("valid", false):
			found = true
			var path: String = str(entry.get("path", ""))
			var name: String = str(entry.get("name", path))
			_add_button(name, _select_iso.bind(path))

	if not found:
		_add_label("Put an ISO in /data/isos, then press Rescan.", 0.7)

	_add_button("Add game over network", _start_net.bind("iso"))
	_add_button("Rescan", _show_setup.bind(""))
	_add_button("Back", _show_menu)
	_focus_first()


func _select_iso(path: String) -> void:
	var result := _cli(["settings", "set", "game.iso", path])
	if result.get("ok", false):
		_refresh_status()
		_show_login_setup()
	else:
		_show_setup(str(result.get("error", "could not save the game image")))


func _show_login_setup(message: String = "") -> void:
	_stop_net()
	_screen = "login-setup"
	_clear()
	_title = _add_title("SLIPPI ACCOUNT", 44)
	_add_label("Sign in to play online, or skip for offline play.", 0.7)
	if message != "":
		_add_label(message, 0.9)
	_add_button("Sign in", _start_login.bind("setup"))
	_add_button("Sign in on phone", _start_net.bind("login"))
	_add_button("Skip", _show_menu)
	_add_button("Back", _show_setup.bind(""))
	_focus_first()


func _show_account() -> void:
	_stop_net()
	_screen = "account"
	_clear()
	_title = _add_title("ACCOUNT", 44)
	var login: Dictionary = _status.get("login", {})
	if login.get("signed_in", false):
		_add_label("Signed in: %s" % str(login.get("detail", "unknown")), 0.9)
	else:
		_add_label("Not signed in.", 0.8)
		_add_button("Sign in", _start_login.bind("account"))
		_add_button("Sign in on phone", _start_net.bind("login"))
	_add_button("Back", _show_menu)
	_focus_first()


func _start_login(origin: String = "setup") -> void:
	_auth_origin = origin
	_auth_email = ""
	_show_auth_field("email", "")


func _show_auth_field(stage: String, message: String = "") -> void:
	_stop_net()
	_screen = "auth"
	_auth_stage = stage
	_auth_field = ""
	_clear()
	_title = _add_title("EMAIL" if stage == "email" else "PASSWORD", 40)
	_auth_label = _add_label("", 1.0)
	if message != "":
		_add_label(message, 0.9)
	_build_keyboard(stage == "password")
	_add_button("Use phone instead", _start_net.bind("login"))
	_add_button("Back", _back_from_auth)
	_focus_first()


func _build_keyboard(masked: bool) -> void:
	var grid := GridContainer.new()
	grid.columns = 10
	grid.add_theme_constant_override("h_separation", 6)
	grid.add_theme_constant_override("v_separation", 6)
	_content.add_child(grid)

	for row in KEY_ROWS:
		for character in row:
			var key := Button.new()
			key.text = character
			key.custom_minimum_size = Vector2(56, 44)
			key.pressed.connect(_append_character.bind(character, masked))
			grid.add_child(key)

	var actions := HBoxContainer.new()
	actions.add_theme_constant_override("separation", 8)
	_content.add_child(actions)

	var space := Button.new()
	space.text = "Space"
	space.custom_minimum_size = Vector2(300, 44)
	space.pressed.connect(_append_character.bind(" ", masked))
	actions.add_child(space)

	var backspace := Button.new()
	backspace.text = "Delete"
	backspace.custom_minimum_size = Vector2(160, 44)
	backspace.pressed.connect(_delete_character.bind(masked))
	actions.add_child(backspace)

	var done := Button.new()
	done.text = "Done"
	done.custom_minimum_size = Vector2(160, 44)
	done.pressed.connect(_finish_field)
	actions.add_child(done)


func _display_auth(masked: bool) -> void:
	if masked:
		_auth_label.text = MASK.repeat(_auth_field.length())
	else:
		_auth_label.text = _auth_field


func _append_character(character: String, masked: bool) -> void:
	_auth_field += character
	_display_auth(masked)


func _delete_character(masked: bool) -> void:
	if _auth_field.length() > 0:
		_auth_field = _auth_field.substr(0, _auth_field.length() - 1)
	_display_auth(masked)


func _finish_field() -> void:
	if _auth_stage == "email":
		var email := _auth_field.strip_edges()
		if email == "" or not email.contains("@"):
			_show_auth_field("email", "Enter a valid email address.")
			return
		_auth_email = email
		_show_auth_field("password", "")
		return

	_submit_login(_auth_field)


func _submit_login(password: String) -> void:
	if password == "":
		_show_auth_field("password", "Password cannot be empty.")
		return

	var path := ProjectSettings.globalize_path("user://credentials.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		_show_auth_field("password", "Could not prepare the sign-in request.")
		return
	file.store_string(JSON.stringify({"email": _auth_email, "password": password}))
	file.close()
	OS.execute("chmod", PackedStringArray(["600", path]))

	var result := _cli(["login", "--file", path])
	DirAccess.remove_absolute(path)

	if result.get("ok", false):
		_refresh_status()
		_show_account()
	else:
		_show_auth_field("password", str(result.get("error", "Sign-in failed.")))


func _show_settings() -> void:
	_stop_net()
	_screen = "settings"
	_clear()
	_title = _add_title("SETTINGS", 44)
	var result := _cli(["settings", "show"])
	var settings: Dictionary = result.get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var audio: Dictionary = settings.get("audio", {})
	var network: Dictionary = settings.get("network", {})
	_add_label("Refresh: %s Hz" % str(display.get("refresh_hz", "auto")), 0.8)
	_add_label("Resolution: %s" % str(display.get("resolution", "auto")), 0.8)
	_add_label("Audio: %s" % str(audio.get("sink", "auto")), 0.8)
	_add_label("Network: %s" % str(network.get("ssid") or "(none)"), 0.8)
	_add_button("Back", _show_menu)
	_focus_first()


func _show_update() -> void:
	_stop_net()
	_screen = "update"
	_clear()
	_title = _add_title("UPDATE", 44)

	var os_check := _cli(["os", "check"])
	var os_status := _cli(["os", "status"])
	var dolphin := _cli(["update", "--check"])

	if os_check.get("ok", false):
		var current := str(os_check.get("current") or "unknown")
		var latest := str(os_check.get("latest") or "unknown")
		_add_label("System: %s" % current, 0.8)
		if os_check.get("update_available", false):
			_add_label("Available: %s" % latest, 0.8)
			_add_button("Install system update", _start_os_update)
		else:
			_add_label("System is up to date.", 0.65)
	elif os_status.get("phase") == "failed":
		_add_label("Last system update failed.", 0.9)
	else:
		_add_label(str(os_check.get("error", "System update check failed.")), 0.9)

	if os_status.get("phase") == "installed":
		_add_button("Restart to finish update", _request_reboot)

	if dolphin.get("ok", false):
		_add_label("Dolphin: %s" % str(dolphin.get("current") or "not installed"), 0.65)
		if dolphin.get("current") != dolphin.get("latest"):
			_add_label("Latest Dolphin: %s" % str(dolphin.get("latest")), 0.65)
	else:
		_add_label(str(dolphin.get("error", "Dolphin update check failed.")), 0.8)

	_add_button("Back", _show_menu)
	_focus_first()


func _start_os_update() -> void:
	if _os_pid != -1 or _os_waiting:
		return
	_screen = "os-update"
	_clear()
	_title = _add_title("UPDATING SYSTEM", 44)
	_add_label("Downloading the system update. This can take a while.", 0.7)
	_os_status = _add_label("Starting…", 0.8)
	_os_pid = OS.create_process("slippios", PackedStringArray(["os", "update"]))
	_os_waiting = true
	_os_timer = 0.0
	_update_process()


func _poll_os_update() -> void:
	var result := _cli(["os", "status"])
	if not result.get("ok", false):
		if is_instance_valid(_os_status):
			_os_status.text = str(result.get("error", "Update status unavailable."))
		return

	var phase := str(result.get("phase", "idle"))
	var done := int(result.get("done", 0))
	var total := int(result.get("total", 0))
	if phase == "downloading":
		var text := "Downloading"
		if total > 0:
			text += " %d%%" % (done * 100 / total)
		if is_instance_valid(_os_status):
			_os_status.text = text + "…"
	elif phase == "installing":
		if is_instance_valid(_os_status):
			_os_status.text = "Installing to the inactive system slot…"
	elif phase == "installed":
		_os_waiting = false
		_os_pid = -1
		_update_process()
		_show_os_result()
		return
	elif phase == "failed":
		_os_waiting = false
		_os_pid = -1
		_update_process()
		_clear()
		_title = _add_title("UPDATE FAILED", 44)
		_add_label(str(result.get("error", "The system update could not be installed.")), 0.9)
		_add_button("Back", _show_update)
		_focus_first()
		return

	if _os_pid != -1 and not OS.is_process_running(_os_pid):
		_os_pid = -1

	if _os_pid == -1 and phase == "idle" and _os_waiting:
		# The staging command finished without finding a newer version.
		_os_waiting = false
		_update_process()
		_show_update()


func _show_os_result() -> void:
	_screen = "os-update"
	_clear()
	_title = _add_title("UPDATE INSTALLED", 44)
	_add_label("Restart to boot the new system. The previous system is kept for rollback.", 0.75)
	_add_button("Restart now", _request_reboot)
	_add_button("Later", _show_menu)
	_focus_first()


func _request_reboot() -> void:
	var result := _cli(["os", "reboot"])
	if not result.get("ok", false):
		_clear()
		_title = _add_title("RESTART FAILED", 44)
		_add_label(str(result.get("error", "Could not request a restart.")), 0.9)
		_add_button("Back", _show_update)
		_focus_first()
		return
	_clear()
	_title = _add_title("RESTARTING", 44)
	_add_label("The console will reboot into the new system.", 0.75)


func _on_play() -> void:
	if _busy:
		return
	if _needs_setup():
		_show_setup("No valid game image yet.")
		return
	_busy = true
	_clear()
	_title = _add_title("LAUNCHING", 44)
	_add_label("Starting Slippi Dolphin…", 0.8)
	_play_pid = OS.create_process(CLI, PackedStringArray(["play"]))
	set_process(true)


func _process(delta: float) -> void:
	if _net_pid != -1:
		_net_timer += delta
		if _net_timer >= 1.0:
			_net_timer = 0.0
			_poll_net()

	if _os_pid != -1 or _os_waiting:
		_os_timer += delta
		if _os_timer >= 1.0:
			_os_timer = 0.0
			_poll_os_update()

	if _play_pid != -1 and not OS.is_process_running(_play_pid):
		_play_pid = -1
		_busy = false
		_update_process()
		_show_menu()


func _update_process() -> void:
	set_process(_play_pid != -1 or _net_pid != -1 or _os_pid != -1 or _os_waiting)


func _local_ip() -> String:
	for address in IP.get_local_addresses():
		if address.contains(".") and not address.begins_with("127.") and not address.begins_with("169.254."):
			return address
	return "127.0.0.1"


func _start_net(mode: String) -> void:
	_stop_net()
	var system_dir := DirAccess.open("/data/system")
	if system_dir != null:
		system_dir.remove("net-upload.json")
		system_dir.remove("net-status.json")
	var token := Crypto.new().generate_random_bytes(8).hex_encode()
	_net_url = "http://%s:8080/%s/" % [_local_ip(), token]
	_net_mode = mode
	_net_timer = 0.0
	_net_pid = OS.create_process("slippios", PackedStringArray([
		"net", "--token", token, "--port", "8080",
	]))
	_update_process()
	_show_net_screen(mode)


func _stop_net() -> void:
	if _net_pid != -1:
		OS.kill(_net_pid)
		_net_pid = -1
	_update_process()


func _show_net_screen(mode: String) -> void:
	_screen = "net"
	_net_mode = mode
	_clear()
	if mode == "iso":
		_title = _add_title("ADD A GAME", 44)
		_add_label("Open this address on a phone or computer:", 0.8)
	else:
		_title = _add_title("SIGN IN ON YOUR PHONE", 40)
		_add_label("Open this address on a phone or computer:", 0.8)

	_add_label(_net_url, 1.0)
	var qr := _make_qr(_net_url)
	if qr != null:
		_content.add_child(qr)
	_add_label("Upload a Melee ISO." if mode == "iso" else "Enter your Slippi email and password.", 0.7)
	_net_status = _add_label("Waiting…", 0.7)
	_add_button("Back", _net_back)
	_focus_first()


func _make_qr(url: String) -> TextureRect:
	var path := ProjectSettings.globalize_path("user://qr.png")
	var code := OS.execute("qrencode", PackedStringArray([
		"-o", path, "-s", "8", "-m", "2", url,
	]))
	if code != 0:
		return null
	var image := Image.load_from_file(path)
	if image == null:
		return null
	var rect := TextureRect.new()
	rect.texture = ImageTexture.create_from_image(image)
	rect.custom_minimum_size = Vector2(260, 260)
	rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	return rect


func _net_back() -> void:
	var mode := _net_mode
	_stop_net()
	if mode == "iso":
		_show_setup()
	else:
		_show_account()


func _read_json_file(path: String) -> Dictionary:
	if not FileAccess.file_exists(path):
		return {}
	var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(path))
	return parsed if typeof(parsed) == TYPE_DICTIONARY else {}


func _poll_net() -> void:
	if _net_pid == -1:
		return
	if not OS.is_process_running(_net_pid):
		_net_pid = -1
		_update_process()
		var status := _read_json_file("/data/system/net-status.json")
		if is_instance_valid(_net_status):
			if not status.get("ok", true):
				_net_status.text = "Network page stopped: %s" % str(status.get("error", "unknown error"))
			else:
				_net_status.text = "Network page stopped."
		return

	if _net_mode == "iso":
		var marker := _read_json_file("/data/system/net-upload.json")
		if marker.get("name", "") != "":
			_stop_net()
			_refresh_status()
			_show_setup("Game image added.")
			return
		if is_instance_valid(_net_status):
			_net_status.text = "Waiting for upload…"
	else:
		var result := _cli(["status"])
		var login: Dictionary = result.get("login", {})
		if login.get("signed_in", false):
			_stop_net()
			_refresh_status()
			_show_account()
			return
		if is_instance_valid(_net_status):
			_net_status.text = "Waiting for sign-in…"


func _back_from_auth() -> void:
	_stop_net()
	if _auth_origin == "account":
		_show_account()
	else:
		_show_login_setup()


func _unhandled_input(event: InputEvent) -> void:
	if not event.is_action_pressed("ui_cancel"):
		return
	match _screen:
		"net":
			_net_back()
		"auth":
			_back_from_auth()
		"login-setup", "setup", "settings", "update", "account":
			_stop_net()
			_show_menu()


func _on_quit() -> void:
	get_tree().quit()
