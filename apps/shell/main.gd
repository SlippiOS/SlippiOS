extends Control


const Cli := preload("res://services/cli.gd")
const NetPage := preload("res://services/net_page.gd")
const UpdateJob := preload("res://services/update_job.gd")
const InstallJob := preload("res://services/install_job.gd")
const Ui := preload("res://services/ui.gd")
const Toasts := preload("res://services/toast.gd")
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
const ControllersScreen := preload("res://screens/controllers.gd")
const ControllerSetupScreen := preload("res://screens/controller_setup.gd")
const ControllerPortScreen := preload("res://screens/controller_port.gd")
const GeckoScreen := preload("res://screens/gecko.gd")
const GeckoCustomScreen := preload("res://screens/gecko_custom.gd")
const ReplaysScreen := preload("res://screens/replays.gd")
const WifiScreen := preload("res://screens/wifi.gd")
const WifiPasswordScreen := preload("res://screens/wifi_password.gd")
const UpdateScreen := preload("res://screens/update.gd")
const InstallScreen := preload("res://screens/install.gd")
const OSUpdateScreen := preload("res://screens/os_update.gd")
const OSResultScreen := preload("res://screens/os_result.gd")
const NetScreen := preload("res://screens/net.gd")
const QuitScreen := preload("res://screens/quit.gd")

var status: Dictionary = {}
var content: VBoxContainer
var center: CenterContainer
var net = NetPage.new()
var update_job = UpdateJob.new()
var install_job = InstallJob.new()
var ui = Ui.new()
var toasts = Toasts.new()
var shell_status
var actions

const ONBOARDED_PATH := "/data/system/onboarded"

var _screen_node = null
var _busy := false
var setup_flow := false
var onboarding := false
var _cursor_hidden := false


func _ready() -> void:
	Engine.max_fps = 60
	shell_status = ShellStatus.new(self)
	actions = Actions.new(self)
	# CI only imports/parses the project; skip the CLI and display probing.
	if OS.get_environment("SLIPPIOS_PARSE_ONLY") == "1":
		return
	_ensure_gamepad_actions()
	for device in Input.get_connected_joypads():
		printerr("joypad %d: %s" % [device, Input.get_joy_name(device)])
	if not Input.get_connected_joypads().is_empty():
		_set_cursor_hidden(true)
	Input.joy_connection_changed.connect(_on_joy_connection_changed)
	_fit_window()
	_build_base()
	net.setup_done.connect(_on_net_setup_done)
	net.upload_done.connect(_on_net_upload_done)
	net.login_done.connect(_on_net_login_done)
	update_job.finished.connect(_on_os_update_finished)
	refresh_status()
	_auto_controllers()
	# The graphical session, not the terminal session, applies the saved mode.
	await _apply_display()
	if OS.get_environment("SLIPPIOS_SKIP_SPLASH") == "1":
		await after_splash()
	else:
		show_boot_splash()


func _process(delta: float) -> void:
	net.poll(delta)
	update_job.poll(delta)
	install_job.poll(delta)


func _ensure_gamepad_actions() -> void:
	_add_joy_button("ui_accept", JOY_BUTTON_A)
	_add_joy_button("ui_cancel", JOY_BUTTON_B)
	_add_joy_button("ui_up", JOY_BUTTON_DPAD_UP)
	_add_joy_button("ui_down", JOY_BUTTON_DPAD_DOWN)
	_add_joy_button("ui_left", JOY_BUTTON_DPAD_LEFT)
	_add_joy_button("ui_right", JOY_BUTTON_DPAD_RIGHT)


func _add_joy_button(action: String, button: int) -> void:
	if not InputMap.has_action(action):
		return
	for existing in InputMap.action_get_events(action):
		if existing is InputEventJoypadButton and existing.button_index == button:
			return
	var event := InputEventJoypadButton.new()
	event.device = -1
	event.button_index = button
	InputMap.action_add_event(action, event)


func _on_joy_connection_changed(device: int, connected: bool) -> void:
	printerr("joypad %d connected=%s" % [device, connected])


func _input(event: InputEvent) -> void:
	if event is InputEventJoypadButton:
		_set_cursor_hidden(true)
	elif event is InputEventJoypadMotion:
		if absf(event.axis_value) >= 0.25:
			_set_cursor_hidden(true)
	elif event is InputEventMouseMotion:
		if event.relative.length() >= 2.0:
			_set_cursor_hidden(false)
	elif event is InputEventMouseButton:
		_set_cursor_hidden(false)


