extends "res://screens/screen.gd"

func present(message: String = "") -> void:
	clear()
	title("MELEE ROMS", 48)
	var settings: Dictionary = app.status.get("settings", {})
	var current := text_value(settings.get("iso_path"), "")
	var result := cli(["isos"])
	var found := false
	var default_entry: Dictionary = {}
	var others: Array = []
	for entry in result.get("isos", []):
		if not entry.get("valid", false):
			continue
		found = true
		if str(entry.get("path", "")) == current:
			default_entry = entry
		else:
			others.append(entry)
	if not default_entry.is_empty():
		var default_path := str(default_entry.get("path", ""))
		var default_name := str(default_entry.get("name", default_path))
		button("%s  (default)" % default_name, _select.bind(default_path))
	for entry in others:
		var path := str(entry.get("path", ""))
		var name := str(entry.get("name", path))
		button(name, _select.bind(path))
	if not found:
		label("No Melee images found on MELEE-ROMS.", 0.8)
	if message != "":
		label(message, 0.9)
	button("Rescan", present.bind(""))
	button("Back", app.show_menu)
	focus()


func _select(path: String) -> void:
	var result := cli(["settings", "set", "game.iso", path])
	if not result.get("ok", false):
		present(str(result.get("error", "Could not save the Melee image.")))
		return
	app.refresh_status()
	app.launch_game()


func back() -> void:
	app.show_menu()
