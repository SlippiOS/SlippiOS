extends "res://screens/screen.gd"

func present(_message: String = "") -> void:
	clear()
	title("SLIPPIOS", 64)
	label("Super Smash Bros. Melee", 0.55)
	var settings: Dictionary = app.status.get("settings", {})
	var current := text_value(settings.get("iso_path"), "")
	if current != "":
		current = current.get_file()
	button(("Play — %s" % current) if current != "" else "Play", app.on_play)
	button("Melee ROMs", app.show_roms.bind(""))
	button("Dolphin", app.show_dolphin.bind(""))
	button("Update", app.show_update)
	button("Account", app.show_account)
	button("Settings", app.show_settings.bind(""))
	button("Quit", app.show_quit)
	label(app.status_text(), 0.6)
	focus()


func back() -> void:
	pass
