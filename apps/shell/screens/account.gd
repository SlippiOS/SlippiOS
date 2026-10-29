extends "res://screens/screen.gd"


func present(message: String = "") -> void:
	clear()
	title("ACCOUNT", 44)
	var login: Dictionary = app.status.get("login", {})
	if login.get("signed_in", false):
		label("Signed in: %s" % str(login.get("detail", "unknown")), 0.9)
		button("Sign out", _sign_out)
	else:
		label("Not signed in.", 0.8)
		button("Sign in", app.show_auth_email)
		button("Sign in on another device", app.start_net.bind("login"))
	if message != "":
		label(message, 0.9)
	button("Back", app.show_menu)
	focus()


func _sign_out() -> void:
	var result := cli(["logout"])
	if not result.get("ok", false):
		present(str(result.get("error", "Could not sign out.")))
		return
	app.refresh_status()
	present("Signed out.")


func back() -> void:
	app.show_menu()
