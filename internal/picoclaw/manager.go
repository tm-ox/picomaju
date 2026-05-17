package picoclaw

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

const githubRepo = "picomaju/picoclaw"

// DefaultVersion is used when PICOCLAW_VERSION env var is not set.
const DefaultVersion = "0.1.0"

// Manager tracks running picoclaw subprocesses keyed by staff ID.
type Manager struct {
	mu         sync.RWMutex
	processes  map[string]*os.Process
	httpClient *http.Client
}

func NewManager() *Manager {
	return &Manager{
		processes:  make(map[string]*os.Process),
		httpClient: &http.Client{},
	}
}

// BinaryPath returns the path to the picoclaw binary within dataDir.
func BinaryPath(dataDir string) string {
	return filepath.Join(dataDir, "bin", "picoclaw")
}

func platformZip() string {
	if runtime.GOOS == "android" || runtime.GOARCH == "arm64" {
		return "picoclaw-android-universal.zip"
	}
	return fmt.Sprintf("picoclaw-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
}

// EnsureBinary downloads and extracts the picoclaw binary if not present.
func (m *Manager) EnsureBinary(dataDir, version string) error {
	bin := BinaryPath(dataDir)
	if _, err := os.Stat(bin); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(bin), 0755); err != nil {
		return err
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", githubRepo, version, platformZip())
	resp, err := m.httpClient.Get(url) //nolint:gosec
	if err != nil {
		return fmt.Errorf("download picoclaw: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download picoclaw: %s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "picoclaw-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()
	return extractBinary(tmp.Name(), bin)
}

func extractBinary(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return err
		}
		_, copyErr := io.Copy(out, rc)
		rc.Close()
		out.Close()
		return copyErr
	}
	return fmt.Errorf("no binary found in zip")
}

// Start launches picoclaw for the given staff member. Kills any existing process first.
func (m *Manager) Start(staffID, workspaceDir, dataDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.processes[staffID]; ok {
		_ = p.Kill()
	}
	cmd := exec.Command(BinaryPath(dataDir), "--config", filepath.Join(workspaceDir, "config.json"))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start picoclaw: %w", err)
	}
	proc := cmd.Process
	m.processes[staffID] = proc
	go func() {
		_ = cmd.Wait()
		m.mu.Lock()
		if m.processes[staffID] == proc {
			delete(m.processes, staffID)
		}
		m.mu.Unlock()
	}()
	return nil
}

// Stop kills the picoclaw process for the given staff member.
func (m *Manager) Stop(staffID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.processes[staffID]; ok {
		_ = p.Kill()
		delete(m.processes, staffID)
	}
}

// IsRunning reports whether a picoclaw process is active for staffID.
func (m *Manager) IsRunning(staffID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.processes[staffID]
	return ok
}

// StopAll kills all managed processes (call on shutdown).
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, p := range m.processes {
		_ = p.Kill()
		delete(m.processes, id)
	}
}
