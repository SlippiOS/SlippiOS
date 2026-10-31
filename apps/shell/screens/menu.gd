extends "res://screens/screen.gd"

func present() -> void:
	clear()
	add_content(_wordmark())
	label("Super Smash Bros. Melee", 0.55)
	var settings: Dictionary = app.status.get("settings", {})
	var current := text_value(settings.get("iso_path"), "")
	if current != "":
		current = current.get_file()
	button(("Play — %s" % current) if current != "" else "Play", app.on_play)

	section("Library")
	button("Melee ROMs", app.show_roms)
	button("Replays", app.show_replays)

	section("System")
	button("Account", app.show_account)
	button("Settings", app.show_settings)
	button("Quit", app.show_quit)

	label(app.status_text(), 0.6)
	focus()


func _wordmark() -> Control:
	var row := HBoxContainer.new()
	row.alignment = BoxContainer.ALIGNMENT_CENTER
	row.add_theme_constant_override("separation", 8)

	var name := Label.new()
	name.text = "Slippi"
	name.vertical_alignment = VERTICAL_ALIGNMENT_CENTER
	if SlippiTheme.font_wordmark != null:
		name.add_theme_font_override("font", SlippiTheme.font_wordmark)
	name.add_theme_font_size_override("font_size", 64)
	name.add_theme_color_override("font_color", SlippiTheme.text)
	row.add_child(name)

	var icon := TextureRect.new()
	icon.texture = load("res://assets/slippi-os-icon.png")
	icon.custom_minimum_size = Vector2(64, 64)
	icon.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	icon.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	icon.texture_filter = CanvasItem.TEXTURE_FILTER_LINEAR_WITH_MIPMAPS
	row.add_child(icon)
	return row


func back() -> void:
	pass
