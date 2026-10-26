extends Control

## SlippiOS shell app: boot flow, screen routing, and system actions. Screens
## build the actual pages; services own the CLI and background processes.

const GAME_EXIT := 42
const DOLPHIN_EXIT := 43
const TERMINAL_EXIT := 44
const EDITOR_EXIT := 45

const Cli := preload("res://services/cli.gd")
const NetPage := preload("res://services/net_page.gd")
const UpdateJob := preload("res://services/update_job.gd")

const BootScreen := preload("res://screens/boot.gd")
const MenuScreen := preload("res://screens/menu.gd")
const RomsScreen := preload("res://screens/roms.gd")
const SetupScreen := preload("res://screens/setup.gd")
const AccountScreen := preload("res://screens/account.gd")
const AuthScreen := preload("res://screens/auth.gd")
const SettingsScreen := preload("res://screens/settings.gd")
const AudioScreen := preload("res://screens/audio.gd")
const DisplayScreen := preload("res://screens/display.gd")
const CustomModeScreen := preload("res://screens/display_custom.gd")
const ExpertScreen := preload("res://screens/expert.gd")
const DolphinScreen := preload("res://screens/dolphin.gd")
const WifiScreen := preload("res://screens/wifi.gd")
const WifiPasswordScreen := preload("res://screens/wifi_password.gd")
const UpdateScreen := preload("res://screens/update.gd")
const OSUpdateScreen := preload("res://screens/os_update.gd")
const OSResultScreen := preload("res://screens/os_result.gd")
const NetScreen := preload("res://screens/net.gd")
const QuitScreen := preload("res://screens/quit.gd")

var status: Dictionary = {}
var content: VBoxContainer
var net = NetPage.new()
var update_job = UpdateJob.new()

var _screen_node = null
var _busy := false


func _ready() -> void:
	# CI only imports/parses the project; skip the CLI and display probing.
	if OS.get_environment("SLIPPIOS_PARSE_ONLY") == "1":
		return
	DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_FULLSCREEN)
	_build_base()
	net.setup_done.connect(_on_net_setup_done)
	net.upload_done.connect(_on_net_upload_done)
	net.login_done.connect(_on_net_login_done)
	update_job.finished.connect(_on_os_update_finished)
	refresh_status()
	# The graphical session does not run slippios-session, so apply the saved
	# display mode here (including re-adding a custom mode after a reboot).
	cli(["display", "apply"])
	if OS.get_environment("SLIPPIOS_SKIP_SPLASH") == "1":
		after_splash()
	else:
		show_boot_splash()


func _process(delta: float) -> void:
	net.poll(delta)
	update_job.poll(delta)


func _unhandled_input(event: InputEvent) -> void:
	if not event.is_action_pressed("ui_cancel"):
		return
	if _screen_node != null and is_instance_valid(_screen_node):
		_screen_node.back()


func _build_base() -> void:
	var background := ColorRect.new()
	background.color = SlippiTheme.background
	background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	background.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(background)

	var center := CenterContainer.new()
	center.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	add_child(center)

	content = VBoxContainer.new()
	content.custom_minimum_size = Vector2(760, 0)
	content.add_theme_constant_override("separation", 12)
	center.add_child(content)


func _present(screen, message: String = "") -> void:
	_screen_node = screen
	screen.present(message)


func show_boot_splash() -> void:
	_present(BootScreen.new(self))


func show_menu() -> void:
	stop_net()
	_present(MenuScreen.new(self))


func show_roms(message: String = "") -> void:
	stop_net()
	_present(RomsScreen.new(self), message)


func show_setup(message: String = "") -> void:
	stop_net()
	_present(SetupScreen.new(self), message)


func show_account() -> void:
	stop_net()
	_present(AccountScreen.new(self))


func show_auth_email() -> void:
	stop_net()
	_present(AuthScreen.new(self))


func show_auth_password(email: String) -> void:
	stop_net()
	var screen = AuthScreen.new(self)
	screen.stage = "password"
	screen.email = email
	_present(screen)


