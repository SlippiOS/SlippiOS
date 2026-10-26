extends "res://screens/screen.gd"

## Short setup check before the browser wizard opens.

func present(message: String = "") -> void:
	clear()
	title("SETTING UP", 48)
	label("Checking the MELEE-ROMS drive…", 0.8)
	var status := label("", 0.75)
	await app.get_tree().process_frame

	var result := cli(["isos"])
	var count := 0
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			count += 1
	if not result.get("ok", true):
		status.text = "Could not read the MELEE-ROMS drive."
	elif count == 0:
		status.text = "No Melee images found."
	else:
		status.text = "Found %d Melee %s." % [count, "image" if count == 1 else "images"]
	await app.get_tree().create_timer(0.8).timeout
	status.text = "Starting setup…"
	await app.get_tree().create_timer(0.4).timeout
	app.start_net("setup", message)


func back() -> void:
	app.show_menu()
