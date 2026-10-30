extends "res://screens/screen.gd"

var _generation := 0


func present() -> void:
	_generation += 1
	var generation := _generation
	clear()
	title("SETTING UP", 48)
	var result := cli(["isos"])
	var count := 0
	var first_iso := ""
	var first_name := ""
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			count += 1
			if first_iso == "":
				first_iso = str(entry.get("path", ""))
				first_name = str(entry.get("name", first_iso))
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

	if first_iso != "":
		button("Use %s" % first_name, _use_iso.bind(first_iso))
		# Wait for the user to choose the image. The button continues to the
		# controller and account steps.
		focus()
		return

	if app.net.local_ip() == "":
		if app.net.wired_link():
			label("Ethernet detected; waiting for an address…", 0.8)
		elif app.net.wired_present():
			label("Ethernet cable not detected. Plug it in, or connect Wi-Fi.", 0.8)
		else:
			label("No Ethernet interface found. Connect Wi-Fi.", 0.8)
		button("Wi-Fi settings", app.show_wifi)
		button("Retry", present)
		button("Skip for offline play", app.skip_setup)
		focus()
		# Give USB Ethernet/DHCP a bounded chance to come up. The screen stays
		# usable; this is not an indefinite gate on network access.
		for _attempt in 15:
			await app.get_tree().create_timer(1.0).timeout
			if generation != _generation or app._screen_node != self:
				return
			if app.net.local_ip() != "":
				toast("Network connected.", "success")
				present()
				return
		return

	label("Starting setup…", 0.75)
	await app.get_tree().create_timer(0.4).timeout
	# Configure the controller before the browser account step so first boot
	# can sign in with a pad in hand.
	app.show_controller_setup(false)


func _use_iso(path: String) -> void:
	var saved := cli(["settings", "set", "game.iso", path])
	if not saved.get("ok", false):
		toast(str(saved.get("error", "Could not save the Melee image.")), "error")
		present()
		return
	app.refresh_status()
	app.show_controller_setup(true)


func back() -> void:
	app.show_menu()
