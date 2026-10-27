extends "res://screens/screen.gd"

const MAX_MODE_ROWS := 24


func present(message: String = "") -> void:
	clear()
	title("DISPLAY", 44)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var display: Dictionary = settings.get("display", {})
	var expert := bool(settings.get("expert", {}).get("enabled", false))
	var result := cli(["display", "modes"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the display modes.")), 0.9)
		button("Back", app.show_settings.bind(""))
		focus()
		return

	var name := text_value(result.get("edid_name"), "")
	var output := text_value(result.get("output"), "")
	var heading := output
	if name != "":
		heading = "%s (%s)" % [output, name]
	label(heading, 0.65)
	if message != "":
		label(message, 0.9)

	var current_res := text_value(display.get("resolution"), "auto")
	var current_rate := text_value(display.get("rate"), "")
	button("Auto (native)%s" % ("  (current)" if current_res == "auto" and
		text_value(display.get("custom"), "") == "" else ""), _select.bind("auto", ""))

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
			var mode_label := "%s @ %s Hz" % [resolution, rate_text]
			if selected:
				mode_label += "  (current)"
			button(mode_label, _select.bind(resolution, rate_text))
			shown += 1
		if shown >= MAX_MODE_ROWS:
			break
	if shown == 0:
		label("No modes reported by the display.", 0.8)

	if expert:
		button("Custom mode…", app.show_custom_mode.bind(""))
	button("Back", app.show_settings.bind(""))
	focus()


func _rate_text(rate: Variant) -> String:
	var value := float(rate)
	if absf(value - roundf(value)) < 0.02:
		return "%d" % int(roundf(value))
	return "%.2f" % value


func _rates_match(left: String, right: String) -> bool:
	if right == "":
		return false
	return absf(float(left) - float(right)) < 0.05


func _select(resolution: String, rate: String) -> void:
	if resolution == "auto":
		cli(["settings", "set", "display.resolution", "auto"])
		cli(["settings", "set", "display.rate", ""])
		cli(["settings", "set", "display.custom", ""])
		app.show_settings("Display set to auto.")
		return
	var result := cli(["display", "mode", resolution, rate])
	if not result.get("ok", false):
		app.show_display("Could not switch mode: %s" %
			str(result.get("error", "unknown error")))
		return
	app.restart_shell()


func back() -> void:
	app.show_settings()
