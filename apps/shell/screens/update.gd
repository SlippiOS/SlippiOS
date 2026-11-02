extends "res://screens/screen.gd"


func present() -> void:
	clear()
	title("UPDATE", 44)

	if app.update_job.active():
		label("A system update is in progress.", 0.8)
		button("View progress", app.show_os_progress)
		button("Back", app.show_settings)
		focus()
		return

	var os_check := cli(["os", "check"])
	var os_status := cli(["os", "status"])
	var dolphin := cli(["update", "--check"])
	var settings: Dictionary = cli(["settings", "show"]).get("settings", {})
	var dolphin_settings: Dictionary = settings.get("dolphin", {})
	var build := text_value(dolphin_settings.get("build"), "stable")
	var stable_version := text_value(
		app.status.get("apps", {}).get("stable", {}).get("version"), "not installed")

	section("System")
	if os_check.get("ok", false):
		var current := text_value(os_check.get("current"), "unknown")
		var latest := text_value(os_check.get("latest"), "unknown")
		label("System: %s" % current, 0.8)
		if os_status.get("phase") == "failed":
			label(text_value(os_status.get("error"), "Last system update failed."), 0.8)
		if os_check.get("update_available", false):
			label("Available: %s" % latest, 0.8)
			button("Install system update", app.start_os_update)
		else:
			label("System is up to date.", 0.65)
	elif os_status.get("phase") == "failed":
		label("Last system update failed.", 0.9)
	else:
		label(str(os_check.get("error", "System update check failed.")), 0.9)

	if os_status.get("phase") == "installed":
		button("Restart to finish update", app.request_reboot)

	section("Dolphin")
	label("Build: %s" % build, 0.65)
	label("Stable: %s" % stable_version, 0.65)
	if dolphin.get("ok", false):
		label("Beta: %s" % text_value(dolphin.get("current"), "not installed"), 0.65)
		if dolphin.get("current") != dolphin.get("latest"):
			label("Latest beta: %s" % str(dolphin.get("latest")), 0.65)
	else:
		label(str(dolphin.get("error", "Dolphin update check failed.")), 0.8)

	button("Back", app.show_settings)
	focus()


func back() -> void:
	app.show_settings()