func show_settings(message: String = "") -> void:
	stop_net()
	_present(SettingsScreen.new(self), message)


func show_audio() -> void:
	stop_net()
	_present(AudioScreen.new(self))


func show_display(message: String = "") -> void:
	stop_net()
	_present(DisplayScreen.new(self), message)


func show_custom_mode(message: String = "") -> void:
	stop_net()
	_present(CustomModeScreen.new(self), message)


func show_expert() -> void:
	stop_net()
	_present(ExpertScreen.new(self))


func show_dolphin(message: String = "") -> void:
	stop_net()
	_present(DolphinScreen.new(self), message)


func show_wifi(message: String = "") -> void:
	stop_net()
	_present(WifiScreen.new(self), message)


func show_wifi_password(ssid: String, message: String = "") -> void:
	stop_net()
	var screen = WifiPasswordScreen.new(self)
	screen.ssid = ssid
	_present(screen, message)


func show_update() -> void:
	stop_net()
	_present(UpdateScreen.new(self))


func show_quit() -> void:
	stop_net()
	_present(QuitScreen.new(self))


func start_net(mode: String, message: String = "") -> void:
	net.start(mode)
	_present(NetScreen.new(self), message)


func stop_net() -> void:
	net.stop()


func skip_setup() -> void:
	stop_net()
	refresh_status()
	show_menu()


func start_os_update() -> void:
	stop_net()
	if update_job.active():
		_present(OSUpdateScreen.new(self))
		return
	update_job.start()
	_present(OSUpdateScreen.new(self))


func show_os_progress() -> void:
	stop_net()
	if update_job.active():
		_present(OSUpdateScreen.new(self))
	else:
		show_update()


func request_reboot() -> void:
	var result := cli(["os", "reboot"])
	if not result.get("ok", false):
		_present_os_result("reboot_failed", str(result.get("error", "Could not request a restart.")))
		return
	_present_os_result("restarting", "")


func _present_os_result(mode: String, message: String) -> void:
	var screen = OSResultScreen.new(self)
	screen.mode = mode
	screen.message = message
	_present(screen)


func _on_net_setup_done() -> void:
	refresh_status()
	if needs_setup():
		show_setup("Setup finished, but the console profile is incomplete.")
		return
	launch_game()


func _on_net_upload_done() -> void:
	refresh_status()
	show_setup("Game image added.")


func _on_net_login_done() -> void:
	refresh_status()
	show_account()


func _on_os_update_finished(phase: String, error: String) -> void:
	if phase == "installed":
		_present_os_result("installed", "")
	elif phase == "failed":
		_present_os_result("failed", error if error != "" else "The system update could not be installed.")
	else:
		show_update()


func after_splash() -> void:
	if needs_setup():
		var settings: Dictionary = status.get("settings", {})
		if settings.get("exists", false) and has_valid_isos():
			show_roms("Default Melee image is missing. Choose one.")
		else:
			show_setup()
	elif OS.get_environment("SLIPPIOS_AUTOPLAY") == "0":
		show_menu()
	else:
		launch_game()


func on_play() -> void:
	if needs_setup():
		if has_valid_isos():
			show_roms("Choose a Melee image to play.")
		else:
			show_setup("No valid game image yet.")
		return
	launch_game()


func launch_game() -> void:
	if _busy:
		return
	if needs_setup():
		show_setup("No valid game image yet.")
		return
	_busy = true
	_screen_node = null
	clear()
	add_title("LAUNCHING", 44)
	add_label("Preparing Slippi Dolphin…", 0.8)
	await get_tree().process_frame

	# Fetch/verify Dolphin while this screen is still visible. The wrapper
	# starts the game after Godot exits, releasing the adapter for Dolphin.
	# Keep stderr out of the captured output: download progress is not JSON.
	var result := cli(["update"], false)
	if not result.get("ok", false):
		_busy = false
		clear()
		add_title("COULD NOT START", 44)
		add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		add_button("Back", show_menu)
		focus_first()
		return
	get_tree().quit(GAME_EXIT)


