package picoclaw

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const githubRepo = "sipeed/picoclaw"

// DefaultVersion is used when PICOCLAW_VERSION env var is not set.
const DefaultVersion = "0.2.8"

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

// platformAsset returns the release asset filename for the current OS/arch.
// Linux and macOS use .tar.gz; Windows and Android use .zip.
func platformAsset() string {
	if runtime.GOOS == "android" {
		return "picoclaw-android-universal.zip"
	}
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("picoclaw_Windows_%s.zip", arch)
	case "darwin":
		return fmt.Sprintf("picoclaw_Darwin_%s.tar.gz", arch)
	default:
		return fmt.Sprintf("picoclaw_Linux_%s.tar.gz", arch)
	}
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
	asset := platformAsset()
	url := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", githubRepo, version, asset) //nolint:gosec
	resp, err := m.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download picoclaw: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download picoclaw: %s", resp.Status)
	}
	ext := ".zip"
	if strings.HasSuffix(asset, ".tar.gz") {
		ext = ".tar.gz"
	}
	tmp, err := os.CreateTemp("", "picoclaw-*"+ext)
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()
	if ext == ".tar.gz" {
		return extractTarGz(tmp.Name(), bin)
	}
	return extractZip(tmp.Name(), bin)
}

func extractZip(archivePath, dest string) error {
	r, err := zip.OpenReader(archivePath)
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

func extractTarGz(archivePath, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		out.Close()
		return copyErr
	}
	return fmt.Errorf("no binary found in archive")
}

// Start launches picoclaw gateway for the given staff member.
// Kills any existing process for staffID first.
// workspaceDir is unused by the CLI but kept for API compatibility.
func (m *Manager) Start(staffID, workspaceDir, dataDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.processes[staffID]; ok {
		_ = p.Kill()
	}
	cmd := exec.Command(BinaryPath(dataDir), "gateway")
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
