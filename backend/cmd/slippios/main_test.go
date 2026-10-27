package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func TestReadCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.json")
	if err := os.WriteFile(path, []byte(`{"email":"a@example.com","password":"secret"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := readCredentials(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if value.Email != "a@example.com" || value.Password != "secret" {
		t.Fatalf("credentials = %#v", value)
	}

	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readCredentials(bad, false); err == nil {
		t.Fatal("invalid file accepted")
	}
	if _, err := readCredentials(filepath.Join(t.TempDir(), "missing"), false); err == nil {
		t.Fatal("missing file accepted")
	}
}

func TestLoginCommandInstallsLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/identity":
			_, _ = w.Write([]byte(`{"idToken":"token","localId":"uid"}`))
		case "/graphql":
			_, _ = w.Write([]byte(`{"data":{"getUser":{
				"displayName":"Player One",
				"connectCode":{"code":"PLR#1"},
				"private":{"playKey":"key"}
			}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	savedKey, savedIdentity, savedGraphQL, savedClient := auth.APIKey, auth.IdentityURL, auth.GraphQLURL, auth.Client
	auth.APIKey = "test"
	auth.IdentityURL = server.URL + "/identity"
	auth.GraphQLURL = server.URL + "/graphql"
	auth.Client = server.Client()
	t.Cleanup(func() {
		auth.APIKey, auth.IdentityURL, auth.GraphQLURL, auth.Client = savedKey, savedIdentity, savedGraphQL, savedClient
	})

	root := t.TempDir()
	t.Setenv("SLIPPIOS_DATA", root)

	var output bytes.Buffer
	mode := &modes{json: true, out: &output, err: &output}
	if err := runLogin(mode, loginOptions{email: "player@example.com", password: "secret"}); err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(output.Bytes(), &payload); err != nil {
		t.Fatalf("output = %q: %v", output.String(), err)
	}
	if payload["ok"] != true || payload["connect_code"] != "PLR#1" {
		t.Fatalf("payload = %#v", payload)
	}

	paths := settings.PathsFor(root)
	installed, err := os.ReadFile(paths.DolphinUserFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installed), `"playKey": "key"`) {
		t.Fatalf("installed = %s", installed)
	}
}

func TestCommandTree(t *testing.T) {
	root := newRootCommand()
	paths := [][]string{
		{"status"},
		{"settings"},
		{"settings", "show"},
		{"settings", "set"},
		{"isos"},
		{"login"},
		{"net"},
		{"update"},
		{"os"},
		{"os", "check"},
		{"os", "status"},
		{"os", "update"},
		{"os", "reboot"},
		{"play"},
		{"dolphin"},
		{"dolphin", "config"},
		{"dolphin", "reset"},
		{"display"},
		{"display", "apply"},
		{"display", "modes"},
		{"display", "mode"},
		{"display", "custom"},
		{"audio"},
		{"audio", "sinks"},
		{"poweroff"},
		{"wifi"},
		{"wifi", "scan"},
		{"wifi", "status"},
		{"wifi", "connect"},
	}
	for _, path := range paths {
		command, _, err := root.Find(path)
		if err != nil {
			t.Fatalf("find %v: %v", path, err)
		}
		if command == nil {
			t.Fatalf("missing command %v", path)
		}
	}
}
