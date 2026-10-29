extends "res://screens/screen.gd"


var _status: Label


func present(message: String = "") -> void:
	clear()
	match app.net.mode:
		"setup":
			title("SET UP ON ANOTHER DEVICE", 34)
			label("Scan with a phone, or open the address on any computer.", 0.75)
		"iso":
			title("ADD A GAME", 44)
			label("Open this address on a phone or computer:", 0.8)
		_:
			title("SIGN IN ON ANOTHER DEVICE", 34)
			label("Open this address on a phone or computer:", 0.8)

	label(app.net.url, 1.0)
	var qr = _make_qr(app.net.url)
	if qr != null:
		add_content(qr)

	if app.net.mode == "setup":
		label("Step 1: add Melee. Step 2: sign in or skip.", 0.65)
		_status = label(message if message != "" else "Waiting for setup…", 0.75)
		button("Skip for offline play", app.skip_setup)
	else:
		label("Upload a Melee ISO." if app.net.mode == "iso" else "Enter your Slippi email and password.", 0.7)
		_status = label("Waiting…", 0.7)
		button("Back", back)

	app.net.status_changed.connect(_on_status)
	app.net.stopped.connect(_on_stopped)
	focus()


func _on_status(text: String) -> void:
	if is_instance_valid(_status):
		_status.text = text


func _on_stopped(error: String) -> void:
	if not is_instance_valid(_status):
		return
	if error != "":
		_status.text = "Network page stopped: %s" % error
	else:
		_status.text = "Network page stopped."


func _make_qr(url: String) -> TextureRect:
	var path := ProjectSettings.globalize_path("user://qr.png")
	var code := OS.execute("qrencode", PackedStringArray([
		"-o", path, "-s", "8", "-m", "2", url,
	]))
	if code != 0:
		return null
	var image := Image.load_from_file(path)
	if image == null:
		return null
	var rect := TextureRect.new()
	rect.texture = ImageTexture.create_from_image(image)
	rect.custom_minimum_size = Vector2(260, 260)
	rect.expand_mode = TextureRect.EXPAND_IGNORE_SIZE
	rect.stretch_mode = TextureRect.STRETCH_KEEP_ASPECT_CENTERED
	return rect


func back() -> void:
	var mode: String = app.net.mode
	app.stop_net()
	match mode:
		"login":
			app.show_account()
		"setup":
			app.show_menu()
		_:
			app.show_roms("")
