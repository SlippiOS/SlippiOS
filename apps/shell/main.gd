extends Control

const CLI := "slippios"
const GAME_EXIT := 42
const DOLPHIN_EXIT := 43
const TERMINAL_EXIT := 44
const EDITOR_EXIT := 45
const MAX_MODE_ROWS := 24
const REFRESH_CHOICES := [0, 60, 120, 144, 240]
const RESOLUTION_CHOICES := ["auto", "1280x720", "1920x1080", "2560x1440", "3840x2160"]
const BOOT_HOLD_MS := 1800
const BootLogo := preload("res://boot_logo.gd")
const KEY_ROWS := ["1234567890", "qwertyuiop", "asdfghjkl", "zxcvbnm", "-_.@", "!@#$%^&*()", "+=[]{};:'\",.<>/?"]
const MASK := "•"

var _status: Dictionary = {}
var _screen := ""
var _content: VBoxContainer
var _title: Label
var _status_label: Label
var _busy := false

var _auth_stage := ""
var _auth_email := ""
var _auth_field := ""
var _auth_label: Label
var _input_mode := "auth"
var _wifi_ssid := ""
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
	# The graphical session does not run slippios-session, so apply the saved
	# display mode here (including re-adding a custom mode after a reboot).
	_cli(["display", "apply"])
	if OS.get_environment("SLIPPIOS_SKIP_SPLASH") == "1":
		_after_splash()
	else:
		_boot_splash()


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


func _text(value: Variant, fallback: String) -> String:
	if value == null:
		return fallback
	var text := str(value)
	return fallback if text == "" else text


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


func _boot_splash() -> void:
	_screen = "boot"
	_clear()
	var logo := BootLogo.new()
	_content.add_child(logo)
	_title = _add_title("SLIPPIOS", 64)
	_title.modulate.a = 0.0
	var prompt := _add_label("Hold A for settings", 0.0)
	var bar := _add_boot_bar()
	var tween := create_tween().set_parallel(true)
	tween.tween_property(_title, "modulate:a", 1.0, 0.55)
	tween.tween_property(prompt, "modulate:a", 0.65, 0.45).set_delay(0.75)
	tween.tween_property(bar, "value", 1.0, 1.6).set_trans(Tween.TRANS_SINE)

	var deadline := Time.get_ticks_msec() + BOOT_HOLD_MS
	while Time.get_ticks_msec() < deadline:
		if _settings_held():
			tween.kill()
			_show_menu()
			return
		await get_tree().process_frame
	_after_splash()


func _after_splash() -> void:
	if _needs_setup():
		var settings: Dictionary = _status.get("settings", {})
		if settings.get("exists", false) and _has_valid_isos():
			_show_roms("Default Melee image is missing. Choose one.")
		else:
			_check_setup()
	elif OS.get_environment("SLIPPIOS_AUTOPLAY") == "0":
		_show_menu()
	else:
		_launch_game()


func _add_boot_bar() -> ProgressBar:
	var bar := ProgressBar.new()
	bar.min_value = 0
	bar.max_value = 1
	bar.value = 0
	bar.show_percentage = false
	bar.custom_minimum_size = Vector2(360, 4)
	bar.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	var background := StyleBoxFlat.new()
	background.bg_color = Color("#202633")
	background.corner_radius_top_left = 2
	background.corner_radius_top_right = 2
	background.corner_radius_bottom_left = 2
	background.corner_radius_bottom_right = 2
	var fill := StyleBoxFlat.new()
	fill.bg_color = Color("#44a963")
	fill.corner_radius_top_left = 2
	fill.corner_radius_top_right = 2
	fill.corner_radius_bottom_left = 2
	fill.corner_radius_bottom_right = 2
	bar.add_theme_stylebox_override("background", background)
	bar.add_theme_stylebox_override("fill", fill)
	_content.add_child(bar)
	return bar


func _settings_held() -> bool:
	for device in Input.get_connected_joypads():
		if Input.is_joy_button_pressed(device, JOY_BUTTON_A):
			return true
	# Keep the menu reachable on a keyboard-only machine.
	return Input.is_physical_key_pressed(KEY_ENTER)


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
	var version: String = _text(dolphin.get("current"), "not installed")
	return "Game: %s\nLogin: %s\nDolphin: %s" % [game, account, version]


