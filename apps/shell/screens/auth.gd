extends "res://screens/screen.gd"


const Keyboard := preload("res://widgets/keyboard.gd")

var stage := "email"
var email := ""


func present() -> void:
	clear()
	title("EMAIL" if stage == "email" else "PASSWORD", 40)
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
			toast("Enter a valid email address.", "error")
			return
		email = address
		stage = "password"
		present()
		return
	_submit_login(value)


func _submit_login(password: String) -> void:
	if password == "":
		toast("Password cannot be empty.", "error")
		return

	var path := ProjectSettings.globalize_path("user://credentials.json")
	var file := FileAccess.open(path, FileAccess.WRITE)
	if file == null:
		toast("Could not prepare the sign-in request.", "error")
		return
	file.store_string(JSON.stringify({"email": email, "password": password}))
	file.close()
	OS.execute("chmod", PackedStringArray(["600", path]))

	var result := cli(["login", "--file", path])
	DirAccess.remove_absolute(path)

	if result.get("ok", false):
		app.refresh_status()
		toast("Signed in.", "success")
		app.show_account()
	else:
		toast(str(result.get("error", "Sign-in failed.")), "error")


func back() -> void:
	app.show_account()
