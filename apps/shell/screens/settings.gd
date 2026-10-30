extends "res://screens/screen.gd"


func present() -> void:
	clear()
	title("SETTINGS", 44)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var audio: Dictionary = settings.get("audio", {})
	var network: Dictionary = settings.get("network", {})
	var dolphin: Dictionary = settings.get("dolphin", {})
	var expert: Dictionary = settings.get("expert", {})
	var expert_on := bool(expert.get("enabled", false))

	section("Display & Audio")
	button("Display: %s" % _display_label(display), app.show_display)
	button("Audio: %s" % _sink_label(audio), app.show_audio)

	section("Gameplay")
	button("Dolphin Settings: %s" % text_value(dolphin.get("build"), "stable"),
		app.show_dolphin)
	button("Controller Settings", app.show_controllers)

	section("Network")
	button("Wi-Fi: %s" % text_value(network.get("ssid"), "not connected"), app.show_wifi)

	section("System")
	button("Update", app.show_update)

	section("Advanced")
	button("Expert mode: %s" % ("on" if expert_on else "off"), _toggle_expert)
	if expert_on:
		button("Run setup again", _run_setup)
		button("Expert tools", app.show_expert)

	button("Back", app.show_menu)
	focus()


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


func _sink_label(audio: Dictionary) -> String:
	var sink := text_value(audio.get("sink"), "auto")
	return "auto (system default)" if sink == "auto" else sink


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
