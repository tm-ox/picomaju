package picoclaw

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// makeZip returns the bytes of a zip containing one file named "picoclaw".
func makeZip(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("picoclaw")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	f.Write(content)
	w.Close()
	return buf.Bytes()
}

// redirectTransport sends every request to the given base URL, preserving path.
type redirectTransport struct {
	base   *url.URL
	client *http.Client
}

func (rt *redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r2 := r.Clone(r.Context())
	r2.URL = rt.base
	return rt.client.Transport.RoundTrip(r2)
}

func newRedirectClient(srv *httptest.Server) *http.Client {
	u, _ := url.Parse(srv.URL)
	return &http.Client{Transport: &redirectTransport{base: u, client: srv.Client()}}
}

// fakeBinary writes a shell script that sleeps indefinitely at the expected binary path.
func fakeBinary(t *testing.T, dataDir string) {
	t.Helper()
	bin := BinaryPath(dataDir)
	os.MkdirAll(filepath.Dir(bin), 0755)
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 60\n"), 0755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
}

// ── EnsureBinary ─────────────────────────────────────────────────────────────

func TestEnsureBinary_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	fakeBinary(t, dir)

	m := NewManager()
	if err := m.EnsureBinary(dir, "0.1.0"); err != nil {
		t.Errorf("expected no error when binary exists: %v", err)
	}
}

func TestEnsureBinary_DownloadNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	m := NewManager()
	m.httpClient = newRedirectClient(srv)

	if err := m.EnsureBinary(t.TempDir(), "0.1.0"); err == nil {
		t.Error("expected error on non-200 response")
	}
}

func TestEnsureBinary_DownloadAndExtract(t *testing.T) {
	zipData := makeZip(t, []byte("#!/bin/sh\nsleep 60\n"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.WriteHeader(http.StatusOK)
		w.Write(zipData)
	}))
	defer srv.Close()

	m := NewManager()
	m.httpClient = newRedirectClient(srv)

	dir := t.TempDir()
	if err := m.EnsureBinary(dir, "0.1.0"); err != nil {
		t.Fatalf("EnsureBinary: %v", err)
	}
	if _, err := os.Stat(BinaryPath(dir)); err != nil {
		t.Errorf("binary not written after download: %v", err)
	}
}

// ── Start / Stop / IsRunning ──────────────────────────────────────────────────

func TestManager_Stop_NoOp(t *testing.T) {
	m := NewManager()
	m.Stop("nonexistent") // must not panic
}

func TestManager_StartStop(t *testing.T) {
	dir := t.TempDir()
	fakeBinary(t, dir)
	wsDir := t.TempDir()

	m := NewManager()
	if err := m.Start("agent1", wsDir, dir); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !m.IsRunning("agent1") {
		t.Error("expected agent1 to be running after Start")
	}

	m.Stop("agent1")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !m.IsRunning("agent1") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("agent1 still running after Stop")
}

func TestManager_StopAll(t *testing.T) {
	dir := t.TempDir()
	fakeBinary(t, dir)
	wsDir := t.TempDir()

	m := NewManager()
	m.Start("a1", wsDir, dir)
	m.Start("a2", wsDir, dir)
	m.StopAll()

	if m.IsRunning("a1") || m.IsRunning("a2") {
		t.Error("expected all agents stopped after StopAll")
	}
}

func TestManager_StartReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	fakeBinary(t, dir)
	wsDir := t.TempDir()

	m := NewManager()
	m.Start("agent1", wsDir, dir)
	m.Start("agent1", wsDir, dir) // second Start kills the first

	if !m.IsRunning("agent1") {
		t.Error("agent1 should still be running after restart")
	}
	m.Stop("agent1")
}
