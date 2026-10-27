package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func readCredentials(path string, fromStdin bool) (credentials, error) {
	var data []byte
	var err error
	if path != "" {
		data, err = os.ReadFile(path)
		if err != nil {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
	} else if fromStdin {
		data, err = io.ReadAll(os.Stdin)
		if err != nil {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
	} else {
		return credentials{}, nil
	}
	var value credentials
	if err := json.Unmarshal(data, &value); err != nil {
		if path != "" {
			return credentials{}, fmt.Errorf("cannot read credentials: %w", err)
		}
		return credentials{}, fmt.Errorf("invalid JSON: %w", err)
	}
	return value, nil
}

func loginCommand(args []string, mode modes) {
	flags := flag.NewFlagSet("login", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	email := flags.String("email", "", "Slippi account email")
	password := flags.String("password", "", "Slippi account password")
	fromStdin := flags.Bool("stdin", false, "read {email,password} JSON from stdin")
	file := flags.String("file", "", "read {email,password} JSON from a file")
	if err := flags.Parse(args); err != nil {
		fail(err.Error())
	}

	credentials, err := readCredentials(*file, *fromStdin)
	if err != nil {
		fail(err.Error())
	}
	if credentials.Email == "" {
		credentials.Email = *email
	}
	if credentials.Password == "" {
		credentials.Password = *password
	}
	if credentials.Email == "" || credentials.Password == "" {
		fail("email and password are required")
	}

	user, err := auth.Login(credentials.Email, credentials.Password)
	if err != nil {
		fail(err.Error())
	}

	paths := settings.DefaultPaths()
	if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
		fail(err.Error())
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	target := paths.Resolve(settings.DefaultUserJSON)
	if err := os.WriteFile(target, data, 0o600); err != nil {
		fail(fmt.Sprintf("cannot write %s: %v", target, err))
	}
	_ = os.Chmod(target, 0o600)

	values, err := paths.Load()
	if err != nil {
		values = settings.Defaults()
	}
	installed, detail := paths.InstallLogin(values)
	payload := map[string]any{
		"ok":           installed,
		"connect_code": user.ConnectCode,
		"display_name": user.DisplayName,
		"detail":       detail,
	}
	if !installed {
		payload["error"] = detail
	}
	emit(payload, mode, func() {
		fmt.Printf("Signed in as %s (%s)\n", user.DisplayName, user.ConnectCode)
	})
}
