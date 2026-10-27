extends "res://screens/screen.gd"

func present(message: String = "") -> void:
	clear()
	title("SETTINGS", 44)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var audio: Dictionary = settings.get("audio", {})
	var network: Dictionary = settings.get("network", {})
	var expert: Dictionary = settings.get("expert", {})
	var expert_on := bool(expert.get("enabled", false))
	button("Display: %s" % _display_label(display), app.show_display.bind(""))
	button("Audio: %s" % _sink_label(audio), app.show_audio)
	label("Network: %s" % text_value(network.get("ssid"), "(none)"), 0.8)
	button("Wi-Fi network", app.show_wifi.bind(""))
	button("Expert mode: %s" % ("on" if expert_on else "off"), _toggle_expert)
	if expert_on:
		button("Expert tools", app.show_expert)
	if message != "":
		label(message, 0.9)
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
		app.show_settings(str(saved.get("error", "Could not save expert mode.")))
		return
	app.show_settings("Expert mode %s." % ("disabled" if enabled else "enabled"))


func back() -> void:
	app.show_menu()