func _cli(args: Array, read_stderr: bool = true) -> Dictionary:
	var output: Array = []
	var code := OS.execute(CLI, PackedStringArray(args), output, read_stderr)
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
	var settings: Dictionary = _status.get("settings", {})
	var current := _text(settings.get("iso_path"), "")
	if current != "":
		current = current.get_file()
	_add_button(("Play — %s" % current) if current != "" else "Play", _on_play)
	_add_button("Melee ROMs", _show_roms)
	_add_button("Dolphin", _show_dolphin)
	_add_button("Update", _show_update)
	_add_button("Account", _show_account)
	_add_button("Settings", _show_settings)
	_add_button("Quit", _show_quit)
	_status_label = _add_label(_status_text(), 0.6)
	_focus_first()


func _show_roms(message: String = "") -> void:
	_stop_net()
	_screen = "roms"
	_clear()
	_title = _add_title("MELEE ROMS", 48)
	var settings: Dictionary = _status.get("settings", {})
	var current := _text(settings.get("iso_path"), "")
	var result := _cli(["isos"])
	var found := false
	var default_entry: Dictionary = {}
	var others: Array = []
	for entry in result.get("isos", []):
		if not entry.get("valid", false):
			continue
		found = true
		if str(entry.get("path", "")) == current:
			default_entry = entry
		else:
			others.append(entry)
	if not default_entry.is_empty():
		var default_path := str(default_entry.get("path", ""))
		var default_name := str(default_entry.get("name", default_path))
		_add_button("%s  (default)" % default_name, _select_rom.bind(default_path))
	for entry in others:
		var path := str(entry.get("path", ""))
		var name := str(entry.get("name", path))
		_add_button(name, _select_rom.bind(path))
	if not found:
		_add_label("No Melee images found on MELEE-ROMS.", 0.8)
	if message != "":
		_add_label(message, 0.9)
	_add_button("Rescan", _show_roms.bind(""))
	_add_button("Back", _show_menu)
	_focus_first()


func _select_rom(path: String) -> void:
	var result := _cli(["settings", "set", "game.iso", path])
	if not result.get("ok", false):
		_show_roms(str(result.get("error", "Could not save the Melee image.")))
		return
	_refresh_status()
	_launch_game()


func _has_valid_isos() -> bool:
	var result := _cli(["isos"])
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			return true
	return false


func _check_setup() -> void:
	_screen = "prepare"
	_clear()
	_title = _add_title("SETTING UP", 48)
	_add_label("Checking the MELEE-ROMS drive…", 0.8)
	var status := _add_label("", 0.75)
	await get_tree().process_frame

	var result := _cli(["isos"])
	var count := 0
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			count += 1
	if not result.get("ok", true):
		status.text = "Could not read the MELEE-ROMS drive."
	elif count == 0:
		status.text = "No Melee images found."
	else:
		status.text = "Found %d Melee %s." % [count, "image" if count == 1 else "images"]
	await get_tree().create_timer(0.8).timeout
	status.text = "Starting setup…"
	await get_tree().create_timer(0.4).timeout
	_show_setup()


func _show_setup(message: String = "") -> void:
	_start_net("setup", message)


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
		_add_button("Sign in", _start_login)
		_add_button("Sign in on another device", _start_net.bind("login"))
	_add_button("Back", _show_menu)
	_focus_first()


func _start_login() -> void:
	_auth_email = ""
	_show_auth_field("email", "")


func _show_auth_field(stage: String, message: String = "") -> void:
	_stop_net()
	_screen = "auth"
	_input_mode = "auth"
	_auth_stage = stage
	_auth_field = ""
	_clear()
	_title = _add_title("EMAIL" if stage == "email" else "PASSWORD", 40)
	_auth_label = _add_label("", 1.0)
	if message != "":
		_add_label(message, 0.9)
	_build_keyboard(stage == "password")
	_add_button("Use browser instead", _start_net.bind("login"))
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
	if _input_mode == "display_custom":
		_apply_custom_mode(_auth_field.strip_edges())
		return
	if _input_mode == "wifi_password":
		_submit_wifi(_wifi_ssid, _auth_field)
		return
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


