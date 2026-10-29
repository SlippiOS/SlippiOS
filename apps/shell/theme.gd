extends Node


var background := Color("#0b0b10")
var surface := Color("#202633")
var accent := Color("#44a963")
var text := Color("#eef1f7")
var brand := Color("#4aa8ff")
var progress := Color("#4aa8ff")

var mark_fraction := 0.30
var mark_fit := 0.92
var pulse_speed := 5.2
var pulse_min := 0.35
var pulse_max := 1.0
var scale_pulse := 0.015


func _ready() -> void:
	_load()
	RenderingServer.set_default_clear_color(background)


func _load() -> void:
	var path := "/etc/slippios/theme.json"
	if not FileAccess.file_exists(path):
		return
	var parsed: Variant = JSON.parse_string(FileAccess.get_file_as_string(path))
	if typeof(parsed) != TYPE_DICTIONARY:
		return
	var data: Dictionary = parsed
	background = _color(data.get("background"), background)
	surface = _color(data.get("surface"), surface)
	accent = _color(data.get("accent"), accent)
	text = _color(data.get("text"), text)
	brand = _color(data.get("brand"), brand)
	progress = _color(data.get("progress"), progress)
	var splash: Dictionary = data.get("splash", {})
	mark_fraction = float(splash.get("mark_fraction", mark_fraction))
	mark_fit = float(splash.get("mark_fit", mark_fit))
	pulse_speed = float(splash.get("pulse_speed", pulse_speed))
	pulse_min = float(splash.get("pulse_min", pulse_min))
	pulse_max = float(splash.get("pulse_max", pulse_max))
	scale_pulse = float(splash.get("scale_pulse", scale_pulse))


func _color(value: Variant, fallback: Color) -> Color:
	if typeof(value) != TYPE_STRING:
		return fallback
	var hex := str(value)
	if hex.begins_with("#") and (hex.length() == 7 or hex.length() == 9):
		return Color(hex)
	return fallback
