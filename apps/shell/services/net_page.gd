extends RefCounted


signal status_changed(text: String)
signal stopped(error: String)
signal setup_done
signal upload_done
signal login_done

const Cli := preload("res://services/cli.gd")

var pid := -1
var mode := ""
var url := ""
var active := false

var _timer := 0.0


func start(next_mode: String) -> void:
	stop()
	var system_dir := DirAccess.open(data_dir() + "/system")
	if system_dir != null:
		system_dir.remove("net-upload.json")
		system_dir.remove("net-status.json")
	var token := Crypto.new().generate_random_bytes(8).hex_encode()
	url = "http://%s:8080/%s/" % [_local_ip(), token]
	mode = next_mode
	_timer = 0.0
	pid = OS.create_process("slippios", PackedStringArray([
		"net", "--token", token, "--port", "8080", "--mode", next_mode,
	]))
	active = true


func stop() -> void:
	if pid != -1:
		OS.kill(pid)
		pid = -1
	active = false


func poll(delta: float) -> void:
	if not active:
		return
	_timer += delta
	if _timer < 1.0:
		return
	_timer = 0.0

	if pid != -1 and not OS.is_process_running(pid):
		pid = -1
		if _completed():
			return
		active = false
		var status := _read_json(data_dir() + "/system/net-status.json")
		var error := ""
		if not status.get("ok", true):
			error = str(status.get("error", "unknown error"))
		stopped.emit(error)
		return

	match mode:
		"setup":
			_poll_setup()
		"iso":
			_poll_iso()
		_:
			_poll_login()


func _completed() -> bool:
	if mode == "setup":
		if _read_json(data_dir() + "/system/setup.json").get("done", false):
			_finish_setup()
			return true
	elif mode == "iso":
		if _read_json(data_dir() + "/system/net-upload.json").get("name", "") != "":
			_finish_iso()
			return true
	else:
		var login: Dictionary = Cli.run(["status"]).get("login", {})
		if login.get("signed_in", false):
			_finish_login()
			return true
	return false


func _poll_setup() -> void:
	var setup := _read_json(data_dir() + "/system/setup.json")
	if setup.get("done", false):
		_finish_setup()
		return
	var problem := str(setup.get("error", ""))
	if problem != "":
		status_changed.emit(problem)
	elif not setup.get("game_ready", false):
		status_changed.emit("Waiting for game selection…")
	elif not setup.get("account_ready", false):
		status_changed.emit("Game added. Finish sign-in or skip in the browser.")
	else:
		status_changed.emit("Finishing setup…")


func _poll_iso() -> void:
	var marker := _read_json(data_dir() + "/system/net-upload.json")
	if marker.get("name", "") != "":
		_finish_iso()
		return
	status_changed.emit("Waiting for upload…")


func _poll_login() -> void:
	var result := Cli.run(["status"])
	var login: Dictionary = result.get("login", {})
	if login.get("signed_in", false):
		_finish_login()
		return
	status_changed.emit("Waiting for sign-in…")


func _finish_setup() -> void:
	stop()
	setup_done.emit()


func _finish_iso() -> void:
	stop()
	upload_done.emit()


func _finish_login() -> void:
	stop()
	login_done.emit()


func _read_json(path: String) -> Dictionary:
	if not FileAccess.file_exists(path):
		return {}
	var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(path))
	return parsed if typeof(parsed) == TYPE_DICTIONARY else {}


func local_ip() -> String:
	var result := Cli.run(["status"])
	var network: Dictionary = result.get("network", {})
	var address := str(network.get("address", ""))
	if address == "" or address == "127.0.0.1" or address.begins_with("169.254."):
		return ""
	return address


func _local_ip() -> String:
	var address := local_ip()
	return address if address != "" else "127.0.0.1"


func data_dir() -> String:
	var configured := OS.get_environment("SLIPPIOS_DATA")
	return configured if configured != "" else "/data"
