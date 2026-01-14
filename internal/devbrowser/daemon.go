package devbrowser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type DaemonOptions struct {
	Profile   string
	Host      string
	Port      int
	CDPPort   int
	Headless  bool
	Window    *WindowSize
	Device    string
	StateFile string
	Logger    *log.Logger
}

type Daemon struct {
	opts   DaemonOptions
	host   *BrowserHost
	server *http.Server
	logger *log.Logger
}

func ServeDaemon(opts DaemonOptions) error {
	logger := opts.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}
	profile := opts.Profile
	if profile == "" {
		profile = "default"
	}
	cdpPort := opts.CDPPort
	if cdpPort == 0 {
		p, err := chooseFreePort()
		if err != nil {
			return err
		}
		cdpPort = p
	}

	host := NewBrowserHost(profile, opts.Headless, cdpPort, opts.Window, opts.Device)
	if err := host.Start(); err != nil {
		return err
	}
	defer host.Stop()
	ws, err := host.WSEndpoint()
	if err != nil {
		return err
	}

	stateFile := opts.StateFile
	if strings.TrimSpace(stateFile) == "" {
		stateFile = filepath.Join(StateDir(profile), "daemon.json")
	}
	if err := os.MkdirAll(filepath.Dir(stateFile), 0o755); err != nil {
		return err
	}

	mux := http.NewServeMux()
	d := &Daemon{opts: opts, host: host, logger: logger}

	mux.HandleFunc("/health", d.handleHealth)
	mux.HandleFunc("/", d.handleRoot)
	mux.HandleFunc("/pages", d.handlePages)
	mux.HandleFunc("/pages/", d.handlePageSubresource)
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		d.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		go func() {
			time.Sleep(50 * time.Millisecond)
			_ = d.server.Shutdown(context.Background())
		}()
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", opts.Host, opts.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	d.server = srv

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}

	if err := writeStateFile(stateFile, map[string]any{
		"pid":        os.Getpid(),
		"host":       opts.Host,
		"port":       ln.Addr().(*net.TCPAddr).Port,
		"profile":    profile,
		"cdpPort":    cdpPort,
		"wsEndpoint": ws,
	}); err != nil {
		ln.Close()
		return err
	}
	defer os.Remove(stateFile)

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (d *Daemon) handleHealth(w http.ResponseWriter, r *http.Request) {
	ws, _ := d.host.WSEndpoint()
	d.writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"pid":        os.Getpid(),
		"profile":    d.opts.Profile,
		"wsEndpoint": ws,
		"version":    "0.1.0-go",
	})
}

func (d *Daemon) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
		return
	}
	ws, err := d.host.WSEndpoint()
	if err != nil {
		d.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	d.writeJSON(w, http.StatusOK, map[string]any{"wsEndpoint": ws})
}

func (d *Daemon) handlePages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		d.writeJSON(w, http.StatusOK, map[string]any{"pages": d.host.ListPages()})
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "name is required and must be a non-empty string"})
			return
		}
		entry, err := d.host.GetOrCreatePage(strings.TrimSpace(body.Name))
		if err != nil {
			d.writeJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		ws, _ := d.host.WSEndpoint()
		d.writeJSON(w, http.StatusOK, map[string]any{"wsEndpoint": ws, "name": entry.Name, "targetId": entry.TargetID})
	default:
		d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
	}
}

func (d *Daemon) handlePageSubresource(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/pages/")
	if rest == "" {
		d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "name required"})
		return
	}
	parts := strings.Split(rest, "/")
	name, err := url.PathUnescape(parts[0])
	if err != nil || strings.TrimSpace(name) == "" {
		d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid page name"})
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodDelete {
			d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
			return
		}
		if closed := d.host.ClosePage(name); !closed {
			d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "page not found"})
			return
		}
		d.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	if len(parts) != 2 || parts[1] != "console" {
		d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
		return
	}
	if r.Method != http.MethodGet {
		d.writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
		return
	}

	query := r.URL.Query()
	since := int64(0)
	if raw := strings.TrimSpace(query.Get("since")); raw != "" {
		val, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || val < 0 {
			d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid since"})
			return
		}
		since = val
	}
	limit := defaultConsoleLogMax
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		val, err := strconv.Atoi(raw)
		if err != nil || val < 0 {
			d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid limit"})
			return
		}
		limit = val
	}

	levelFilter, err := parseConsoleLevels(query.Get("levels"))
	if err != nil {
		d.writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	logs, lastID, err := d.host.ConsoleLogs(name, since, 0)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "page not found") {
			status = http.StatusNotFound
		}
		d.writeJSON(w, status, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	logs = selectConsoleLogs(logs, levelFilter, since, limit)
	if len(logs) > 0 {
		lastID = logs[len(logs)-1].ID
	}

	d.writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"page":    name,
		"since":   since,
		"limit":   limit,
		"last_id": lastID,
		"logs":    logs,
	})
}

func selectConsoleLogs(logs []ConsoleEntry, filter consoleLevelFilter, since int64, limit int) []ConsoleEntry {
	entries := filterConsoleEntries(logs, filter)
	if limit <= 0 || len(entries) <= limit {
		return entries
	}
	if since > 0 {
		return entries[:limit]
	}
	return entries[len(entries)-limit:]
}

func (d *Daemon) writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func chooseFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func writeStateFile(path string, data map[string]any) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, path)
}