func _show_settings(message: String = "") -> void:
	_stop_net()
	_input_mode = ""
	_screen = "settings"
	_clear()
	_title = _add_title("SETTINGS", 44)
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var audio: Dictionary = settings.get("audio", {})
	var network: Dictionary = settings.get("network", {})
	var expert: Dictionary = settings.get("expert", {})
	var expert_on := bool(expert.get("enabled", false))
	_add_button("Display: %s" % _display_label(display), _show_display)
	_add_button("Audio: %s" % _sink_label(audio), _show_audio)
	_add_label("Network: %s" % _text(network.get("ssid"), "(none)"), 0.8)
	_add_button("Wi-Fi network", _show_wifi)
	_add_button("Expert mode: %s" % ("on" if expert_on else "off"), _toggle_expert)
	if expert_on:
		_add_button("Expert tools", _show_expert)
	if message != "":
		_add_label(message, 0.9)
	_add_button("Back", _show_menu)
	_focus_first()


func _display_label(display: Dictionary) -> String:
	var custom := _text(display.get("custom"), "")
	if custom != "":
		return "custom %s" % custom
	var resolution := _text(display.get("resolution"), "auto")
	if resolution == "" or resolution == "auto":
		return "auto"
	var rate := _text(display.get("rate"), "")
	if rate == "":
		rate = _text(display.get("refresh_hz"), "")
	if rate == "" or rate == "0":
		return resolution
	return "%s @ %s Hz" % [resolution, rate]


func _sink_label(audio: Dictionary) -> String:
	var sink := _text(audio.get("sink"), "auto")
	return "auto (system default)" if sink == "auto" else sink


func _show_audio() -> void:
	_stop_net()
	_screen = "audio"
	_clear()
	_title = _add_title("AUDIO OUTPUT", 40)
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	var current := _text(settings.get("audio", {}).get("sink"), "auto")
	_add_button("Auto (system default)%s" % ("  (selected)" if current == "auto" else ""),
		_select_sink.bind("auto"))
	var result := _cli(["audio", "sinks"])
	if not result.get("ok", false):
		_add_label(str(result.get("error", "Could not list audio outputs.")), 0.8)
	else:
		var sinks: Array = result.get("sinks", [])
		if sinks.is_empty():
			_add_label("No audio outputs found.", 0.8)
		for sink in sinks:
			var name := str(sink.get("name", ""))
			if name == "":
				continue
			var label := _text(sink.get("description"), name)
			if name == current:
				label += "  (selected)"
			_add_button(label, _select_sink.bind(name))
	_add_button("Back", _show_settings.bind(""))
	_focus_first()


func _select_sink(name: String) -> void:
	var saved := _cli(["settings", "set", "audio.sink", name])
	if not saved.get("ok", false):
		_show_audio()
		return
	var applied := _cli(["display", "apply"])
	if not applied.get("ok", false):
		_show_settings("Audio saved, but applying it failed: %s" %
			str(applied.get("error", "unknown error")))
		return
	_show_settings("Audio output set.")


