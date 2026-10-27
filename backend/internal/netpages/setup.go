package netpages

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

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
