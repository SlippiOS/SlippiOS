extends "res://screens/screen.gd"

## On-TV email/password entry. The browser flow remains available.

const Keyboard := preload("res://widgets/keyboard.gd")

var stage := "email"
var email := ""


func present(message: String = "") -> void:
	clear()
	title("EMAIL" if stage == "email" else "PASSWORD", 40)
	if message != "":
		label(message, 0.9)
	var keyboard = Keyboard.new(stage == "password")
	keyboard.submitted.connect(_on_submitted)
	add_content(keyboard)
	button("Use browser instead", app.start_net.bind("login"))
	button("Back", back)
	focus()


func _on_submitted(value: String) -> void:
	if stage == "email":
		var address := value.strip_edges()
		if address == "" or not address.contains("@"):
			present("Enter a valid email address.")
			return
		email = address
		stage = "password"
		present("")
		return
	_submit_login(value)


func _submit_login(password: String) -> void:
	if password == "":
		present("Password cannot be empty.")
		return

	var path := ProjectSettings.globalize_path("user://credentials.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		present("Could not prepare the sign-in request.")
		return
	file.store_string(JSON.stringify({"email": email, "password": password}))
	file.close()
	OS.execute("chmod", PackedStringArray(["600", path]))

	var result := cli(["login", "--file", path])
	DirAccess.remove_absolute(path)

	if result.get("ok", false):
		app.refresh_status()
		app.show_account()
	else:
		present(str(result.get("error", "Sign-in failed.")))


func back() -> void:
	app.show_account()
