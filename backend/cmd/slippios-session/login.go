package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func ensureLogin(paths settings.Paths, values map[string]any) (bool, string) {
	if ok, detail := paths.InstallLogin(values); ok {
		return true, detail
	}
	for attempt := 0; attempt < 3; attempt++ {
		clear()
		fmt.Print("\n                 SLIPPI ACCOUNT\n\n")
		email, ok := line("Email (blank to skip): ")
		if !ok || email == "" {
			return false, "skipped"
		}
		password, ok := readPassword()
		if !ok {
			return false, "skipped"
		}
		user, err := auth.Login(email, password)
		if err != nil {
			fmt.Printf("\n                 %s\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		target := filepath.Join(paths.ProfileDir, settings.DefaultUserJSON)
		if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		data, err := jsonMarshalIndent(user)
		if err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			alert([]string{"Could not save the login.", err.Error()})
			return false, "sign-in failed"
		}
		_ = os.Chmod(target, 0o600)
		return paths.InstallLogin(values)
	}
	return false, "sign-in failed"
}

func jsonMarshalIndent(value any) ([]byte, error) {
	return json.MarshalIndent(value, "", "  ")
}
