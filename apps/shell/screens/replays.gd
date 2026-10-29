extends "res://screens/screen.gd"


func present(message: String = "") -> void:
	clear()
	title("REPLAYS", 44)
	if message != "":
		label(message, 0.9)

	var result := cli(["replay", "list"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the replay folder.")), 0.9)

		button("Back", app.show_menu)
		focus()
		return

	var replays: Array = array_value(result.get("replays", []))
	if replays.is_empty():
		label("No Slippi replays found yet.", 0.7)
	for item in replays:
		var name := str(item.get("name", ""))
		var path := str(item.get("path", ""))
		if name == "" or path == "":
			continue
		var modified := int(item.get("modified", 0))
		var when := ""
		if modified > 0:
			when = Time.get_datetime_string_from_unix_time(modified).replace("T", " ").substr(0, 16)
		button("%s  %s" % [name, when], app.play_replay.bind(path))

	button("Back", app.show_menu)
	focus()


func back() -> void:
	app.show_menu()
