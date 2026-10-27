package netpages

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/SlippiOS/SlippiOS/backend/internal/auth"
	"github.com/SlippiOS/SlippiOS/backend/internal/settings"
)

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