func _set_cursor_hidden(hidden: bool) -> void:
	if _cursor_hidden == hidden:
		return
	_cursor_hidden = hidden
	Input.set_mouse_mode(Input.MOUSE_MODE_HIDDEN if hidden else Input.MOUSE_MODE_VISIBLE)


func _unhandled_input(event: InputEvent) -> void:
	if not event.is_action_pressed("ui_cancel"):
		return
	if actions != null and actions.launching():
		actions.cancel_launch()
		return
	if _screen_node != null and is_instance_valid(_screen_node):
		_screen_node.back()


func _build_base() -> void:
	var background := ColorRect.new()
	background.color = SlippiTheme.background
	background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	background.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(background)

	var margin := MarginContainer.new()
	margin.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	margin.add_theme_constant_override("margin_left", 24)
	margin.add_theme_constant_override("margin_right", 24)
	margin.add_theme_constant_override("margin_top", 16)
	margin.add_theme_constant_override("margin_bottom", 16)
	add_child(margin)

	var scroll := ScrollContainer.new()
	scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
	scroll.vertical_scroll_mode = ScrollContainer.SCROLL_MODE_AUTO
	scroll.follow_focus = true
	scroll.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	scroll.size_flags_vertical = Control.SIZE_EXPAND_FILL
	margin.add_child(scroll)

	center = CenterContainer.new()
	center.size_flags_horizontal = Control.SIZE_EXPAND_FILL
	scroll.add_child(center)

	content = VBoxContainer.new()
	content.add_theme_constant_override("separation", 12)
	center.add_child(content)
	ui.content = content
	ui.scroll = scroll
	toasts.attach(self)
	_resize_content()
	get_viewport().size_changed.connect(_resize_content)


func _apply_display() -> void:
	cli(["display", "apply"])
	await get_tree().create_timer(0.25).timeout
	_fit_window()
	_resize_content()
	await get_tree().process_frame


func _fit_window() -> void:
	var screen := DisplayServer.window_get_current_screen()
	var size := DisplayServer.screen_get_size(screen)
	DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_WINDOWED)
	DisplayServer.window_set_flag(DisplayServer.WINDOW_FLAG_BORDERLESS, true)
	DisplayServer.window_set_position(Vector2i(0, 0))
	DisplayServer.window_set_size(size)


func _resize_content() -> void:
	if content == null:
		return
	var available := get_viewport_rect().size.x - 64.0
	content.custom_minimum_size.x = clampf(available, 0.0, 760.0)


func _present(screen) -> void:
	var same_screen: bool = _screen_node != null and is_instance_valid(_screen_node) and \
		_screen_node.get_script() == screen.get_script()
	ui.preserve_focus = same_screen
	_screen_node = screen
	center_content_vertically(false)
	screen.present()
	ui.preserve_focus = true


func center_content_vertically(enabled: bool) -> void:
	if center == null:
		return
	center.size_flags_vertical = Control.SIZE_EXPAND_FILL if enabled else Control.SIZE_FILL


func show_boot_splash() -> void:
	_present(BootScreen.new(self))


func show_menu() -> void:
	stop_net()
	setup_flow = false
	_present(MenuScreen.new(self))


func show_roms() -> void:
	stop_net()
	_present(RomsScreen.new(self))


func show_setup() -> void:
	stop_net()
	setup_flow = true
	_present(SetupScreen.new(self))


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


func show_settings() -> void:
	stop_net()
	_present(SettingsScreen.new(self))


func show_audio() -> void:
	stop_net()
	_present(AudioScreen.new(self))


func show_display() -> void:
	stop_net()
	_present(DisplayScreen.new(self))


func show_custom_mode() -> void:
	stop_net()
	_present(CustomModeScreen.new(self))


func show_expert() -> void:
	stop_net()
	_present(ExpertScreen.new(self))


func show_dolphin() -> void:
	stop_net()
	_present(DolphinScreen.new(self))


func show_controllers() -> void:
	stop_net()
	_present(ControllersScreen.new(self))


func show_controller_setup(launch_after: bool = false) -> void:
	stop_net()
	var screen = ControllerSetupScreen.new(self)
	screen.launch_after = launch_after
	_present(screen)


func show_controller_port(port: int) -> void:
	stop_net()
	var screen = ControllerPortScreen.new(self)
	screen.port = port
	_present(screen)


func show_gecko() -> void:
	stop_net()
	_present(GeckoScreen.new(self))


func show_gecko_custom() -> void:
	stop_net()
	_present(GeckoCustomScreen.new(self))


