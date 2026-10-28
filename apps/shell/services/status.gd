extends RefCounted


var app


func _init(shell) -> void:
	app = shell


func refresh() -> void:
	app.status = app.cli(["status"])


func needs_setup() -> bool:
	var settings: Dictionary = app.status.get("settings", {})
	return not settings.get("exists", false) or not settings.get("iso_valid", false)


func has_valid_isos() -> bool:
	var result: Dictionary = app.cli(["isos"])
	for entry in result.get("isos", []):
		if entry.get("valid", false):
			return true
	return false


func expert_enabled() -> bool:
	var settings: Dictionary = app.cli(["settings", "show"]).get("settings", {})
	return bool(settings.get("expert", {}).get("enabled", false))


func text() -> String:
	var settings: Dictionary = app.status.get("settings", {})
	var login: Dictionary = app.status.get("login", {})
	var dolphin: Dictionary = app.status.get("dolphin", {})
	var game: String = "ready" if settings.get("iso_valid", false) else str(settings.get("iso_reason", "not configured"))
	var account: String = "signed in" if login.get("signed_in", false) else "not signed in"
	var version: String = app.text_value(dolphin.get("current"), "not installed")
	var lines := PackedStringArray(["Game: %s" % game, "Login: %s" % account, "Dolphin: %s" % version])
	var os: Dictionary = app.status.get("os", {})
	var os_version := app.text_value(os.get("version"), "")
	if os_version != "":
		lines.append("SlippiOS: %s" % os_version)
	return "\n".join(lines)
