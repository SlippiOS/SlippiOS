package netpages

import (
	"encoding/json"
	"errors"
	"fmt"
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

func LocalAddress() string {
	if address := interfaceAddress(); address != "" {
		return address
	}
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

func interfaceAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch value := address.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip == nil || ip.To4() == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
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
