extends Control

## SlippiOS shell app: boot flow, screen routing, and system actions. Screens
## build the actual pages; services own the CLI and background processes.

const Cli := preload("res://services/cli.gd")
const NetPage := preload("res://services/net_page.gd")
const UpdateJob := preload("res://services/update_job.gd")
const Ui := preload("res://services/ui.gd")
const ShellStatus := preload("res://services/status.gd")
const Actions := preload("res://services/actions.gd")

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
var ui = Ui.new()
var shell_status
var actions

var _screen_node = null
var _busy := false


func _ready() -> void:
	shell_status = ShellStatus.new(self)
	actions = Actions.new(self)
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
	ui.content = content


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
	await actions.launch_game()


func open_dolphin_settings() -> void:
	await actions.open_dolphin_settings()


func open_terminal() -> void:
	await actions.open_terminal()


func open_editor() -> void:
	await actions.open_editor()


func connect_wifi(ssid: String, password: String) -> void:
	actions.connect_wifi(ssid, password)


func refresh_status() -> void:
	shell_status.refresh()


func needs_setup() -> bool:
	return shell_status.needs_setup()


func has_valid_isos() -> bool:
	return shell_status.has_valid_isos()


func expert_enabled() -> bool:
	return shell_status.expert_enabled()


func status_text() -> String:
	return shell_status.text()


func settings_held() -> bool:
	for device in Input.get_connected_joypads():
		if Input.is_joy_button_pressed(device, JOY_BUTTON_A):
			return true
	# Keep the menu reachable on a keyboard-only machine.
	return Input.is_physical_key_pressed(KEY_ENTER)


func cli(args: Array, read_stderr: bool = true) -> Dictionary:
	return Cli.run(args, read_stderr)


func text_value(value: Variant, fallback: String) -> String:
	return ui.text_value(value, fallback)


func clear() -> void:
	ui.clear()


func add_title(text: String, size: int = 56) -> Label:
	return ui.add_title(text, size)


func add_label(text: String, alpha: float = 0.65) -> Label:
	return ui.add_label(text, alpha)


func add_button(text: String, callback: Callable, height: int = 50) -> Button:
	return ui.add_button(text, callback, height)


func add_content(node: Control) -> void:
	ui.add_content(node)


func focus_first() -> void:
	ui.focus_first()
