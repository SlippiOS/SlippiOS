package netpages

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

func testServer(t *testing.T, login func(string, string) (auth.User, error)) (*Server, *httptest.Server) {
	t.Helper()
	server := New("abc123", 0, settings.PathsFor(t.TempDir()))
	if login != nil {
		server.Login = login
	}
	return server, httptest.NewServer(server)
}

func TestPageRequiresToken(t *testing.T) {
	_, httpServer := testServer(t, nil)
	defer httpServer.Close()

	response, err := http.Get(httpServer.URL + "/wrong/")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", response.StatusCode)
	}

	response, err = http.Get(httpServer.URL + "/abc123/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if !strings.Contains(string(body), `action="/abc123/auth"`) ||
		!strings.Contains(string(body), "/abc123/iso?name=") {
		t.Fatalf("page = %s", body)
	}
}

func TestAuthenticate(t *testing.T) {
	server, httpServer := testServer(t, func(email, password string) (auth.User, error) {
		if email != "player@example.com" || password != "secret" {
			return auth.User{}, &auth.AuthError{Message: "Incorrect email or password."}
		}
		return auth.User{
			UID: "u", DisplayName: "Player One", ConnectCode: "PLR#1", PlayKey: "k",
		}, nil
	})
	defer httpServer.Close()

	response, err := http.PostForm(httpServer.URL+"/abc123/auth", url.Values{
		"email":    {"player@example.com"},
		"password": {"secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, body)
	}
	if !strings.Contains(string(body), "Signed in as Player One (PLR#1)") {
		t.Fatalf("body = %s", body)
	}

	source, err := os.ReadFile(filepath.Join(server.Paths.ProfileDir, settings.DefaultUserJSON))
	if err != nil {
		t.Fatal(err)
	}
	var user map[string]any
	if err := json.Unmarshal(source, &user); err != nil {
		t.Fatal(err)
	}
	if user["playKey"] != "k" || user["connectCode"] != "PLR#1" {
		t.Fatalf("user = %#v", user)
	}
	installed, err := os.ReadFile(server.Paths.DolphinUserFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(source) {
		t.Fatalf("installed = %s", installed)
	}
}

func TestAuthenticateFailure(t *testing.T) {
	_, httpServer := testServer(t, func(string, string) (auth.User, error) {
		return auth.User{}, &auth.AuthError{Message: "Incorrect email or password."}
	})
	defer httpServer.Close()

	response, err := http.PostForm(httpServer.URL+"/abc123/auth", url.Values{
		"email": {"player@example.com"}, "password": {"wrong"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "Incorrect email or password.") {
		t.Fatalf("status = %d: %s", response.StatusCode, body)
	}
}

func TestUpload(t *testing.T) {
	server, httpServer := testServer(t, nil)
	defer httpServer.Close()

	image := append([]byte("GALE01"), make([]byte, 128)...)
	response, err := http.Post(
		httpServer.URL+"/abc123/iso?name=melee.iso",
		"application/octet-stream",
		bytes.NewReader(image),
	)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d: %s", response.StatusCode, body)
	}
	if !strings.Contains(string(body), "Added melee.iso") {
		t.Fatalf("body = %s", body)
	}

	destination := filepath.Join(server.Paths.ISODir, "melee.iso")
	stored, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, image) {
		t.Fatal("stored image differs")
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o444 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}

	marker, err := os.ReadFile(server.Marker)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(marker, &value); err != nil {
		t.Fatal(err)
	}
	if value["name"] != "melee.iso" || value["path"] != destination {
		t.Fatalf("marker = %#v", value)
	}
	if _, err := os.Stat(destination + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial file still exists")
	}
}

func TestUploadRejectsInvalidImages(t *testing.T) {
	server, httpServer := testServer(t, nil)
	defer httpServer.Close()

	response, err := http.Post(
		httpServer.URL+"/abc123/iso?name=notes.txt",
		"text/plain", strings.NewReader("hello"),
	)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("extension status = %d", response.StatusCode)
	}

	response, err = http.Post(
		httpServer.URL+"/abc123/iso?name=bad.iso",
		"application/octet-stream", strings.NewReader("not an iso"),
	)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("content status = %d", response.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(server.Paths.ISODir, "bad.iso.part")); !os.IsNotExist(err) {
		t.Fatal("invalid partial still exists")
	}
}

func TestUploadLengthErrors(t *testing.T) {
	server, httpServer := testServer(t, nil)
	defer httpServer.Close()

	image := append([]byte("GALE01"), make([]byte, 32)...)
	request := httptest.NewRequest(http.MethodPost,
		"/abc123/iso?name=short.iso", bytes.NewReader(image))
	request.ContentLength = int64(len(image) + 10)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("short status = %d", recorder.Code)
	}

	request, err := http.NewRequest(http.MethodPost,
		httpServer.URL+"/abc123/iso?name=huge.iso", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.ContentLength = MaxUpload + 1
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize status = %d", response.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(server.Paths.ISODir, "short.iso.part")); !os.IsNotExist(err) {
		t.Fatal("partial still exists")
	}
}

func TestSetupFlow(t *testing.T) {
	paths := settings.PathsFor(t.TempDir())
	if err := os.MkdirAll(paths.ISODir, 0o755); err != nil {
		t.Fatal(err)
	}
	isoPath := filepath.Join(paths.ISODir, "melee.iso")
	if err := os.WriteFile(isoPath, append([]byte("GALE01"), make([]byte, 64)...), 0o444); err != nil {
		t.Fatal(err)
	}
	server := New("abc123", 0, paths)
	server.Mode = "setup"
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	response, err := http.Get(httpServer.URL + "/abc123/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "Add your game") ||
		!strings.Contains(string(body), "melee.iso") {
		t.Fatalf("game step = %s", body)
	}

	response, err = client.PostForm(httpServer.URL+"/abc123/select", url.Values{
		"path": {isoPath},
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("select status = %d", response.StatusCode)
	}
	if values, err := paths.Load(); err != nil || values["game"].(map[string]any)["iso"] != "melee.iso" {
		t.Fatalf("settings = %#v, %v", values, err)
	}
	state := readSetupFile(server.Setup)
	if !state.GameReady || state.AccountReady || state.Done {
		t.Fatalf("state after game = %#v", state)
	}

	response, err = http.Get(httpServer.URL + "/abc123/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "Sign in to Slippi") ||
		!strings.Contains(string(body), "Skip for offline play") {
		t.Fatalf("account step = %s", body)
	}

	response, err = client.PostForm(httpServer.URL+"/abc123/skip", url.Values{})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("skip status = %d", response.StatusCode)
	}
	state = readSetupFile(server.Setup)
	if !state.Done || state.Account != "offline" {
		t.Fatalf("state after skip = %#v", state)
	}

	response, err = http.Get(httpServer.URL + "/abc123/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if !strings.Contains(string(body), "Melee is starting on your TV") {
		t.Fatalf("done step = %s", body)
	}
}

func TestSetupUploadAndLogin(t *testing.T) {
	server := New("abc123", 0, settings.PathsFor(t.TempDir()))
	server.Mode = "setup"
	server.Login = func(email, password string) (auth.User, error) {
		return auth.User{
			UID: "u", DisplayName: "Player One", ConnectCode: "PLR#1", PlayKey: "key",
		}, nil
	}
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}

	image := append([]byte("GALE01"), make([]byte, 128)...)
	response, err := http.Post(
		httpServer.URL+"/abc123/iso?name=melee.iso",
		"application/octet-stream", bytes.NewReader(image),
	)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("upload status = %d: %s", response.StatusCode, body)
	}
	state := readSetupFile(server.Setup)
	if !state.GameReady || state.AccountReady {
		t.Fatalf("state after upload = %#v", state)
	}

	response, err = client.PostForm(httpServer.URL+"/abc123/auth", url.Values{
		"email": {"player@example.com"}, "password": {"secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("auth status = %d", response.StatusCode)
	}
	state = readSetupFile(server.Setup)
	if !state.Done || state.Account != "PLR#1" {
		t.Fatalf("state after auth = %#v", state)
	}
	if _, err := os.Stat(server.Paths.DolphinUserFile); err != nil {
		t.Fatal(err)
	}
}

func TestSetupRejectsUnknownGame(t *testing.T) {
	server := New("abc123", 0, settings.PathsFor(t.TempDir()))
	server.Mode = "setup"
	httpServer := httptest.NewServer(server)
	defer httpServer.Close()

	response, err := http.PostForm(httpServer.URL+"/abc123/select", url.Values{
		"path": {"/etc/passwd"},
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", response.StatusCode)
	}
	if state := readSetupFile(server.Setup); state.GameReady {
		t.Fatalf("state = %#v", state)
	}
}

func TestStartPublishesStatus(t *testing.T) {
	server := New("abc123", 0, settings.PathsFor(t.TempDir()))
	url, err := server.Start()
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = server.Serve() }()
	t.Cleanup(func() { _ = server.Close() })

	if !strings.Contains(url, "/abc123/") {
		t.Fatalf("url = %s", url)
	}
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}

	data, err := os.ReadFile(server.Status)
	if err != nil {
		t.Fatal(err)
	}
	var status map[string]any
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status["ok"] != true || status["url"] != url {
		t.Fatalf("status = %#v", status)
	}

	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(server.Status)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	if status["ok"] != false || status["error"] != "network page stopped" {
		t.Fatalf("stopped status = %#v", status)
	}
}
