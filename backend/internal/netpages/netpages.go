// Package netpages serves the LAN sign-in and ISO upload pages.
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

type setupPage struct {
	Token        string
	GameReady    bool
	AccountReady bool
	Done         bool
	GameName     string
	Account      string
	ISOs         []settings.ISO
	Message      string
	Error        string
}

var setupTemplate = template.Must(template.New("setup").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Set up SlippiOS</title>
<style>
:root{color-scheme:dark}
*{box-sizing:border-box}
body{margin:0;min-height:100vh;background:radial-gradient(circle at 50% -10%,#1d2738 0,#0b0b10 55%);color:#eef1f7;font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;line-height:1.5}
main{width:min(36rem,calc(100% - 2rem));margin:0 auto;padding:3rem 0 4rem}
.brand{font-size:.85rem;font-weight:700;letter-spacing:.22em;text-transform:uppercase;color:#7db5ff}
h1{font-size:2rem;margin:.4rem 0 .2rem}
.sub{color:#9aa4b5;margin:0 0 2rem}
.card{background:#141821;border:1px solid #252c3a;border-radius:1rem;padding:1.4rem;box-shadow:0 20px 60px rgb(0 0 0 / .3)}
.steps{display:flex;align-items:center;gap:.6rem;margin:0 0 1.5rem}
.step{display:flex;align-items:center;gap:.45rem;color:#6f7a8c;font-size:.85rem;white-space:nowrap}
.step b{display:grid;place-items:center;width:1.6rem;height:1.6rem;border-radius:50%;background:#252c3a;font-size:.75rem}
.step.active{color:#eef1f7}.step.active b{background:#2b6cb0;color:#fff}
.step.done{color:#79d99a}.step.done b{background:#1f5c3a;color:#b9f5cf}
.line{height:1px;flex:1;background:#252c3a}
h2{font-size:1.25rem;margin:0 0 .35rem}
p{margin:.4rem 0}
.muted{color:#9aa4b5;font-size:.92rem}
.chosen{background:#0f131b;border:1px solid #252c3a;border-radius:.6rem;padding:.65rem .8rem;margin:.9rem 0;color:#cdd5e2}
label{display:block;font-size:.82rem;color:#9aa4b5;margin:.75rem 0 .25rem}
input,select,button{font:inherit}
input[type=text],input[type=email],input[type=password],select{width:100%;padding:.7rem .8rem;border-radius:.55rem;border:1px solid #30394a;background:#0f131b;color:#eef1f7}
button{width:100%;border:0;border-radius:.55rem;padding:.8rem 1rem;background:#2b6cb0;color:#fff;font-weight:650;cursor:pointer;margin-top:.7rem}
button:hover{background:#347cc7}
button.secondary{background:#222a37;color:#cbd4e2;font-weight:500}
button.secondary:hover{background:#2c3646}
.or{text-align:center;color:#6f7a8c;font-size:.82rem;margin:1rem 0 .2rem}
.status{min-height:1.4rem;color:#9aa4b5;font-size:.88rem;word-break:break-word}
.err{color:#ff8b8b}.ok{color:#7bd88f}
.done-title{color:#79d99a}
.small{font-size:.82rem;color:#6f7a8c}
</style></head><body><main>
<div class="brand">SlippiOS</div>
<h1>{{if .Done}}You're all set{{else}}Set up your console{{end}}</h1>
<p class="sub">This page only works on your local network.</p>
<section class="card">
  <div class="steps">
    <div class="step {{if .GameReady}}done{{else}}active{{end}}"><b>1</b> Game</div>
    <div class="line"></div>
    <div class="step {{if .AccountReady}}done{{else if .GameReady}}active{{end}}"><b>2</b> Account</div>
  </div>

  {{if .Error}}<p class="err">{{.Error}}</p>{{end}}
  {{if .Message}}<p class="ok">{{.Message}}</p>{{end}}

  {{if not .GameReady}}
    <h2>Add your game</h2>
    <p class="muted">Upload a Melee 1.02 ISO, or choose one already on the console.</p>
    {{if .ISOs}}
    <form method="post" action="/{{.Token}}/select">
      <label for="game">Game image</label>
      <select id="game" name="path">
        {{range .ISOs}}<option value="{{.Path}}">{{.Name}}</option>{{end}}
      </select>
      <button type="submit">Use this game</button>
    </form>
    <p class="or">or upload a new image</p>
    {{end}}
    <div>
      <input id="file" type="file" accept=".iso,application/x-iso9660-image">
      <button type="button" onclick="upload()">Upload ISO</button>
      <p id="status" class="status"></p>
    </div>
  {{else if not .AccountReady}}
    <h2>Sign in to Slippi</h2>
    <div class="chosen">Game: {{.GameName}}</div>
    <p class="muted">Sign in for online play, or skip to play offline.</p>
    <form method="post" action="/{{.Token}}/auth">
      <label for="email">Email</label>
      <input id="email" type="email" name="email" autocomplete="username" required>
      <label for="password">Password</label>
      <input id="password" type="password" name="password" autocomplete="current-password" required>
      <button type="submit">Sign in</button>
    </form>
    <form method="post" action="/{{.Token}}/skip">
      <button type="submit" class="secondary">Skip for offline play</button>
    </form>
  {{else}}
    <h2 class="done-title">Melee is starting on your TV</h2>
    <p class="muted">You can close this page.</p>
    {{if .Account}}<p class="small">Slippi: {{.Account}}</p>{{end}}
  {{end}}
</section>
</main>
<script>
async function upload() {
  const status = document.getElementById('status');
  const file = document.getElementById('file').files[0];
  if (!file) { status.textContent = 'Choose an ISO first.'; return; }
  status.textContent = 'Uploading ' + file.name + '…';
  try {
    const response = await fetch('/{{.Token}}/iso?name=' + encodeURIComponent(file.name), {method: 'POST', body: file});
    const text = await response.text();
    if (response.ok) {
      window.location.href = '/{{.Token}}/';
    } else {
      status.textContent = text;
    }
  } catch (error) { status.textContent = String(error); }
}
</script></body></html>`))

// Server hosts the LAN pages for one token.
type Server struct {
	Token  string
	Port   int
	Paths  settings.Paths
	Mode   string
	Login  func(email, password string) (auth.User, error)
	Status string
	Marker string
	Setup  string

	listener net.Listener
	http     *http.Server
	mu       sync.Mutex
	setupMu  sync.Mutex
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
		Setup:  filepath.Join(system, "setup.json"),
	}
}

// LocalAddress is the address another device on the same network can reach.
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

type setupState struct {
	GameReady    bool   `json:"game_ready"`
	AccountReady bool   `json:"account_ready"`
	Done         bool   `json:"done"`
	ISO          string `json:"iso,omitempty"`
	Account      string `json:"account,omitempty"`
	Error        string `json:"error,omitempty"`
	UpdatedAt    int64  `json:"updated_at"`
}

func readSetupFile(path string) setupState {
	state := setupState{}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &state)
	}
	return state
}

func (s *Server) setupStatusLocked() setupState {
	state := readSetupFile(s.Setup)
	state.GameReady = false
	state.ISO = ""
	values, err := s.Paths.Load()
	if err == nil {
		if game, ok := values["game"].(map[string]any); ok {
			if raw, ok := game["iso"].(string); ok {
				if valid, _ := s.Paths.ISOStatus(raw); valid {
					state.GameReady = true
					state.ISO = raw
				}
			}
		}
	}
	if ok, detail := s.Paths.LoginStatus(s.Paths.DolphinUserFile); ok {
		state.AccountReady = true
		state.Account = detail
	}
	state.Done = state.GameReady && state.AccountReady
	state.Error = ""
	return state
}

func (s *Server) setupStatus() setupState {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	return s.setupStatusLocked()
}

func (s *Server) persistSetupLocked(state setupState) error {
	state.Done = state.GameReady && state.AccountReady
	state.UpdatedAt = time.Now().Unix()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(s.Setup), 0o755); err != nil {
		return err
	}
	temporary := s.Setup + ".tmp"
	if err := os.WriteFile(temporary, data, 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, s.Setup)
}

func (s *Server) refreshSetup() {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	state := s.setupStatusLocked()
	_ = s.persistSetupLocked(state)
}

func (s *Server) selectGame(value string) error {
	valid, reason := s.Paths.ISOStatus(value)
	if !valid {
		if reason == "" {
			reason = "not a Melee image"
		}
		return errors.New(reason)
	}
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	values, err := s.Paths.Load()
	if err != nil {
		values = settings.Defaults()
	}
	game, ok := values["game"].(map[string]any)
	if !ok {
		game = map[string]any{}
		values["game"] = game
	}
	path := s.Paths.ISOPath(value)
	stored := path
	if relative, err := filepath.Rel(s.Paths.ISODir, path); err == nil &&
		relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		stored = relative
	}
	game["iso"] = stored
	if err := s.Paths.Save(values); err != nil {
		return err
	}
	state := s.setupStatusLocked()
	state.GameReady = true
	state.ISO = stored
	return s.persistSetupLocked(state)
}

func (s *Server) markAccount(label string) error {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()
	state := s.setupStatusLocked()
	state.AccountReady = true
	state.Account = label
	return s.persistSetupLocked(state)
}

func (s *Server) knownISO(path string) bool {
	path = filepath.Clean(path)
	for _, iso := range s.Paths.FindISOs() {
		if iso.Valid && filepath.Clean(iso.Path) == path {
			return true
		}
	}
	return false
}

func (s *Server) renderSetup(w http.ResponseWriter, code int, message, problem string) {
	state := s.setupStatus()
	isos := []settings.ISO{}
	for _, iso := range s.Paths.FindISOs() {
		if iso.Valid {
			isos = append(isos, iso)
		}
	}
	data := setupPage{
		Token:        s.Token,
		GameReady:    state.GameReady,
		AccountReady: state.AccountReady,
		Done:         state.Done,
		GameName:     filepath.Base(state.ISO),
		Account:      state.Account,
		ISOs:         isos,
		Message:      message,
		Error:        problem,
	}
	var rendered strings.Builder
	if err := setupTemplate.Execute(&rendered, data); err != nil {
		s.sendHTML(w, http.StatusInternalServerError, "setup page error")
		return
	}
	s.sendHTML(w, code, rendered.String())
}

func (s *Server) setupUpload(w http.ResponseWriter, request *http.Request) {
	name, ok := s.receiveISO(w, request)
	if !ok {
		return
	}
	if err := s.selectGame(name); err != nil {
		s.sendText(w, http.StatusInternalServerError, "Could not save the game: "+err.Error())
		return
	}
	s.sendText(w, http.StatusOK, fmt.Sprintf("Added %s.", name))
}

func (s *Server) setupSelect(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, maxFormBytes)
	if err := request.ParseForm(); err != nil {
		s.renderSetup(w, http.StatusBadRequest, "", "Invalid game selection.")
		return
	}
	path := request.PostFormValue("path")
	if !s.knownISO(path) {
		s.renderSetup(w, http.StatusBadRequest, "", "That game image is no longer available.")
		return
	}
	if err := s.selectGame(path); err != nil {
		s.renderSetup(w, http.StatusInternalServerError, "", err.Error())
		return
	}
	http.Redirect(w, request, "/"+s.Token+"/", http.StatusSeeOther)
}

func (s *Server) setupAuthenticate(w http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(w, request.Body, maxFormBytes)
	if err := request.ParseForm(); err != nil {
		s.renderSetup(w, http.StatusBadRequest, "", "Invalid sign-in form.")
		return
	}
	user, err := s.Login(
		strings.TrimSpace(request.PostFormValue("email")),
		request.PostFormValue("password"),
	)
	if err != nil {
		s.renderSetup(w, http.StatusBadRequest, "", err.Error())
		return
	}
	if err := s.installUser(user); err != nil {
		s.renderSetup(w, http.StatusInternalServerError, "", err.Error())
		return
	}
	if err := s.markAccount(user.ConnectCode); err != nil {
		s.renderSetup(w, http.StatusInternalServerError, "", err.Error())
		return
	}
	http.Redirect(w, request, "/"+s.Token+"/", http.StatusSeeOther)
}

func (s *Server) setupSkip(w http.ResponseWriter, request *http.Request) {
	if err := s.markAccount("offline"); err != nil {
		s.renderSetup(w, http.StatusInternalServerError, "", err.Error())
		return
	}
	http.Redirect(w, request, "/"+s.Token+"/", http.StatusSeeOther)
}

func (s *Server) serveSetup(w http.ResponseWriter, request *http.Request, parts []string) {
	if request.Method == http.MethodGet {
		if len(parts) != 0 {
			http.NotFound(w, request)
			return
		}
		s.renderSetup(w, http.StatusOK, "", "")
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, request)
		return
	}
	switch request.Method {
	case http.MethodPost:
		switch parts[0] {
		case "iso":
			s.setupUpload(w, request)
		case "select":
			s.setupSelect(w, request)
		case "auth":
			s.setupAuthenticate(w, request)
		case "skip":
			s.setupSkip(w, request)
		default:
			http.NotFound(w, request)
		}
	case http.MethodPut:
		if parts[0] != "iso" {
			http.NotFound(w, request)
			return
		}
		s.setupUpload(w, request)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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
	if s.Mode == "setup" {
		s.serveSetup(w, request, parts)
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

func (s *Server) installUser(user auth.User) error {
	if err := os.MkdirAll(s.Paths.ProfileDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return err
	}
	target := filepath.Join(s.Paths.ProfileDir, settings.DefaultUserJSON)
	if err := os.WriteFile(target, data, 0o600); err != nil {
		return err
	}
	_ = os.Chmod(target, 0o600)
	installed, detail := s.Paths.InstallLogin(settings.Defaults())
	if !installed {
		return errors.New(detail)
	}
	return nil
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

	if err := s.installUser(user); err != nil {
		s.sendHTML(w, http.StatusInternalServerError, s.page(err.Error(), true))
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

func (s *Server) receiveISO(w http.ResponseWriter, request *http.Request) (string, bool) {
	name := filepath.Base(request.URL.Query().Get("name"))
	if name == "" || name == "." {
		name = "upload.iso"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".iso") {
		s.sendText(w, http.StatusBadRequest, "Only .iso files are accepted.")
		return "", false
	}

	length := request.ContentLength
	if length <= 0 || length > MaxUpload {
		s.sendText(w, http.StatusRequestEntityTooLarge, "Missing or oversized upload.")
		return "", false
	}

	if err := os.MkdirAll(s.Paths.ISODir, 0o755); err != nil {
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return "", false
	}
	partial := filepath.Join(s.Paths.ISODir, name+".part")
	destination := filepath.Join(s.Paths.ISODir, name)

	file, err := os.OpenFile(partial, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return "", false
	}
	written, copyErr := io.CopyN(file, request.Body, length)
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || written != length {
		_ = os.Remove(partial)
		if errors.Is(copyErr, io.EOF) || errors.Is(copyErr, io.ErrUnexpectedEOF) || copyErr == nil {
			s.sendText(w, http.StatusBadRequest, "Upload incomplete.")
			return "", false
		}
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+copyErr.Error())
		return "", false
	}

	valid, reason := s.Paths.ISOStatus(filepath.Base(partial))
	if !valid {
		_ = os.Remove(partial)
		s.sendText(w, http.StatusBadRequest, reason)
		return "", false
	}
	if err := os.Rename(partial, destination); err != nil {
		_ = os.Remove(partial)
		s.sendText(w, http.StatusInsufficientStorage, "Could not store the image: "+err.Error())
		return "", false
	}
	_ = os.Chmod(destination, 0o444)
	s.writeMarker(name, destination)
	return name, true
}

func (s *Server) upload(w http.ResponseWriter, request *http.Request) {
	name, ok := s.receiveISO(w, request)
	if !ok {
		return
	}
	s.sendText(w, http.StatusOK, fmt.Sprintf("Added %s. Return to the console.", name))
}

// Start listens on the configured port and publishes the LAN URL.
func (s *Server) Start() (string, error) {
	_ = os.MkdirAll(filepath.Dir(s.Status), 0o755)
	_ = os.Remove(s.Marker)
	if s.Mode == "setup" {
		s.refreshSetup()
	}

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
