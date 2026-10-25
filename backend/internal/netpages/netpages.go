// Package netpages serves the phone-facing sign-in and ISO upload pages.
package netpages

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

const (
	// MaxUpload is the largest ISO accepted by the upload page.
	MaxUpload = int64(8 * 1024 * 1024 * 1024)

	maxFormBytes = 1 << 20
)

type pageData struct {
	Token   string
	Message string
	Tone    string
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>SlippiOS</title>
<style>
body{font-family:system-ui,sans-serif;background:#111;color:#eee;max-width:34rem;margin:2rem auto;padding:0 1rem;line-height:1.5}
h1{font-size:1.5rem}h2{font-size:1.1rem;margin-top:2rem}
input,button{font:inherit}input[type=text],input[type=email],input[type=password]{width:100%;padding:.6rem;margin:.3rem 0;border-radius:.4rem;border:1px solid #333;background:#1b1b1b;color:#eee}
button{background:#2b6cb0;color:#fff;border:0;border-radius:.4rem;padding:.6rem 1rem;margin-top:.4rem}
.ok{color:#7bd88f}.err{color:#ff8b8b}code{background:#222;padding:.1rem .3rem;border-radius:.2rem}
</style></head><body>
<h1>SlippiOS</h1>
<p>This page is only reachable on your local network.</p>
{{if .Message}}<p class="{{.Tone}}">{{.Message}}</p>{{end}}
<h2>Sign in to Slippi</h2>
<form method="post" action="/{{.Token}}/auth">
  <input type="email" name="email" placeholder="Email" autocomplete="username" required>
  <input type="password" name="password" placeholder="Password" autocomplete="current-password" required>
  <button type="submit">Sign in</button>
</form>
<h2>Add a game image</h2>
<p>Select a Melee 1.02 ISO. It is copied to <code>/data/isos</code>.</p>
<input id="file" type="file" accept=".iso,application/x-iso9660-image">
<button onclick="upload()">Upload</button>
<pre id="status"></pre>
<script>
async function upload() {
  const status = document.getElementById('status');
  const file = document.getElementById('file').files[0];
  if (!file) { status.textContent = 'Choose an ISO first.'; return; }
  status.textContent = 'Uploading ' + file.name + '…';
  try {
    const response = await fetch('/{{.Token}}/iso?name=' + encodeURIComponent(file.name), {method: 'POST', body: file});
    status.textContent = await response.text();
  } catch (error) { status.textContent = String(error); }
}
</script></body></html>`))

// Server hosts the LAN pages for one token.
type Server struct {
	Token  string
	Port   int
	Paths  settings.Paths
	Login  func(email, password string) (auth.User, error)
	Status string
	Marker string

	listener net.Listener
	http     *http.Server
	mu       sync.Mutex
	closed   bool
}

func New(token string, port int, paths settings.Paths) *Server {
	system := filepath.Join(paths.DataDir, "system")
	return &Server{
		Token:  token,
		Port:   port,
		Paths:  paths,
		Login:  auth.Login,
		Status: filepath.Join(system, "net-status.json"),
		Marker: filepath.Join(system, "net-upload.json"),
	}
}

// LocalAddress is the address a phone on the same network can reach.
func LocalAddress() string {
	connection, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer connection.Close()
	if address, ok := connection.LocalAddr().(*net.UDPAddr); ok && address.IP != nil {
		return address.IP.String()
	}
	return "127.0.0.1"
}

func (s *Server) writeStatus(ok bool, extra map[string]any) {
	payload := map[string]any{"ok": ok}
	for key, value := range extra {
		payload[key] = value
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	data = append(data, '\n')
	_ = os.MkdirAll(filepath.Dir(s.Status), 0o755)
	temporary := s.Status + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return
	}
	_ = os.Rename(temporary, s.Status)
}

func (s *Server) writeMarker(name, path string) {
	_ = os.MkdirAll(filepath.Dir(s.Marker), 0o755)
	data, err := json.Marshal(map[string]any{
		"name": name,
		"path": path,
		"at":   time.Now().Unix(),
	})
	if err != nil {
		return
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.Marker, data, 0o644); err != nil {
		return
	}
	_ = os.Chmod(s.Marker, 0o644)
}

func (s *Server) page(message string, isError bool) string {
	tone := "ok"
	if isError {
		tone = "err"
	}
	var rendered strings.Builder
	if err := pageTemplate.Execute(&rendered, pageData{
		Token:   s.Token,
		Message: message,
		Tone:    tone,
	}); err != nil {
		return ""
	}
	return rendered.String()
}

func (s *Server) sendHTML(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	_, _ = io.WriteString(w, body)
}

func (s *Server) sendText(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(code)
	_, _ = io.WriteString(w, body)
}

func (s *Server) route(request *http.Request) ([]string, bool) {
	path := strings.Trim(request.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] != s.Token {
		return nil, false
	}
	return parts[1:], true
}

func (s *Server) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	parts, ok := s.route(request)
	if !ok {
		http.NotFound(w, request)
		return
	}
	switch request.Method {
	case http.MethodGet:
		if len(parts) != 0 {
			http.NotFound(w, request)
			return
		}
		s.sendHTML(w, http.StatusOK, s.page("", false))
	case http.MethodPost:
		switch {
		case len(parts) == 1 && parts[0] == "auth":
			s.authenticate(w, request)
		case len(parts) == 1 && parts[0] == "iso":
			s.upload(w, request)
		default:
			http.NotFound(w, request)
		}
	case http.MethodPut:
		s.upload(w, request)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) authenticate(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, maxFormBytes)
	if err := request.ParseForm(); err != nil {
		s.sendHTML(w, http.StatusBadRequest, s.page("Invalid sign-in form.", true))
		return
	}
	email := strings.TrimSpace(request.PostFormValue("email"))
	password := request.PostFormValue("password")

	user, err := s.Login(email, password)
	if err != nil {
		s.sendHTML(w, http.StatusBadRequest, s.page(err.Error(), true))
		return
	}

	if err := os.MkdirAll(s.Paths.ProfileDir, 0o755); err != nil {
		s.sendHTML(w, http.StatusInternalServerError, s.page(err.Error(), true))
		return
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		s.sendHTML(w, http.StatusInternalServerError, s.page(err.Error(), true))
		return
	}
	target := filepath.Join(s.Paths.ProfileDir, settings.DefaultUserJSON)
	if err := os.WriteFile(target, data, 0o600); err != nil {
		s.sendHTML(w, http.StatusInternalServerError, s.page(err.Error(), true))
		return
	}
	_ = os.Chmod(target, 0o600)

	installed, detail := s.Paths.InstallLogin(settings.Defaults())
	if !installed {
		s.sendHTML(w, http.StatusInternalServerError, s.page(detail, true))
		return
	}

	name := user.DisplayName
	if name == "" {
		name = "player"
	}
	s.sendHTML(w, http.StatusOK, s.page(
		fmt.Sprintf("Signed in as %s (%s). You can return to the console.", name, user.ConnectCode),
		false,
	))
}

func (s *Server) upload(w http.ResponseWriter, request *http.Request) {
	name := filepath.Base(request.URL.Query().Get("name"))
	if name == "" || name == "." {
		name = "upload.iso"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".iso") {
		s.sendText(w, http.StatusBadRequest, "Only .iso files are accepted.")
		return
	}

	length := request.ContentLength
	if length <= 0 || length > MaxUpload {
		s.sendText(w, http.StatusRequestEntityTooLarge, "Missing or oversized upload.")
		return
	}

	if err := os.MkdirAll(s.Paths.ISODir, 0o755); err != nil {
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return
	}
	partial := filepath.Join(s.Paths.ISODir, name+".part")
	destination := filepath.Join(s.Paths.ISODir, name)

	file, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return
	}
	written, copyErr := io.CopyN(file, request.Body, length)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written != length {
		_ = os.Remove(partial)
		if errors.Is(copyErr, io.EOF) || errors.Is(copyErr, io.ErrUnexpectedEOF) || copyErr == nil {
			s.sendText(w, http.StatusBadRequest, "Upload incomplete.")
			return
		}
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+copyErr.Error())
		return
	}

	valid, reason := s.Paths.ISOStatus(filepath.Base(partial))
	if !valid {
		_ = os.Remove(partial)
		s.sendText(w, http.StatusBadRequest, reason)
		return
	}
	if err := os.Rename(partial, destination); err != nil {
		_ = os.Remove(partial)
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return
	}
	_ = os.Chmod(destination, 0o444)
	s.writeMarker(name, destination)
	s.sendText(w, http.StatusOK, fmt.Sprintf("Added %s. Return to the console.", name))
}

// Start listens on the configured port and publishes the phone URL.
func (s *Server) Start() (string, error) {
	_ = os.MkdirAll(filepath.Dir(s.Status), 0o755)
	_ = os.Remove(s.Marker)

	var listener net.Listener
	var lastError error
	for attempt := 0; attempt < 10; attempt++ {
		address := fmt.Sprintf("0.0.0.0:%d", s.Port)
		listener, lastError = net.Listen("tcp4", address)
		if lastError == nil {
			break
		}
		if errors.Is(lastError, syscall.EADDRINUSE) && attempt < 9 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		s.writeStatus(false, map[string]any{"error": lastError.Error()})
		return "", lastError
	}

	port := s.Port
	if address, ok := listener.Addr().(*net.TCPAddr); ok {
		port = address.Port
	}
	url := fmt.Sprintf("http://%s:%d/%s/", LocalAddress(), port, s.Token)
	s.listener = listener
	s.http = &http.Server{Handler: s}
	s.writeStatus(true, map[string]any{"url": url})
	return url, nil
}

// Serve blocks until the server is closed.
func (s *Server) Serve() error {
	if s.http == nil {
		return errors.New("network page is not started")
	}
	err := s.http.Serve(s.listener)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if err != nil {
		s.writeStatus(false, map[string]any{"error": err.Error()})
	}
	return err
}

// Close stops the listener and records that the page is no longer running.
func (s *Server) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()
	var err error
	if s.http != nil {
		err = s.http.Close()
	} else if s.listener != nil {
		err = s.listener.Close()
	}
	s.writeStatus(false, map[string]any{"error": "network page stopped"})
	return err
}
