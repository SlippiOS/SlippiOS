extends "res://screens/screen.gd"

var _generation := 0


func present(message: String = "") -> void:
	_generation += 1
	var generation := _generation
	clear()
	title("SETTING UP", 48)
	var result := cli(["isos"])
	var count := 0
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			count += 1
	var mounted := bool(result.get("mounted", true))
	var mount_error := str(result.get("mount_error", ""))
	if not result.get("ok", true):
		label("Could not read the MELEE-ROMS drive.", 0.9)
	elif not mounted:
		var detail := "MELEE-ROMS is not mounted."
		if mount_error != "":
			detail += " " + mount_error
		label(detail, 0.9)
	elif count == 0:
		label("No Melee images found on MELEE-ROMS.", 0.8)
	else:
		label("Found %d Melee %s." % [count, "image" if count == 1 else "images"], 0.8)
	if message != "":
		label(message, 0.9)

	if app.net.local_ip() == "":
		label("Connect to Wi-Fi or Ethernet, then retry.", 0.8)
		button("Wi-Fi settings", app.show_wifi.bind(""))
		button("Retry", present.bind(""))
		button("Skip for now", app.skip_setup)
		focus()
		# Pick up Ethernet or Wi-Fi as soon as it comes up, without making the
		# user press Retry.
		for _attempt in 20:
			await app.get_tree().create_timer(1.0).timeout
			if generation != _generation or app._screen_node != self:
				return
			if app.net.local_ip() != "":
				present("Network connected.")
				return
		return

	label("Starting setup…", 0.75)
	await app.get_tree().create_timer(0.4).timeout
	app.start_net("setup", message)


func back() -> void:
	app.show_menu()
