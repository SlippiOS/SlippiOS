extends "res://screens/screen.gd"


func present() -> void:
	clear()
	title("CUSTOM CODES", 40)
	label("Add codes from the Expert terminal, then enable them here.", 0.7)
	label("slippios gecko add /path/to/code.txt", 0.9)

	var result := cli(["gecko", "list"])
	if result.get("ok", false):
		var found := false
		for code in result.get("codes", []):
			if not bool(code.get("user_defined", false)):
				continue
			var name := str(code.get("name", ""))
			if name == "":
				continue
			found = true
			button("Remove: %s" % name, _remove.bind(name))
		if not found:
			label("No custom codes installed.", 0.7)

	else:
		label(str(result.get("error", "Could not read the Gecko codes.")), 0.9)

	button("Back", app.show_gecko)
	focus()


func _remove(name: String) -> void:
	var result := cli(["gecko", "remove", name])
	if not result.get("ok", false):
		toast(str(result.get("error", "Could not remove the code.")), "error")
		app.show_gecko_custom()
		return
	toast("Removed %s." % name, "success")
	app.show_gecko_custom()


func back() -> void:
	app.show_gecko()