func _show_display(message: String = "") -> void:
	_stop_net()
	_input_mode = ""
	_screen = "display"
	_clear()
	_title = _add_title("DISPLAY", 44)
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var expert := bool(settings.get("expert", {}).get("enabled", false))
	var result := _cli(["display", "modes"])
	if not result.get("ok", false):
		_add_label(str(result.get("error", "Could not read the display modes.")), 0.9)
		_add_button("Back", _show_settings.bind(""))
		_focus_first()
		return

	var name := _text(result.get("edid_name"), "")
	var output := _text(result.get("output"), "")
	var heading := output
	if name != "":
		heading = "%s (%s)" % [output, name]
	_add_label(heading, 0.65)
	if message != "":
		_add_label(message, 0.9)

	var current_res := _text(display.get("resolution"), "auto")
	var current_rate := _text(display.get("rate"), "")
	_add_button("Auto (native)%s" % ("  (current)" if current_res == "auto" and
		_text(display.get("custom"), "") == "" else ""), _select_mode.bind("auto", ""))

	var shown := 0
	for mode in result.get("modes", []):
		var resolution := str(mode.get("resolution", ""))
		if resolution == "":
			continue
		for rate in mode.get("rates", []):
			if shown >= MAX_MODE_ROWS:
				break
			var rate_text := _rate_text(rate)
			var selected := resolution == current_res and _rates_match(rate_text, current_rate)
			var label := "%s @ %s Hz" % [resolution, rate_text]
			if selected:
				label += "  (current)"
			_add_button(label, _select_mode.bind(resolution, rate_text))
			shown += 1
		if shown >= MAX_MODE_ROWS:
			break
	if shown == 0:
		_add_label("No modes reported by the display.", 0.8)

	if expert:
		_add_button("Custom mode…", _show_custom_mode.bind(""))
	_add_button("Back", _show_settings.bind(""))
	_focus_first()


func _rate_text(rate: Variant) -> String:
	var value := float(rate)
	if absf(value - roundf(value)) < 0.02:
		return "%d" % int(roundf(value))
	return "%.2f" % value


func _rates_match(left: String, right: String) -> bool:
	if right == "":
		return false
	return absf(float(left) - float(right)) < 0.05


func _select_mode(resolution: String, rate: String) -> void:
	if resolution == "auto":
		_cli(["settings", "set", "display.resolution", "auto"])
		_cli(["settings", "set", "display.rate", ""])
		_cli(["settings", "set", "display.custom", ""])
		_show_settings("Display set to auto.")
		return
	var result := _cli(["display", "mode", resolution, rate])
	if not result.get("ok", false):
		_show_display("Could not switch mode: %s" %
			str(result.get("error", "unknown error")))
		return
	_show_settings("Display set to %s @ %s Hz." % [resolution, rate])


func _show_custom_mode(message: String = "") -> void:
	_stop_net()
	_screen = "display-custom"
	_input_mode = "display_custom"
	_auth_field = ""
	_clear()
	_title = _add_title("CUSTOM MODE", 40)
	_add_label("Enter WIDTHxHEIGHT or WIDTHxHEIGHT@HZ, then Done.", 0.8)
	if message != "":
		_add_label(message, 0.9)
	_auth_label = _add_label("", 1.0)
	_build_keyboard(false)
	_add_button("Back", _show_display.bind(""))
	_focus_first()


func _apply_custom_mode(spec: String) -> void:
	if not spec.contains("x"):
		_show_custom_mode("Use WIDTHxHEIGHT, for example 1920x1080.")
		return
	var result := _cli(["display", "custom", spec])
	if not result.get("ok", false):
		_show_custom_mode(str(result.get("error", "The display rejected that mode.")))
		return
	_show_settings("Custom mode applied: %s." % spec)


func _toggle_expert() -> void:
	var enabled := _expert_enabled()
	var saved := _cli(["settings", "set", "expert.enabled", "false" if enabled else "true"])
	if not saved.get("ok", false):
		_show_settings(str(saved.get("error", "Could not save expert mode.")))
		return
	_show_settings("Expert mode %s." % ("disabled" if enabled else "enabled"))


func _expert_enabled() -> bool:
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	return bool(settings.get("expert", {}).get("enabled", false))


func _show_expert() -> void:
	_stop_net()
	_screen = "expert"
	_clear()
	_title = _add_title("EXPERT TOOLS", 40)
	_add_label("These tools can change the system. Use them with care.", 0.7)
	_add_button("Dolphin settings window", _open_dolphin_settings)
	_add_button("Terminal", _open_terminal)
	_add_button("Text editor", _open_editor)
	_add_button("Custom display mode", _show_custom_mode.bind(""))
	_add_button("Back", _show_settings.bind(""))
	_focus_first()


func _open_terminal() -> void:
	_clear()
	_title = _add_title("TERMINAL", 44)
	_add_label("Opening a terminal…", 0.8)
	await get_tree().process_frame
	get_tree().quit(TERMINAL_EXIT)


