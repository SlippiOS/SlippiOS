package netpages

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
	return name, true
}

func (s *Server) upload(w http.ResponseWriter, request *http.Request) {
	name, ok := s.receiveISO(w, request)
	if !ok {
		return
	}
	s.sendText(w, http.StatusOK, fmt.Sprintf("Added %s. Return to the console.", name))
	// Write the marker after the response so the shell cannot stop the
	// server before the browser sees a successful upload.
	s.writeMarker(name, filepath.Join(s.Paths.ISODir, name))
}
