extends "res://screens/screen.gd"


var _status: Label


func present() -> void:
	clear()
	title("UPDATING SYSTEM", 44)
	label("Downloading the system update. This can take a while.", 0.7)
	_status = label("Starting…", 0.8)
	app.update_job.progress.connect(_on_progress)


func _on_progress(phase: String, done: int, total: int, error: String) -> void:
	if not is_instance_valid(_status):
		return
	if phase == "downloading":
		var text := "Downloading"
		if total > 0:
			text += " %.1f%% (%s / %s)" % [
				float(done) * 100.0 / float(total), _size(done), _size(total)
			]
		_status.text = text + "…"
	elif phase == "installing":
		_status.text = "Installing to the inactive system slot…"
	elif phase == "error":
		_status.text = error if error != "" else "Update status unavailable."
	else:
		_status.text = "Working…"


func _size(value: int) -> String:
	if value >= 1000000000:
		return "%.1f GB" % (float(value) / 1000000000.0)
	if value >= 1000000:
		return "%.1f MB" % (float(value) / 1000000.0)
	if value >= 1000:
		return "%.1f KB" % (float(value) / 1000.0)
	return "%d B" % value


func back() -> void:
	pass
