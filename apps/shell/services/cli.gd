extends RefCounted

## Thin wrapper around the slippios CLI. JSON output is automatic when stdout
## is piped, so the shell never has to parse human text.

const COMMAND := "slippios"


static func run(args: Array, read_stderr: bool = true) -> Dictionary:
	var output: Array = []
	var code := OS.execute(COMMAND, PackedStringArray(args), output, read_stderr)
	var text := ""
	for line in output:
		text += str(line)
	var parsed: Variant = JSON.parse_string(text)
	if typeof(parsed) != TYPE_DICTIONARY:
		return {"ok": false, "error": "invalid CLI output (exit %d)" % code}
	return parsed
