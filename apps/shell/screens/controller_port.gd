extends "res://screens/screen.gd"

const DEVICES := ["adapter", "pad", "keyboard", "none"]

var port := 1


func present(message: String = "") -> void:
	clear()
	title("PORT %d" % port, 44)

	var result := cli(["controllers", "list"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the controller settings.")), 0.9)
		button("Back", app.show_controllers)
		focus()
		return

	var current := _current(result.get("ports", []))
	var device := text_value(current.get("device"), "adapter")
	var profile := text_value(current.get("profile"), "")
	var profiles: Array = result.get("profiles", [])

	if message != "":
		label(message, 0.9)
	button("Input: %s" % _device_label(device), _cycle_device.bind(device, profile))
	if device == "pad":
		button("Profile: %s" % _profile_label(profile, profiles),
			_cycle_profile.bind(profile, profiles))
		label("Use a profile that matches the connected pad. Custom keeps changes made in Dolphin.", 0.6)
	else:
		label("The GameCube adapter is native and needs no profile.", 0.6)
	label("Changes apply the next time Melee starts.", 0.6)
	button("Back", app.show_controllers)
	focus()


func _current(ports: Array) -> Dictionary:
	for item in ports:
		if int(item.get("port", 0)) == port:
			return item
	return {}


func _device_label(device: String) -> String:
	match device:
		"pad":
			return "standard pad"
		"keyboard":
			return "keyboard"
		"none":
			return "none"
		_:
			return "GameCube adapter"


func _profile_label(name: String, profiles: Array) -> String:
	if name == "":
		return "not set"
	if name == "custom":
		return "custom (Dolphin settings)"
	for profile in profiles:
		if str(profile.get("name", "")) != name:
			continue
		if bool(profile.get("connected", false)):
			return "%s (detected)" % name
		return name
	return name


func _cycle_device(current: String, profile: String) -> void:
	var index := DEVICES.find(current)
	if index < 0:
		index = 0
	var next: String = DEVICES[(index + 1) % DEVICES.size()]
	if next == "pad" and profile == "":
		profile = _detected_profile()
	var saved := cli(["controllers", "set", str(port), next, profile])
	if not saved.get("ok", false):
		app.show_controller_port(port, str(saved.get("error", "Could not save the controller.")))
		return
	app.show_controller_port(port, "Port %d set to %s." % [port, _device_label(next)])


func _cycle_profile(current: String, profiles: Array) -> void:
	var names: Array[String] = []
	for profile in profiles:
		names.append(str(profile.get("name", "")))
	names.append("custom")
	var index := names.find(current)
	if index < 0:
		index = 0
	var next := names[(index + 1) % names.size()]
	var saved := cli(["controllers", "set", str(port), "pad", next])
	if not saved.get("ok", false):
		app.show_controller_port(port, str(saved.get("error", "Could not apply the profile.")))
		return
	app.show_controller_port(port, "Profile set to %s." % _profile_label(next, profiles))


func _detected_profile() -> String:
	var result := cli(["controllers", "profiles"])
	for profile in result.get("profiles", []):
		if bool(profile.get("connected", false)):
			return str(profile.get("name", ""))
	return "custom"


func back() -> void:
	app.show_controllers()