func _open_editor() -> void:
	_clear()
	_title = _add_title("TEXT EDITOR", 44)
	_add_label("Opening settings.toml…", 0.8)
	await get_tree().process_frame
	get_tree().quit(EDITOR_EXIT)


func _show_dolphin(message: String = "") -> void:
	_stop_net()
	_input_mode = ""
	_screen = "dolphin"
	_clear()
	_title = _add_title("DOLPHIN", 44)
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	if message != "":
		_add_label(message, 0.9)
	_add_button("Graphics: %s" % _text(dolphin.get("backend"), "Vulkan"),
		_cycle_dolphin.bind("backend", ["Vulkan", "OGL"]))
	_add_button("Resolution: %s" % _internal_resolution_label(dolphin),
		_cycle_dolphin.bind("resolution", [1, 2, 3, 4, 5, 6]))
	_add_button("Aspect: %s" % _aspect_label(dolphin),
		_cycle_dolphin.bind("aspect", [3, 1, 2, 4]))
	_add_button("VSync: %s" % ("on" if dolphin.get("vsync", false) else "off"),
		_toggle_dolphin.bind("vsync"))
	_add_button("Anti-aliasing: %s" % _msaa_label(dolphin),
		_cycle_dolphin.bind("msaa", [1, 2, 4, 8]))
	_add_button("Audio: %s" % _text(dolphin.get("audio_backend"), "Cubeb"),
		_cycle_dolphin.bind("audio_backend", ["Cubeb", "ALSA", "NoSound"]))
	_add_button("Controller: %s" % _controller_label(dolphin),
		_cycle_dolphin.bind("controller", ["adapter", "pad", "keyboard"]))
	_add_button("Reset Dolphin config", _reset_dolphin)
	_add_button("Back", _show_menu)
	_focus_first()


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


func _controller_label(dolphin: Dictionary) -> String:
	match _text(dolphin.get("controller"), "adapter"):
		"pad":
			return "standard pad"
		"keyboard":
			return "keyboard"
		_:
			return "GameCube adapter"


func _cycle_dolphin(key: String, choices: Array) -> void:
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
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
	var saved := _cli(["settings", "set", "dolphin." + key, str(next)])
	if not saved.get("ok", false):
		_show_dolphin(str(saved.get("error", "Could not save the Dolphin setting.")))
		return
	_show_dolphin()


