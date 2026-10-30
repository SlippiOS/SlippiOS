extends "res://screens/screen.gd"


var _status: Label
var _detail: Label
var _bar: ProgressBar
var _retry: Button
var _wifi: Button
var _done := false


func present(_message: String = "") -> void:
	clear()
	title("INSTALLING SLIPPI", 44)
	label("Downloading the stable, beta, and playback builds. This needs internet.", 0.7)
	_status = label("Starting…", 0.85)
	_detail = label("", 0.55)
	_bar = _add_bar()
	_retry = button("Retry", _retry_install)
	_retry.visible = false
	_wifi = button("Wi-Fi settings", _open_wifi)
	_wifi.visible = false
	focus()
	app.install_job.progress.connect(_on_progress)
	app.install_job.finished.connect(_on_finished)
	if not app.install_job.active():
		app.install_job.start()


func _on_progress(phase: String, done: int, total: int, error: String) -> void:
	if app._screen_node != self or not is_instance_valid(_status):
		return
	var value := 0.0
	if phase == "beta" and total > 0:
		_status.text = "Downloading the beta build… %d%%" % (done * 100 / total)
		value = 0.33 + 0.33 * float(done) / float(total)
	elif phase == "stable":
		_status.text = "Downloading the stable build…"
		value = 0.05
	elif phase == "beta":
		_status.text = "Downloading the beta build…"
		value = 0.33
	elif phase == "playback":
		_status.text = "Downloading the playback build…"
		value = 0.66
	elif phase == "ready":
		_status.text = "Finishing…"
		value = 1.0
	elif phase == "error":
		_status.text = "Install failed."
	else:
		_status.text = "Working…"
	if _bar != null:
		_bar.value = value
	if _detail != null and error != "":
		_detail.text = error


func _on_finished(ok: bool, error: String) -> void:
	if _done or app._screen_node != self:
		return
	if ok:
		_done = true
		app.finish_app_install()
		return
	_fail(error if error != "" else "The install did not finish. Check the network and retry.")


func _add_bar() -> ProgressBar:
	var bar := ProgressBar.new()
	bar.min_value = 0.0
	bar.max_value = 1.0
	bar.value = 0.0
	bar.show_percentage = false
	bar.custom_minimum_size = Vector2(360, 4)
	bar.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	var background := StyleBoxFlat.new()
	background.bg_color = SlippiTheme.surface
	background.corner_radius_top_left = 2
	background.corner_radius_top_right = 2
	background.corner_radius_bottom_left = 2
	background.corner_radius_bottom_right = 2
	var fill := StyleBoxFlat.new()
	fill.bg_color = SlippiTheme.progress
	fill.corner_radius_top_left = 2
	fill.corner_radius_top_right = 2
	fill.corner_radius_bottom_left = 2
	fill.corner_radius_bottom_right = 2
	bar.add_theme_stylebox_override("background", background)
	bar.add_theme_stylebox_override("fill", fill)
	add_content(bar)
	return bar


func _retry_install() -> void:
	if _retry != null:
		_retry.visible = false
	if _wifi != null:
		_wifi.visible = false
	if _detail != null:
		_detail.text = ""
	if _status != null:
		_status.text = "Starting…"
	if _bar != null:
		_bar.value = 0.0
	app.install_job.start()


func _fail(message: String) -> void:
	if _status != null:
		_status.text = message
	if _retry != null:
		_retry.visible = true
		_retry.grab_focus()
	if _wifi != null:
		_wifi.visible = true


func _open_wifi() -> void:
	app.show_wifi("Connect to Wi-Fi or Ethernet, then retry the install.")


func back() -> void:
	pass
