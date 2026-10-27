package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
	"github.com/spf13/cobra"
)

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginOptions struct {
	email    string
	password string
	stdin    bool
	file     string
}

func newLoginCommand(mode *modes) *cobra.Command {
	options := loginOptions{}
	command := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Slippi",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(mode, options)
		},
	}
	command.Flags().StringVar(&options.email, "email", "", "Slippi account email")
	command.Flags().StringVar(&options.password, "password", "", "Slippi account password")
	command.Flags().BoolVar(&options.stdin, "stdin", false, "read {email,password} JSON from stdin")
	command.Flags().StringVar(&options.file, "file", "", "read {email,password} JSON from a file")
	return command
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

func runLogin(mode *modes, options loginOptions) error {
	credentials, err := readCredentials(options.file, options.stdin)
	if err != nil {
		return mode.fail(err.Error())
	}
	if credentials.Email == "" {
		credentials.Email = options.email
	}
	if credentials.Password == "" {
		credentials.Password = options.password
	}
	if credentials.Email == "" || credentials.Password == "" {
		return mode.fail("email and password are required")
	}

	user, err := auth.Login(credentials.Email, credentials.Password)
	if err != nil {
		return mode.fail(err.Error())
	}

	paths := settings.DefaultPaths()
	if err := os.MkdirAll(paths.ProfileDir, 0o755); err != nil {
		return mode.fail(err.Error())
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return mode.fail(err.Error())
	}
	target := paths.Resolve(settings.DefaultUserJSON)
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return mode.fail(fmt.Sprintf("cannot write %s: %v", target, err))
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
	return mode.emit(payload, func() {
		mode.printf("Signed in as %s (%s)\n", user.DisplayName, user.ConnectCode)
	})
}
