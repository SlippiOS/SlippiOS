extends "res://screens/screen.gd"

func present() -> void:
	clear()
	title("EXPERT TOOLS", 44)
	label("These tools can change the system. Use them with care.", 0.7)

	section("Tools")
	button("Open Dolphin GUI", app.open_dolphin_settings)
	button("Terminal", app.open_terminal)
	button("Text editor", app.open_editor)

	section("Display")
	button("Custom display mode", app.show_custom_mode)

	button("Back", app.show_settings)
	focus()


func back() -> void:
	app.show_settings()