func show_replays() -> void:
	stop_net()
	_present(ReplaysScreen.new(self))


func play_replay(path: String) -> void:
	await actions.play_replay(path)


func show_wifi() -> void:
	stop_net()
	_present(WifiScreen.new(self))


func show_wifi_password(ssid: String) -> void:
	stop_net()
	var screen = WifiPasswordScreen.new(self)
	screen.ssid = ssid
	_present(screen)


func show_update() -> void:
	stop_net()
	_present(UpdateScreen.new(self))


func show_quit() -> void:
	stop_net()
	_present(QuitScreen.new(self))


func start_net(mode: String) -> void:
	if net.local_ip() == "":
		toast("Connect to Wi-Fi or Ethernet, then retry.")
		show_wifi()
		return
	net.start(mode)
	_present(NetScreen.new(self))


func start_iso_upload() -> void:
	start_net("iso")


func stop_net() -> void:
	net.stop()


func skip_setup() -> void:
	stop_net()
	mark_onboarded()
	onboarding = false
	setup_flow = false
	refresh_status()
	if shell_status.apps_installed():
		show_menu()
	else:
		start_app_install()


func onboarded() -> bool:
	return FileAccess.file_exists(ONBOARDED_PATH)


func mark_onboarded() -> void:
	var file := FileAccess.open(ONBOARDED_PATH, FileAccess.WRITE)
	if file != null:
		file.store_string("1\n")
		file.close()


func start_onboarding() -> void:
	onboarding = true
	setup_flow = true
	if not needs_setup():
		# A fresh flash can already have a valid ISO. Go straight to the
		# controller and account steps so sign-in is still offered.
		show_controller_setup(false)
		return
	var settings: Dictionary = status.get("settings", {})
	if settings.get("exists", false) and has_valid_isos():
		toast("Choose the Melee image for this console.")
		show_roms()
	elif not net.wired_present():
		toast("No Ethernet interface found. Connect Wi-Fi to continue setup.")
		show_wifi()
	else:
		show_setup()


func finish_onboarding() -> void:
	mark_onboarded()
	onboarding = false
	setup_flow = false
	refresh_status()
	show_menu()


func start_app_install() -> void:
	stop_net()
	_present(InstallScreen.new(self))


func finish_app_install() -> void:
	refresh_status()
	finish_onboarding()


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
	if shell_status.apps_installed():
		finish_onboarding()
	else:
		start_app_install()


func _on_net_upload_done() -> void:
	refresh_status()
	toast("Game image added.", "success")
	show_roms()


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
	if not onboarded():
		var login: Dictionary = status.get("login", {})
		if login.get("signed_in", false) and not needs_setup():
			# Updating an existing console should not repeat onboarding.
			mark_onboarded()
		else:
			start_onboarding()
			return
	if needs_setup():
		var settings: Dictionary = status.get("settings", {})
		if settings.get("exists", false) and has_valid_isos():
			toast("Default Melee image is missing. Choose one.")
			show_roms()
		elif not net.wired_present():
			setup_flow = true
			toast("No Ethernet interface found. Connect Wi-Fi to continue setup.")
			show_wifi()
		else:
			# A wired interface exists. Keep the combined setup screen instead
			# of forcing the Wi-Fi-only page while DHCP is still running.
			show_setup()
	elif not shell_status.apps_installed():
		start_app_install()
	elif OS.get_environment("SLIPPIOS_AUTOPLAY") == "0":
		show_menu()
	else:
		launch_game()


func on_play() -> void:
	if needs_setup():
		if has_valid_isos():
			toast("Choose a Melee image to play.")
			show_roms()
		else:
			toast("No valid game image yet.")
			show_setup()
		return
	if not shell_status.apps_installed():
		start_app_install()
		return
	launch_game()


func launch_game() -> void:
	await actions.launch_game()


func restart_shell() -> void:
	get_tree().quit(46)


func quit_shell() -> void:
	get_tree().quit(0)


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


func _auto_controllers() -> void:
	var result := cli(["controllers", "auto"])
	if bool(result.get("changed", false)):
		refresh_status()


func adapter_present() -> bool:
	return bool(status.get("adapter", {}).get("present", false))


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


func add_section(text: String) -> void:
	ui.add_section(text)


func add_button(text: String, callback: Callable, height: int = 46) -> Button:
	return ui.add_button(text, callback, height)


func toast(text: String, kind: String = "info") -> void:
	toasts.show(text, kind)


func add_content(node: Control) -> void:
	ui.add_content(node)


func focus_first() -> void:
	ui.focus_first()
