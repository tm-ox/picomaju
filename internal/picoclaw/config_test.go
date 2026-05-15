package picoclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBinaryPath(t *testing.T) {
	got := BinaryPath("/data")
	want := filepath.Join("/data", "bin", "picoclaw")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlatformZip_Format(t *testing.T) {
	name := platformZip()
	if !strings.HasPrefix(name, "picoclaw-") || !strings.HasSuffix(name, ".zip") {
		t.Errorf("unexpected platform zip name: %q", name)
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		AgentID:      "agent-1",
		WorkspaceDir: "/ws/agent-1",
		Tools: []ToolConfig{
			{Type: "whatsapp", Config: map[string]any{"token": "abc"}},
		},
		LLMProxy: &LLMProxyConfig{URL: "http://localhost:18800/proxy", Token: "tok"},
	}
	if err := WriteConfig(cfg, dir); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	var out Config
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q", out.AgentID)
	}
	if out.LLMProxy == nil || out.LLMProxy.Token != "tok" {
		t.Errorf("LLMProxy not preserved: %+v", out.LLMProxy)
	}
	if len(out.Tools) != 1 || out.Tools[0].Type != "whatsapp" {
		t.Errorf("Tools not preserved: %+v", out.Tools)
	}
}

func TestWriteConfig_FileMode(t *testing.T) {
	dir := t.TempDir()
	WriteConfig(Config{AgentID: "a1"}, dir)
	info, err := os.Stat(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected mode 0600, got %o", perm)
	}
}

func TestManager_IsRunning_InitiallyFalse(t *testing.T) {
	m := NewManager()
	if m.IsRunning("any-agent") {
		t.Error("new manager should have no running processes")
	}
}

func TestManager_StopAll_Empty(t *testing.T) {
	m := NewManager()
	m.StopAll() // should not panic
}
