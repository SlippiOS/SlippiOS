extends "res://screens/screen.gd"

## Audio output picker.

func present(_message: String = "") -> void:
	clear()
	title("AUDIO OUTPUT", 40)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var current := text_value(settings.get("audio", {}).get("sink"), "auto")
	button("Auto (system default)%s" % ("  (selected)" if current == "auto" else ""),
		_select.bind("auto"))
	var result := cli(["audio", "sinks"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not list audio outputs.")), 0.8)
	else:
		var sinks: Array = result.get("sinks", [])
		if sinks.is_empty():
			label("No audio outputs found.", 0.8)
		for sink in sinks:
			var name := str(sink.get("name", ""))
			if name == "":
				continue
			var sink_label := text_value(sink.get("description"), name)
			if name == current:
				sink_label += "  (selected)"
			button(sink_label, _select.bind(name))
	button("Back", app.show_settings.bind(""))
	focus()


func _select(name: String) -> void:
	var saved := cli(["settings", "set", "audio.sink", name])
	if not saved.get("ok", false):
		app.show_audio()
		return
	var applied := cli(["display", "apply"])
	if not applied.get("ok", false):
		app.show_settings("Audio saved, but applying it failed: %s" %
			str(applied.get("error", "unknown error")))
		return
	app.show_settings("Audio output set.")


func back() -> void:
	app.show_settings()
