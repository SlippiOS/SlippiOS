extends RefCounted


signal progress(phase: String, done: int, total: int, error: String)
signal finished(ok: bool, error: String)

const Cli := preload("res://services/cli.gd")
const PROGRESS_PATH := "/data/system/apps-progress.json"

var pid := -1
var waiting := false

var _timer := 0.0


func start() -> void:
	if active():
		return
	if FileAccess.file_exists(PROGRESS_PATH):
		DirAccess.remove_absolute(PROGRESS_PATH)
	pid = OS.create_process("slippios", PackedStringArray([
		"apps", "install", "--progress", PROGRESS_PATH,
	]))
	waiting = true
	_timer = 0.0
	if pid == -1:
		waiting = false
		finished.emit(false, "Could not start the installer.")


func active() -> bool:
	return pid != -1 or waiting


func poll(delta: float) -> void:
	if not active():
		return
	_timer += delta
	if _timer < 1.0:
		return
	_timer = 0.0

	_read_progress()

	if pid != -1 and OS.is_process_running(pid):
		return
	pid = -1
	if not waiting:
		return
	waiting = false

	var result := Cli.run(["apps", "status"])
	if bool(result.get("installed", false)):
		finished.emit(true, "")
		return
	finished.emit(false, "The install did not finish. Check the network and retry.")


func _read_progress() -> void:
	if not FileAccess.file_exists(PROGRESS_PATH):
		return
	var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(PROGRESS_PATH))
	if typeof(parsed) != TYPE_DICTIONARY:
		return
	var data: Dictionary = parsed
	progress.emit(
		str(data.get("phase", "")),
		int(data.get("done", 0)),
		int(data.get("total", 0)),
		str(data.get("error", "")),
	)
