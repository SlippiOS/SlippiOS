package osupdate

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// currentBootID identifies the running boot, so an install can tell a
// completed reboot from a rollback to the previous system.
func currentBootID() string {
	data, err := os.ReadFile(bootIDFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
func numberValue(value any) int64 {
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case json.Number:
		number, _ := strconv.ParseInt(typed.String(), 10, 64)
		return number
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func errorValue(progress, service map[string]any) any {
	if value, ok := progress["error"]; ok && value != nil && value != "" {
		return value
	}
	if value, ok := service["error"]; ok && value != nil && value != "" {
		return value
	}
	return nil
}

// Status reports the installed version and any staged update.
func (p Paths) Status() map[string]any {
	current := currentVersion(p)
	cached := readJSON(p.Latest)
	service := readJSON(p.StatusFile)
	progress := readJSON(p.Progress)
	latest := cached["latest"]
	available := false
	if latestVersion, ok := latest.(string); ok && latestVersion != "" {
		available = latestVersion != current && CompareVersions(latestVersion, current) > 0
	}

	phase := "idle"
	if value, ok := progress["phase"].(string); ok && value != "" {
		phase = value
	}
	if value, ok := service["phase"].(string); ok && value != "" {
		phase = value
	}
	latestVersion, _ := latest.(string)
	target, _ := service["version"].(string)
	if target == "" {
		// Status written by an older installer does not name its target.
		target = latestVersion
	}

	switch phase {
	case "installed":
		if target != "" && target == current {
			// The staged update is now running.
			service = map[string]any{}
			_ = os.Remove(p.StatusFile)
			_ = os.Remove(p.Progress)
			phase = "idle"
		} else if target != "" && target != current {
			recorded, _ := service["boot_id"].(string)
			if recorded != "" && recorded != currentBootID() {
				// The console rebooted but came back on another version:
				// boot assessment rolled the update back.
				service = map[string]any{
					"ok":    false,
					"phase": "failed",
					"error": "The updated system did not boot; the previous system was restored.",
				}
				phase = "failed"
				_ = writeJSON(p.StatusFile, service)
				_ = os.Remove(p.Progress)
			}
		}
	case "failed":
		// Once the running version is the one the last update targeted, the
		// failure is stale (for example after a manual install).
		if latestVersion != "" && latestVersion == current {
			service = map[string]any{}
			_ = os.Remove(p.StatusFile)
			_ = os.Remove(p.Progress)
			phase = "idle"
		}
	}

	var serviceOK any
	if value, ok := service["ok"]; ok {
		serviceOK = value
	}
	return map[string]any{
		"ok":               true,
		"current":          current,
		"latest":           latest,
		"update_available": available,
		"phase":            phase,
		"done":             numberValue(progress["done"]),
		"total":            numberValue(progress["total"]),
		"error":            errorValue(progress, service),
		"service_ok":       serviceOK,
	}
}
