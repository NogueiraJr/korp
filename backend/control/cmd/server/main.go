// Control is a lightweight supervisor that starts and stops the estoque and
// faturamento microservices on demand, exposing an HTTP API consumed by the
// Angular frontend. It only manages child processes it has spawned itself.
package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	startTimeout = 12 * time.Second
	stopTimeout  = 6 * time.Second
)

type service struct {
	name    string
	port    string
	bin     string
	logFile string

	mu      sync.Mutex
	cmd     *exec.Cmd
	running bool
	doneCh  chan struct{}
}

func (s *service) isRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *service) start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	if _, err := os.Stat(s.bin); err != nil {
		return err
	}

	logf, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	cmd := exec.Command(s.bin)
	cmd.Env = append(os.Environ(), "PORT="+s.port)
	cmd.Dir = filepath.Dir(s.bin)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		logf.Close()
		return err
	}

	s.mu.Lock()
	s.cmd = cmd
	s.running = true
	s.doneCh = make(chan struct{})
	doneCh := s.doneCh
	s.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		s.running = false
		s.cmd = nil
		s.mu.Unlock()
		close(doneCh)
		_ = logf.Close()
	}()

	return s.waitHealthy()
}

func (s *service) stop() error {
	s.mu.Lock()
	if !s.running || s.cmd == nil {
		s.mu.Unlock()
		return nil
	}
	pid := s.cmd.Process.Pid
	doneCh := s.doneCh
	s.mu.Unlock()

	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-doneCh:
		return nil
	case <-time.After(stopTimeout):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		select {
		case <-doneCh:
			return nil
		case <-time.After(2 * time.Second):
			return &stopError{name: s.name}
		}
	}
}

func (s *service) waitHealthy() error {
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		if !s.isRunning() {
			return &startError{name: s.name, reason: "processo encerrou inesperadamente"}
		}
		conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", s.port), 500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return &startError{name: s.name, reason: "timeout aguardando a porta"}
}

type startError struct {
	name   string
	reason string
}

func (e *startError) Error() string { return e.name + ": " + e.reason }

type stopError struct {
	name string
}

func (e *stopError) Error() string { return "falha ao encerrar " + e.name }

// defaultBin resolves the sibling service binary path relative to the control
// executable, e.g. <backend>/<name>/bin/<name>.
func defaultBin(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join("..", "..", name, "bin", name)
	}
	binDir := filepath.Dir(exe)                  // <backend>/control/bin
	backendDir := filepath.Dir(filepath.Dir(binDir)) // <backend>
	return filepath.Join(backendDir, name, "bin", name)
}

func main() {
	port := getEnv("CONTROL_PORT", "8080")
	svcs := map[string]*service{
		"estoque": {
			name:    "estoque",
			port:    getEnv("ESTOQUE_PORT", "8081"),
			bin:     getEnv("ESTOQUE_BIN", defaultBin("estoque")),
			logFile: getEnv("ESTOQUE_LOG", "/tmp/estoque.log"),
		},
		"faturamento": {
			name:    "faturamento",
			port:    getEnv("FATURAMENTO_PORT", "8082"),
			bin:     getEnv("FATURAMENTO_BIN", defaultBin("faturamento")),
			logFile: getEnv("FATURAMENTO_LOG", "/tmp/faturamento.log"),
		},
	}

	for _, s := range svcs {
		if err := s.start(); err != nil {
			log.Printf("[control] erro ao iniciar %s: %v", s.name, err)
		} else {
			log.Printf("[control] %s iniciado na porta %s", s.name, s.port)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "control"})
	})
	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"estoque":     map[string]any{"running": svcs["estoque"].isRunning()},
			"faturamento": map[string]any{"running": svcs["faturamento"].isRunning()},
		})
	})
	mux.HandleFunc("POST /api/estoque/start", actionHandler(svcs["estoque"]))
	mux.HandleFunc("POST /api/estoque/stop", actionHandler(svcs["estoque"]))
	mux.HandleFunc("POST /api/faturamento/start", actionHandler(svcs["faturamento"]))
	mux.HandleFunc("POST /api/faturamento/stop", actionHandler(svcs["faturamento"]))

	server := &http.Server{
		Addr:    ":" + port,
		Handler: cors(mux),
	}

	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("[control] listen %s: %v", port, err)
	}
	log.Printf("[control] listening on :%s", port)
	if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("[control] server error: %v", err)
	}
}

func actionHandler(s *service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		switch {
		case r.Method == http.MethodPost && hasSuffix(r.URL.Path, "/start"):
			if s.isRunning() {
				err = nil
			} else {
				err = s.start()
			}
		case r.Method == http.MethodPost && hasSuffix(r.URL.Path, "/stop"):
			err = s.stop()
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"ok": false, "running": s.isRunning(), "error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "running": s.isRunning()})
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:4200")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}