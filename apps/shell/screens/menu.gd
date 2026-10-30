extends "res://screens/screen.gd"

func present() -> void:
	clear()
	title("SlippiOS", 64)
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


func back() -> void:
	pass
