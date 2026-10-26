extends "res://screens/screen.gd"

## Progress while the system update downloads and installs.

var _status: Label


func present(_message: String = "") -> void:
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
			text += " %d%%" % (done * 100 / total)
		_status.text = text + "…"
	elif phase == "installing":
		_status.text = "Installing to the inactive system slot…"
	elif phase == "error":
		_status.text = error if error != "" else "Update status unavailable."
	else:
		_status.text = "Working…"


func back() -> void:
	pass