func _toggle_dolphin(key: String) -> void:
	var settings: Dictionary = _cli(["settings", "show"]).get("settings", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var next := not bool(dolphin.get(key, false))
	var saved := _cli(["settings", "set", "dolphin." + key, "true" if next else "false"])
	if not saved.get("ok", false):
		_show_dolphin(str(saved.get("error", "Could not save the Dolphin setting.")))
		return
	_show_dolphin()


func _reset_dolphin() -> void:
	var result := _cli(["dolphin", "reset"])
	if not result.get("ok", false):
		_show_dolphin(str(result.get("error", "Could not reset Dolphin.")))
		return
	_show_dolphin("Dolphin configuration reset.")


func _show_wifi(message: String = "") -> void:
	_stop_net()
	_screen = "wifi"
	_input_mode = ""
	_clear()
	_title = _add_title("WI-FI", 44)

	var status := _cli(["wifi", "status"])
	if status.get("ok", false) and status.get("connected", false):
		_add_label("Connected to %s" % str(status.get("ssid")), 0.8)
	else:
		_add_label("Not connected.", 0.8)
	if message != "":
		_add_label(message, 0.9)

	var result := _cli(["wifi", "scan"])
	if not result.get("ok", false):
		_add_label(str(result.get("error", "Could not scan for networks.")), 0.9)
	else:
		var networks: Array = result.get("networks", [])
		if networks.is_empty():
			_add_label("No networks found.", 0.7)
		var shown := 0
		for network in networks:
			if shown >= 8:
				break
			var ssid := str(network.get("ssid", ""))
			var security := str(network.get("security", ""))
			var label := ssid
			if security != "":
				label += "  (%s)" % security
			_add_button(label, _select_wifi.bind(ssid, security))
			shown += 1

	_add_button("Rescan", _show_wifi.bind(""))
	_add_button("Back", _show_settings)
	_focus_first()


func _select_wifi(ssid: String, security: String) -> void:
	if security == "" or security == "--":
		_submit_wifi(ssid, "")
	else:
		_show_wifi_password(ssid)


func _show_wifi_password(ssid: String, message: String = "") -> void:
	_clear()
	_screen = "wifi-password"
	_input_mode = "wifi_password"
	_wifi_ssid = ssid
	_auth_field = ""
	_title = _add_title("WI-FI PASSWORD", 40)
	_add_label(ssid, 0.8)
	if message != "":
		_add_label(message, 0.9)
	_auth_label = _add_label("", 1.0)
	_build_keyboard(true)
	_add_button("Back", _show_wifi.bind(""))
	_focus_first()


func _submit_wifi(ssid: String, password: String) -> void:
	var result := _cli(["wifi", "connect", "--ssid", ssid, "--password", password])
	if result.get("ok", false):
		_refresh_status()
		_show_wifi("Connected to %s" % ssid)
	elif password == "":
		_show_wifi("Could not connect: %s" % str(result.get("error", "unknown error")))
	else:
		_show_wifi_password(ssid, str(result.get("error", "Could not connect.")))


func _show_update() -> void:
	_stop_net()
	_screen = "update"
	_clear()
	_title = _add_title("UPDATE", 44)

	var os_check := _cli(["os", "check"])
	var os_status := _cli(["os", "status"])
	var dolphin := _cli(["update", "--check"])

	if os_check.get("ok", false):
		var current := _text(os_check.get("current"), "unknown")
		var latest := _text(os_check.get("latest"), "unknown")
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
		_add_label("Dolphin: %s" % _text(dolphin.get("current"), "not installed"), 0.65)
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
	if _needs_setup():
		if _has_valid_isos():
			_show_roms("Choose a Melee image to play.")
		else:
			_show_setup("No valid game image yet.")
		return
	_launch_game()


func _launch_game() -> void:
	if _busy:
		return
	if _needs_setup():
		_show_setup("No valid game image yet.")
		return
	_busy = true
	_clear()
	_title = _add_title("LAUNCHING", 44)
	_add_label("Preparing Slippi Dolphin…", 0.8)
	await get_tree().process_frame

	# Fetch/verify Dolphin while this screen is still visible. The wrapper
	# starts the game after Godot exits, releasing the adapter for Dolphin.
	# Keep stderr out of the captured output: download progress is not JSON.
	var result := _cli(["update"], false)
	if not result.get("ok", false):
		_busy = false
		_clear()
		_title = _add_title("COULD NOT START", 44)
		_add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		_add_button("Back", _show_menu)
		_focus_first()
		return
	get_tree().quit(GAME_EXIT)


func _open_dolphin_settings() -> void:
	_busy = true
	_clear()
	_title = _add_title("LAUNCHING", 44)
	_add_label("Preparing Slippi Dolphin…", 0.8)
	await get_tree().process_frame

	# Same hand-off as Play: make sure Dolphin is installed, then let the
	# wrapper start its settings window after Godot releases the adapter.
	var result := _cli(["update"], false)
	if not result.get("ok", false):
		_busy = false
		_clear()
		_title = _add_title("COULD NOT START", 44)
		_add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		_add_button("Back", _show_menu)
		_focus_first()
		return
	get_tree().quit(DOLPHIN_EXIT)


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


func _update_process() -> void:
	set_process(_net_pid != -1 or _os_pid != -1 or _os_waiting)


func _local_ip() -> String:
	for address in IP.get_local_addresses():
		if address.contains(".") and not address.begins_with("127.") and not address.begins_with("169.254."):
			return address
	return "127.0.0.1"


func _data_dir() -> String:
	return OS.get_environment("SLIPPIOS_DATA") if OS.get_environment("SLIPPIOS_DATA") != "" else "/data"


func _start_net(mode: String, message: String = "") -> void:
	_stop_net()
	var system_dir := DirAccess.open(_data_dir() + "/system")
	if system_dir != null:
		system_dir.remove("net-upload.json")
		system_dir.remove("net-status.json")
	var token := Crypto.new().generate_random_bytes(8).hex_encode()
	_net_url = "http://%s:8080/%s/" % [_local_ip(), token]
	_net_mode = mode
	_net_timer = 0.0
	_net_pid = OS.create_process("slippios", PackedStringArray([
		"net", "--token", token, "--port", "8080", "--mode", mode,
	]))
	_update_process()
	_show_net_screen(mode, message)


func _stop_net() -> void:
	if _net_pid != -1:
		OS.kill(_net_pid)
		_net_pid = -1
	_update_process()


func _show_net_screen(mode: String, message: String = "") -> void:
	_screen = "net"
	_net_mode = mode
	_clear()
	if mode == "setup":
		_title = _add_title("SET UP ON ANOTHER DEVICE", 34)
		_add_label("Scan with a phone, or open the address on any computer.", 0.75)
	elif mode == "iso":
		_title = _add_title("ADD A GAME", 44)
		_add_label("Open this address on a phone or computer:", 0.8)
	else:
		_title = _add_title("SIGN IN ON ANOTHER DEVICE", 34)
		_add_label("Open this address on a phone or computer:", 0.8)

	_add_label(_net_url, 1.0)
	var qr := _make_qr(_net_url)
	if qr != null:
		_content.add_child(qr)
	if mode == "setup":
		_add_label("Step 1: add Melee. Step 2: sign in or skip.", 0.65)
		_net_status = _add_label(message if message != "" else "Waiting for setup…", 0.75)
		_add_button("Restart setup page", _show_setup)
	else:
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
	if mode == "login":
		_show_account()
	elif mode == "setup":
		_show_menu()
	else:
		_show_setup()


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
		var status := _read_json_file(_data_dir() + "/system/net-status.json")
		if is_instance_valid(_net_status):
			if not status.get("ok", true):
				_net_status.text = "Network page stopped: %s" % str(status.get("error", "unknown error"))
			else:
				_net_status.text = "Network page stopped."
		return

	if _net_mode == "setup":
		var setup := _read_json_file(_data_dir() + "/system/setup.json")
		if setup.get("done", false):
			_stop_net()
			_refresh_status()
			if _needs_setup():
				_show_setup("Setup finished, but the console profile is incomplete.")
				return
			_launch_game()
			return
		if is_instance_valid(_net_status):
			var problem := str(setup.get("error", ""))
			if problem != "":
				_net_status.text = problem
			elif not setup.get("game_ready", false):
				_net_status.text = "Waiting for game selection…"
			elif not setup.get("account_ready", false):
				_net_status.text = "Game added. Finish sign-in or skip in the browser."
			else:
				_net_status.text = "Finishing setup…"
		return

	if _net_mode == "iso":
		var marker := _read_json_file(_data_dir() + "/system/net-upload.json")
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
	_show_account()


func _unhandled_input(event: InputEvent) -> void:
	if not event.is_action_pressed("ui_cancel"):
		return
	match _screen:
		"net":
			_net_back()
		"auth":
			_back_from_auth()
		"wifi", "audio", "expert":
			_show_settings()
		"dolphin":
			_show_menu()
		"roms":
			_show_menu()
		"wifi-password":
			_show_wifi("")
		"display":
			_show_settings()
		"display-custom":
			_show_display("")
		"login-setup", "setup", "settings", "update", "account", "quit":
			_stop_net()
			_show_menu()


func _show_quit() -> void:
	_stop_net()
	_screen = "quit"
	_clear()
	_title = _add_title("POWER OFF", 44)
	_add_label("Power off the console?", 0.8)
	_add_button("Power off", _poweroff)
	_add_button("Cancel", _show_menu)
	_focus_first()


func _poweroff() -> void:
	_clear()
	_title = _add_title("POWERING OFF", 44)
	var status := _add_label("You can unplug the drive when the screen goes dark.", 0.8)
	await get_tree().process_frame
	var result := _cli(["poweroff"])
	if not result.get("ok", false):
		status.text = str(result.get("error", "Could not power off."))
		_add_button("Back", _show_menu)
		_focus_first()
