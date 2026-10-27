extends "res://screens/screen.gd"


var mode := "installed"
var message := ""


func present(_message: String = "") -> void:
	clear()
	match mode:
		"installed":
			title("UPDATE INSTALLED", 44)
			label("Restart to boot the new system. The previous system is kept for rollback.", 0.75)
			button("Restart now", app.request_reboot)
			button("Later", app.show_menu)
		"failed":
			title("UPDATE FAILED", 44)
			label(message, 0.9)
			button("Back", app.show_update)
		"reboot_failed":
			title("RESTART FAILED", 44)
			label(message, 0.9)
			button("Back", app.show_update)
		"restarting":
			title("RESTARTING", 44)
			label("The console will reboot into the new system.", 0.75)
		_:
			title("UPDATE", 44)
			button("Back", app.show_update)
	focus()


func back() -> void:
	match mode:
		"installed", "restarting":
			pass
		_:
			app.show_update()
