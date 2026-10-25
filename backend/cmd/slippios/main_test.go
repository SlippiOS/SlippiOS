package main

import (
	"encoding/json"
	"io"
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

	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	loginCommand([]string{"--email", "player@example.com", "--password", "secret"}, modes{json: true})
	_ = writer.Close()
	os.Stdout = old
	output, _ := io.ReadAll(reader)
	reader.Close()

	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("output = %q: %v", output, err)
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
