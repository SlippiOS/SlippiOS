extends "res://screens/screen.gd"


func present(_message: String = "") -> void:
	clear()
	title("ACCOUNT", 44)
	var login: Dictionary = app.status.get("login", {})
	if login.get("signed_in", false):
		label("Signed in: %s" % str(login.get("detail", "unknown")), 0.9)
	else:
		label("Not signed in.", 0.8)
		button("Sign in", app.show_auth_email)
		button("Sign in on another device", app.start_net.bind("login"))
	button("Back", app.show_menu)
	focus()


func back() -> void:
	app.show_menu()
