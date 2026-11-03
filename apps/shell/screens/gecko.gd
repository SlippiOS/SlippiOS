extends "res://screens/screen.gd"


func present() -> void:
	clear()
	title("GECKO CODES", 40)
	label("Required codes are locked. Changes apply the next time Melee starts.", 0.6)

	var result := cli(["gecko", "list"])
	if not result.get("ok", false):
		label(str(result.get("error", "Could not read the Gecko codes.")), 0.9)
		button("Back", app.show_dolphin_group.bind("advanced"))
		focus()
		return

	section("Codes")
	var codes: Array = array_value(result.get("codes", []))
	if codes.is_empty():
		label("No Gecko codes found.", 0.8)
	for code in codes:
		var name := str(code.get("name", ""))
		if name == "":
			continue
		var enabled := bool(code.get("enabled", false))
		var required := bool(code.get("required", false))
		var row := ("[x] " if enabled else "[ ] ") + name
		if required:
			row += "  (required)"
		button(row, _toggle.bind(name, enabled, required))

	section("Custom")
	button("Custom codes…", app.show_gecko_custom)
	button("Back", app.show_dolphin_group.bind("advanced"))
	focus()


func _toggle(name: String, enabled: bool, required: bool) -> void:
	if required:
		toast("%s is required and cannot be disabled." % name, "error")
		app.show_gecko()
		return
	var result := cli(["gecko", "set", name, "off" if enabled else "on"])
	if not result.get("ok", false):
		toast(str(result.get("error", "Could not save the Gecko code.")), "error")
		app.show_gecko()
		return
	toast("%s is now %s." % [name, "off" if enabled else "on"], "success")
	app.show_gecko()


func back() -> void:
	app.show_dolphin_group("advanced")
