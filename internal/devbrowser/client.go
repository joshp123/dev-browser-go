package devbrowser

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type DaemonState struct {
	PID        int    `json:"pid"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Profile    string `json:"profile"`
	CDPPort    int    `json:"cdpPort"`
	WSEndpoint string `json:"wsEndpoint"`
}

func ReadState(profile string) (*DaemonState, error) {
	path := StateFile(profile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var state DaemonState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func DaemonBaseURL(profile string) string {
	state, err := ReadState(profile)
	if err != nil || state == nil {
		return ""
	}
	if state.Host == "" || state.Port == 0 {
		return ""
	}
	return fmt.Sprintf("http://%s:%d", state.Host, state.Port)
}

func HTTPJSON(method string, url string, body map[string]any, timeout time.Duration) (map[string]any, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func IsDaemonHealthy(profile string) bool {
	base := DaemonBaseURL(profile)
	if base == "" {
		return false
	}
	data, err := HTTPJSON(http.MethodGet, base+"/health", nil, 1500*time.Millisecond)
	if err != nil {
		return false
	}
	ok, _ := data["ok"].(bool)
	return ok
}

func StartDaemon(profile string, headless bool, window *WindowSize, device string) error {
	if IsDaemonHealthy(profile) {
		return nil
	}
	if window != nil && strings.TrimSpace(device) != "" {
		return errors.New("use either --window-size/--window-scale or --device")
	}

	dir := StateDir(profile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(dir, "daemon.log")

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"--daemon", "--profile", profile}
	if headless {
		args = append(args, "--headless")
	}
	if window != nil {
		args = append(args, "--window-size", fmt.Sprintf("%dx%d", window.Width, window.Height))
	}
	if strings.TrimSpace(device) != "" {
		args = append(args, "--device", device)
	}

	cmd := exec.Command(exe, args...)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return err
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if IsDaemonHealthy(profile) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for dev-browser daemon (profile=%s). See %s", profile, logPath)
}

func StopDaemon(profile string) (bool, error) {
	state, err := ReadState(profile)
	base := DaemonBaseURL(profile)
	if state == nil || base == "" || err != nil {
		return false, nil
	}

	_, _ = HTTPJSON(http.MethodPost, base+"/shutdown", map[string]any{}, 3*time.Second)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !IsDaemonHealthy(profile) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if IsDaemonHealthy(profile) && state.PID > 0 {
		_ = syscall.Kill(state.PID, syscall.SIGTERM)
		time.Sleep(200 * time.Millisecond)
	}

	_ = os.Remove(StateFile(profile))
	return true, nil
}

func EnsurePage(profile string, headless bool, page string, window *WindowSize, device string) (string, string, error) {
	if err := StartDaemon(profile, headless, window, device); err != nil {
		return "", "", err
	}
	base := DaemonBaseURL(profile)
	if base == "" {
		return "", "", errors.New("daemon state missing after start")
	}
	data, err := HTTPJSON(http.MethodPost, base+"/pages", map[string]any{"name": page}, 10*time.Second)
	if err != nil {
		return "", "", err
	}
	ws, _ := data["wsEndpoint"].(string)
	tid, _ := data["targetId"].(string)
	if strings.TrimSpace(ws) == "" {
		return "", "", errors.New("daemon did not return wsEndpoint")
	}
	if strings.TrimSpace(tid) == "" {
		return "", "", errors.New("daemon did not return targetId")
	}
	return ws, tid, nil
}

func WriteOutput(profile string, mode string, result map[string]any, outPath string) (string, error) {
	switch mode {
	case "json":
		enc, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(enc), nil
	case "summary":
		if snap, ok := result["snapshot"].(string); ok {
			return snap, nil
		}
		if path, ok := result["path"].(string); ok {
			return path, nil
		}
		enc, err := json.Marshal(result)
		if err != nil {
			return "", err
		}
		return string(enc), nil
	case "path":
		path, err := SafeArtifactPath(ArtifactDir(profile), outPath, fmt.Sprintf("cli-%d.json", NowMS()))
		if err != nil {
			return "", err
		}
		enc, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, enc, 0o644); err != nil {
			return "", err
		}
		return path, nil
	default:
		return "", fmt.Errorf("unknown output mode: %s", mode)
	}
}
