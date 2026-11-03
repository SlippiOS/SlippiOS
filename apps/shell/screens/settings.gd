extends "res://screens/screen.gd"


const CategoryMenu := preload("res://widgets/category_menu.gd")
const CATEGORIES := ["Display & Audio", "Gameplay", "Network", "System", "Advanced"]

static var selected := 0

var _menu: CategoryMenu
var _settings: Dictionary = {}


func present() -> void:
	clear()
	title("SETTINGS", 44)
	_settings = cli(["settings", "show"]).get("settings", {})
	_menu = CategoryMenu.new(app, CATEGORIES, _build_category, app.show_menu)
	_menu.build(selected)
	_menu.focus_selected()


func _build_category(index: int, menu: CategoryMenu) -> void:
	selected = index
	match index:
		0:
			_display_audio(menu)
		1:
			_gameplay(menu)
		2:
			_network(menu)
		3:
			_system(menu)
		4:
			_advanced(menu)


func _display_audio(menu: CategoryMenu) -> void:
	var display: Dictionary = _settings.get("display", {})
	var audio: Dictionary = _settings.get("audio", {})
	menu.pane_title("Display & Audio")
	menu.pane_button("Display: %s" % _display_label(display), app.show_display)
	menu.pane_button("Audio: %d%%" % int(audio.get("volume", 100)), app.show_audio)
	menu.pane_hint("Changes apply immediately and are saved for the next boot.")


func _gameplay(menu: CategoryMenu) -> void:
	menu.pane_title("Gameplay")
	menu.pane_button("Dolphin Settings", app.show_dolphin)
	menu.pane_hint("Controllers, graphics, audio, and the build.")


func _network(menu: CategoryMenu) -> void:
	var network: Dictionary = _settings.get("network", {})
	menu.pane_title("Network")
	menu.pane_button("Wi-Fi: %s" % text_value(network.get("ssid"), "not connected"), app.show_wifi)
	menu.pane_hint("Ethernet works automatically when a cable is connected.")


func _system(menu: CategoryMenu) -> void:
	menu.pane_title("System")
	menu.pane_button("Update", app.show_update)
	menu.pane_hint("Install system updates and check the bundled Dolphin build.")


func _advanced(menu: CategoryMenu) -> void:
	var expert: Dictionary = _settings.get("expert", {})
	var expert_on := bool(expert.get("enabled", false))
	menu.pane_title("Advanced")
	menu.pane_button("Expert mode: %s" % ("on" if expert_on else "off"), _toggle_expert)
	menu.pane_hint("Unlocks the terminal, text editor, and Dolphin GUI.")
	if expert_on:
		menu.pane_button("Run setup again", _run_setup)
		menu.pane_button("Expert tools", app.show_expert)


func _display_label(display: Dictionary) -> String:
	var custom := text_value(display.get("custom"), "")
	if custom != "":
		return "custom %s" % custom
	var resolution := text_value(display.get("resolution"), "auto")
	if resolution == "" or resolution == "auto":
		return "auto"
	var rate := text_value(display.get("rate"), "")
	if rate == "":
		rate = text_value(display.get("refresh_hz"), "")
	if rate == "" or rate == "0":
		return resolution
	return "%s @ %s Hz" % [resolution, rate]


func _toggle_expert() -> void:
	var enabled: bool = app.expert_enabled()
	var saved := cli(["settings", "set", "expert.enabled", "false" if enabled else "true"])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save expert mode.")), "error")
		app.show_settings()
		return
	toast("Expert mode %s." % ("disabled" if enabled else "enabled"), "success")
	app.show_settings()


func _run_setup() -> void:
	app.show_setup()


func back() -> void:
	app.show_menu()
