extends "res://screens/screen.gd"

func present(_message: String = "") -> void:
	clear()
	title("POWER OFF", 44)
	label("Power off the console?", 0.8)
	button("Power off", _poweroff)
	button("Cancel", app.show_menu)
	focus()


func _poweroff() -> void:
	clear()
	title("POWERING OFF", 44)
	var status := label("You can unplug the drive when the screen goes dark.", 0.8)
	await app.get_tree().process_frame
	var result := cli(["poweroff"])
	if not result.get("ok", false):
		status.text = str(result.get("error", "Could not power off."))
		button("Back", app.show_menu)
		focus()


func back() -> void:
	app.show_menu()
