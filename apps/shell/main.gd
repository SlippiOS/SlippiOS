extends Control

const CLI := "slippios-ctl"

var _status: Dictionary = {}
var _play_pid := -1
var _busy := false
var _menu: VBoxContainer
var _status_label: Label
var _title: Label


func _ready() -> void:
	DisplayServer.window_set_mode(DisplayServer.WINDOW_MODE_FULLSCREEN)
	set_process(false)
	_build()
	_refresh_status()
	_animate_boot()


func _build() -> void:
	var background := ColorRect.new()
	background.color = Color("#0b0b10")
	background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	background.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(background)

	var center := CenterContainer.new()
	center.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
	add_child(center)

	var column := VBoxContainer.new()
	column.custom_minimum_size = Vector2(560, 0)
	column.add_theme_constant_override("separation", 18)
	center.add_child(column)

	_title = Label.new()
	_title.text = "SLIPPIOS"
	_title.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_title.add_theme_font_size_override("font_size", 64)
	column.add_child(_title)

	var subtitle := Label.new()
	subtitle.text = "Super Smash Bros. Melee"
	subtitle.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	subtitle.modulate = Color(1, 1, 1, 0.55)
	column.add_child(subtitle)

	_menu = VBoxContainer.new()
	_menu.add_theme_constant_override("separation", 10)
	column.add_child(_menu)

	_add_button("Play", _on_play)
	_add_button("Update", _on_update)
	_add_button("Account", _on_account)
	_add_button("Settings", _on_settings)
	_add_button("Quit", _on_quit)

	_status_label = Label.new()
	_status_label.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
	_status_label.modulate = Color(1, 1, 1, 0.6)
	_status_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	column.add_child(_status_label)


func _add_button(text: String, callback: Callable) -> void:
	var button := Button.new()
	button.text = text
	button.custom_minimum_size = Vector2(0, 52)
	button.pressed.connect(callback)
	_menu.add_child(button)


func _animate_boot() -> void:
	_title.modulate.a = 0.0
	var tween := create_tween()
	tween.tween_property(_title, "modulate:a", 1.0, 0.8)
	tween.tween_interval(0.6)
	tween.tween_callback(func() -> void:
		var first := _menu.get_child(0) as Button
		if first != null:
			first.grab_focus()
	)


func _refresh_status() -> void:
	_status = _cli(["status"])
	var settings: Dictionary = _status.get("settings", {})
	var login: Dictionary = _status.get("login", {})
	var dolphin: Dictionary = _status.get("dolphin", {})

	var game: String = "ready" if settings.get("iso_valid", false) else str(settings.get("iso_reason", "not configured"))
	var account: String = "signed in" if login.get("signed_in", false) else "not signed in"
	var version: String = str(dolphin.get("current") or "not installed")
	_status_label.text = "Game: %s\nLogin: %s\nDolphin: %s" % [game, account, version]


func _cli(args: Array) -> Dictionary:
	var output: Array = []
	var code := OS.execute(CLI, PackedStringArray(args), output, true)
	var text := ""
	for line in output:
		text += str(line)
	var parsed: Variant = JSON.parse_string(text)
	if typeof(parsed) != TYPE_DICTIONARY:
		return {"ok": false, "error": "invalid CLI output (exit %d)" % code}
	return parsed


func _on_play() -> void:
	if _busy:
		return
	var settings: Dictionary = _status.get("settings", {})
	if not settings.get("iso_valid", false):
		_status_label.text = "No valid Melee image."
		return
	_busy = true
	_status_label.text = "Launching Dolphin…"
	_play_pid = OS.create_process(CLI, PackedStringArray(["play"]))
	set_process(true)


func _process(_delta: float) -> void:
	if _play_pid == -1:
		return
	if not OS.is_process_running(_play_pid):
		_play_pid = -1
		_busy = false
		set_process(false)
		_refresh_status()


func _on_update() -> void:
	if _busy:
		return
	_busy = true
	_status_label.text = "Checking for updates…"
	var result := _cli(["update", "--check"])
	_busy = false
	if result.get("ok", false):
		_status_label.text = "Latest: %s" % result.get("latest", "?")
	else:
		_status_label.text = "Update error: %s" % result.get("error", "?")
	_refresh_status()


func _on_account() -> void:
	_status_label.text = "Account sign-in UI is next."


func _on_settings() -> void:
	_status_label.text = "Settings UI is next."


func _on_quit() -> void:
	get_tree().quit()
