package picoclaw

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

// makeTarGz returns the bytes of a .tar.gz containing one file named "picoclaw".
func makeTarGz(t *testing.T, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: "picoclaw",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write: %v", err)
	}
	tw.Close()
	gw.Close()
	return buf.Bytes()
}

// redirectTransport sends every request to the given base URL, preserving nothing.
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
	if err := m.EnsureBinary(dir, DefaultVersion); err != nil {
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

	if err := m.EnsureBinary(t.TempDir(), DefaultVersion); err == nil {
		t.Error("expected error on non-200 response")
	}
}

func TestEnsureBinary_DownloadAndExtract(t *testing.T) {
	content := []byte("#!/bin/sh\nsleep 60\n")
	asset := platformAsset()
	var archiveData []byte
	if strings.HasSuffix(asset, ".tar.gz") {
		archiveData = makeTarGz(t, content)
	} else {
		archiveData = makeZip(t, content)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(archiveData)
	}))
	defer srv.Close()

	m := NewManager()
	m.httpClient = newRedirectClient(srv)

	dir := t.TempDir()
	if err := m.EnsureBinary(dir, DefaultVersion); err != nil {
		t.Fatalf("EnsureBinary: %v", err)
	}
	if _, err := os.Stat(BinaryPath(dir)); err != nil {
		t.Errorf("binary not written after download: %v", err)
	}
}

func TestExtractZip_WritesBinary(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picoclaw")
	content := []byte("#!/bin/sh\necho hi\n")
	zipData := makeZip(t, content)

	tmp, err := os.CreateTemp(dir, "*.zip")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Write(zipData)
	tmp.Close()

	if err := extractZip(tmp.Name(), dest); err != nil {
		t.Fatalf("extractZip: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q", got)
	}
}

func TestExtractZip_EmptyArchive(t *testing.T) {
	dir := t.TempDir()
	// Build a zip with no files (only directories).
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	w.Create("somedir/") //nolint:errcheck
	w.Close()

	tmp, err := os.CreateTemp(dir, "*.zip")
	if err != nil {
		t.Fatal(err)
	}
	tmp.Write(buf.Bytes())
	tmp.Close()

	if err := extractZip(tmp.Name(), filepath.Join(dir, "out")); err == nil {
		t.Error("expected error for zip with no binary")
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
