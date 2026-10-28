extends "res://screens/screen.gd"


const BootLogo := preload("res://boot_logo.gd")
const BOOT_HOLD_MS := 2600


func present(_message: String = "") -> void:
	clear()
	app.center_content_vertically(true)
	var logo := BootLogo.new()
	add_content(logo)
	var heading: Label = app.add_title("SLIPPIOS", 64)
	heading.modulate.a = 0.0
	var prompt := label("Hold A for settings", 0.0)
	var bar := _add_boot_bar()
	var tween: Tween = app.create_tween().set_parallel(true)
	tween.tween_property(heading, "modulate:a", 1.0, 0.55)
	tween.tween_property(prompt, "modulate:a", 0.65, 0.45).set_delay(0.75)
	tween.tween_property(bar, "value", 1.0, 2.4).set_trans(Tween.TRANS_SINE)

	var deadline := Time.get_ticks_msec() + BOOT_HOLD_MS
	while Time.get_ticks_msec() < deadline:
		if app.settings_held():
			tween.kill()
			app.show_menu()
			return
		await app.get_tree().process_frame
	await app.after_splash()


func _add_boot_bar() -> ProgressBar:
	var bar := ProgressBar.new()
	bar.min_value = 0
	bar.max_value = 1
	bar.value = 0
	bar.show_percentage = false
	bar.custom_minimum_size = Vector2(360, 4)
	bar.size_flags_horizontal = Control.SIZE_SHRINK_CENTER
	var background := StyleBoxFlat.new()
	background.bg_color = SlippiTheme.surface
	background.corner_radius_top_left = 2
	background.corner_radius_top_right = 2
	background.corner_radius_bottom_left = 2
	background.corner_radius_bottom_right = 2
	var fill := StyleBoxFlat.new()
	fill.bg_color = SlippiTheme.accent
	fill.corner_radius_top_left = 2
	fill.corner_radius_top_right = 2
	fill.corner_radius_bottom_left = 2
	fill.corner_radius_bottom_right = 2
	bar.add_theme_stylebox_override("background", background)
	bar.add_theme_stylebox_override("fill", fill)
	add_content(bar)
	return bar


func back() -> void:
	pass
