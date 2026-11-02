extends "res://screens/screen.gd"

func present() -> void:
	clear()
	title("AUDIO", 44)
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var audio: Dictionary = settings.get("audio", {})
	var volume := int(audio.get("volume", 100))
	var expert := bool(settings.get("expert", {}).get("enabled", false))

	section("Volume")
	var volume_label := label("Volume: %d%%" % volume, 0.8)
	var slider := HSlider.new()
	slider.min_value = 0
	slider.max_value = 100
	slider.step = 5
	slider.value = volume
	slider.custom_minimum_size = Vector2(360, 32)
	slider.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	slider.value_changed.connect(_set_volume.bind(volume_label))
	add_content(slider)

	if expert:
		var current := text_value(audio.get("sink"), "auto")
		section("Outputs")
		button("Auto (system default)%s" % ("  (selected)" if current == "auto" else ""),
			_select.bind("auto"))
		var result := cli(["audio", "sinks"])
		if not result.get("ok", false):
			label(str(result.get("error", "Could not list audio outputs.")), 0.8)
		else:
			var sinks: Array = array_value(result.get("sinks", []))
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
	else:
		label("Output selection is available in Expert mode.", 0.6)

	button("Back", app.show_settings)
	focus()
	slider.grab_focus()


func _set_volume(value: float, volume_label: Label) -> void:
	volume_label.text = "Volume: %d%%" % int(value)
	var result := cli(["audio", "volume", str(int(value))])
	if not result.get("ok", false):
		toast(str(result.get("error", "Could not set the volume.")), "error")


func _select(name: String) -> void:
	var saved := cli(["settings", "set", "audio.sink", name])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save the audio output.")), "error")
		app.show_audio()
		return
	var applied := cli(["audio", "apply"])
	if not applied.get("ok", false):
		toast("Audio saved, but applying it failed: %s" %
			str(applied.get("error", "unknown error")), "error")
		app.show_settings()
		return
	toast("Audio output set.", "success")
	app.show_settings()


func back() -> void:
	app.show_settings()
