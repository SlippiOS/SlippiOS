extends "res://screens/screen.gd"

func present(_message: String = "") -> void:
	clear()
	title("EXPERT TOOLS", 40)
	label("These tools can change the system. Use them with care.", 0.7)
	button("Dolphin settings window", app.open_dolphin_settings)
	button("Terminal", app.open_terminal)
	button("Text editor", app.open_editor)
	button("Custom display mode", app.show_custom_mode.bind(""))
	button("Back", app.show_settings.bind(""))
	focus()


func back() -> void:
	app.show_settings()
