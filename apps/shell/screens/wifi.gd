extends "res://screens/screen.gd"

## Wi-Fi network picker.

func present(message: String = "") -> void:
	clear()
	title("WI-FI", 44)

	var status := cli(["wifi", "status"])
	if status.get("ok", false) and status.get("connected", false):
		label("Connected to %s" % str(status.get("ssid")), 0.8)
	else:
		label("Not connected.", 0.8)
	if message != "":
		label(message, 0.9)

	var result := cli(["wifi", "scan"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not scan for networks.")), 0.9)
	else:
		var networks: Array = result.get("networks", [])
		if networks.is_empty():
			label("No networks found.", 0.7)
		var shown := 0
		for network in networks:
			if shown >= 8:
				break
			var ssid := str(network.get("ssid", ""))
			var security := str(network.get("security", ""))
			var network_label := ssid
			if security != "":
				network_label += "  (%s)" % security
			button(network_label, _select.bind(ssid, security))
			shown += 1

	button("Rescan", present.bind(""))
	button("Back", app.show_settings.bind(""))
	focus()


func _select(ssid: String, security: String) -> void:
	if security == "" or security == "--":
		app.connect_wifi(ssid, "")
	else:
		app.show_wifi_password(ssid)


func back() -> void:
	app.show_settings()
