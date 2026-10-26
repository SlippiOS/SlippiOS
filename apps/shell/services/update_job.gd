extends RefCounted

## Runs `slippios os update` and turns its status file into progress/finish
## signals. The app owns navigation; this only manages the process.

signal progress(phase: String, done: int, total: int, error: String)
signal finished(phase: String, error: String)

const Cli := preload("res://services/cli.gd")

var pid := -1
var waiting := false

var _timer := 0.0


func start() -> void:
	if active():
		return
	pid = OS.create_process("slippios", PackedStringArray(["os", "update"]))
	waiting = true
	_timer = 0.0


func active() -> bool:
	return pid != -1 or waiting


func poll(delta: float) -> void:
	if not active():
		return
	_timer += delta
	if _timer < 1.0:
		return
	_timer = 0.0

	var result := Cli.run(["os", "status"])
	if not result.get("ok", false):
		progress.emit("error", 0, 0, str(result.get("error", "Update status unavailable.")))
		return

	var phase := str(result.get("phase", "idle"))
	var done := int(result.get("done", 0))
	var total := int(result.get("total", 0))
	var error := str(result.get("error", ""))

	if phase == "installed":
		waiting = false
		_kill()
		finished.emit("installed", "")
		return
	if phase == "failed":
		waiting = false
		_kill()
		finished.emit("failed", error if error != "" else "The system update could not be installed.")
		return

	if pid != -1 and not OS.is_process_running(pid):
		pid = -1
	if pid == -1 and phase == "idle" and waiting:
		waiting = false
		finished.emit("idle", "")
		return

	progress.emit(phase, done, total, error)


func _kill() -> void:
	if pid != -1:
		if OS.is_process_running(pid):
			OS.kill(pid)
		pid = -1
