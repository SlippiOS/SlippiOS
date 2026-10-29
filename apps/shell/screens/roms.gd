extends "res://screens/screen.gd"

func present(message: String = "") -> void:
	clear()
	title("MELEE ROMS", 48)
	var settings: Dictionary = app.status.get("settings", {})
	var current := text_value(settings.get("iso_path"), "")
	var result := cli(["isos"])
	var mounted := bool(result.get("mounted", true))
	var mount_error := str(result.get("mount_error", ""))
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
	if not mounted:
		var detail := "MELEE-ROMS is not mounted."
		if mount_error != "":
			detail += " " + mount_error
		label(detail, 0.8)
	elif not found:
		label("No Melee images found on MELEE-ROMS.", 0.8)
	if message != "":
		label(message, 0.9)
	button("Upload ISO", app.start_iso_upload)
	button("Rescan", present.bind(""))
	button("Back", app.show_menu)
	focus()


func _select(path: String) -> void:
	var first_setup: bool = app.needs_setup()
	var result := cli(["settings", "set", "game.iso", path])
	if not result.get("ok", false):
		present(str(result.get("error", "Could not save the Melee image.")))
		return
	app.refresh_status()
	if first_setup:
		app.show_controller_setup()
	else:
		app.launch_game()


func back() -> void:
	app.show_menu()