func open_dolphin_settings() -> void:
	_busy = true
	_screen_node = null
	clear()
	add_title("LAUNCHING", 44)
	add_label("Preparing Slippi Dolphin…", 0.8)
	await get_tree().process_frame

	# Same hand-off as Play: make sure Dolphin is installed, then let the
	# wrapper start its settings window after Godot releases the adapter.
	var result := cli(["update"], false)
	if not result.get("ok", false):
		_busy = false
		clear()
		add_title("COULD NOT START", 44)
		add_label(str(result.get("error", "Could not prepare Slippi Dolphin.")), 0.9)
		add_button("Back", show_menu)
		focus_first()
		return
	get_tree().quit(DOLPHIN_EXIT)


func open_terminal() -> void:
	_screen_node = null
	clear()
	add_title("TERMINAL", 44)
	add_label("Opening a terminal…", 0.8)
	await get_tree().process_frame
	get_tree().quit(TERMINAL_EXIT)


func open_editor() -> void:
	_screen_node = null
	clear()
	add_title("TEXT EDITOR", 44)
	add_label("Opening settings.toml…", 0.8)
	await get_tree().process_frame
	get_tree().quit(EDITOR_EXIT)


func connect_wifi(ssid: String, password: String) -> void:
	var result := cli(["wifi", "connect", "--ssid", ssid, "--password", password])
	if result.get("ok", false):
		refresh_status()
		show_wifi("Connected to %s" % ssid)
	elif password == "":
		show_wifi("Could not connect: %s" % str(result.get("error", "unknown error")))
	else:
		show_wifi_password(ssid, str(result.get("error", "Could not connect.")))


func refresh_status() -> void:
	status = cli(["status"])


func needs_setup() -> bool:
	var settings: Dictionary = status.get("settings", {})
	return not settings.get("exists", false) or not settings.get("iso_valid", false)


func has_valid_isos() -> bool:
	var result := cli(["isos"])
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			return true
	return false


func expert_enabled() -> bool:
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	return bool(settings.get("expert", {}).get("enabled", false))


func status_text() -> String:
	var settings: Dictionary = status.get("settings", {})
	var login: Dictionary = status.get("login", {})
	var dolphin: Dictionary = status.get("dolphin", {})
	var game: String = "ready" if settings.get("iso_valid", false) else str(settings.get("iso_reason", "not configured"))
	var account: String = "signed in" if login.get("signed_in", false) else "not signed in"
	var version: String = text_value(dolphin.get("current"), "not installed")
	return "Game: %s\nLogin: %s\nDolphin: %s" % [game, account, version]


func settings_held() -> bool:
	for device in Input.get_connected_joypads():
		if Input.is_joy_button_pressed(device, JOY_BUTTON_A):
			return true
	# Keep the menu reachable on a keyboard-only machine.
	return Input.is_physical_key_pressed(KEY_ENTER)


func cli(args: Array, read_stderr: bool = true) -> Dictionary:
	return Cli.run(args, read_stderr)


func text_value(value: Variant, fallback: String) -> String:
	if value == null:
		return fallback
	var text := str(value)
	return fallback if text == "" else text


func clear() -> void:
	for child in content.get_children():
		content.remove_child(child)
		child.queue_free()


func add_title(text: String, size: int = 56) -> Label:
	var node := Label.new()
	node.text = text
	node.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	node.add_theme_font_size_override("font_size", size)
	content.add_child(node)
	return node


func add_label(text: String, alpha: float = 0.65) -> Label:
	var node := Label.new()
	node.text = text
	node.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	node.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	node.modulate = Color(1, 1, 1, alpha)
	content.add_child(node)
	return node


func add_button(text: String, callback: Callable, height: int = 50) -> Button:
	var node := Button.new()
	node.text = text
	node.custom_minimum_size = Vector2(0, height)
	node.pressed.connect(callback)
	content.add_child(node)
	return node


func add_content(node: Control) -> void:
	content.add_child(node)


func focus_first() -> void:
	var first := _find_button(content)
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
