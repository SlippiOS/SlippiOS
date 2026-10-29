extends "res://screens/screen.gd"

func present(message: String = "") -> void:
	clear()
	if app.setup_flow:
		title("CONNECT", 44)
		label("Connect to Wi-Fi or plug in Ethernet to continue setup.", 0.8)
	else:
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
		var networks: Array = array_value(result.get("networks", []))
		if networks.is_empty():
			label("No networks found.", 0.7)
		for network in networks:
			var ssid := str(network.get("ssid", ""))
			var security := str(network.get("security", ""))
			var network_label := ssid
			if security != "":
				network_label += "  (%s)" % security
			button(network_label, _select.bind(ssid, security))

	button("Rescan", present.bind(""))
	if app.setup_flow:
		button("Skip for offline play", app.skip_setup)
	button("Back", back)
	focus()


func _select(ssid: String, security: String) -> void:
	if security == "" or security == "--":
		app.connect_wifi(ssid, "")
	else:
		app.show_wifi_password(ssid)


func back() -> void:
	if app.setup_flow:
		app.show_setup()
	else:
		app.show_settings()
