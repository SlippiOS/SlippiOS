package dolphin

// ShaderCompilationMode values from Dolphin's VideoConfig.h.
const (
	ShaderModeSynchronous int64 = iota
	ShaderModeSynchronousUber
	ShaderModeAsynchronousUber
	ShaderModeAsynchronousSkip
)

func ShaderModeValue(mode string) int64 {
	switch mode {
	case "sync":
		return ShaderModeSynchronous
	case "sync_uber":
		return ShaderModeSynchronousUber
	case "async_skip":
		return ShaderModeAsynchronousSkip
	default:
		return ShaderModeAsynchronousUber
	}
}

func ShaderModeName(value int64) string {
	switch value {
	case ShaderModeSynchronous:
		return "sync"
	case ShaderModeSynchronousUber:
		return "sync_uber"
	case ShaderModeAsynchronousSkip:
		return "async_skip"
	default:
		return "async_uber"
	}
}

func ShaderModeLabel(mode string) string {
	switch mode {
	case "sync":
		return "Synchronous"
	case "sync_uber":
		return "Synchronous (Uber)"
	case "async_skip":
		return "Asynchronous (Skip Rendering)"
	default:
		return "Asynchronous (Uber)"
	}
}

// StableAspectValue maps the shell's aspect setting to Ishiiruka's enum.
func StableAspectValue(aspect int64) int64 {
	switch aspect {
	case 1:
		return 1
	case 2:
		return 2
	case 4:
		return 3
	default:
		return 5
	}
}

// AspectFromStable maps Ishiiruka's enum back to the shell's aspect setting.
func AspectFromStable(value int64) (int64, bool) {
	switch value {
	case 1:
		return 1, true
	case 2:
		return 2, true
	case 3:
		return 4, true
	case 5:
		return 3, true
	default:
		return 0, false
	}
}
