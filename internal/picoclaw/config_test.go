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

func TestPlatformAsset_Format(t *testing.T) {
	name := platformAsset()
	if !strings.HasPrefix(name, "picoclaw") {
		t.Errorf("unexpected asset name: %q", name)
	}
	if !strings.HasSuffix(name, ".zip") && !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("expected .zip or .tar.gz, got %q", name)
	}
}

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	cfg := Config{
		Version: 2,
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Workspace: "/ws/agent-1",
				ModelName: "picomaju-proxy",
			},
		},
		ModelList: []ModelEntry{
			{ModelName: "picomaju-proxy", Model: "openai/claude-sonnet-4-5", APIKeys: []string{"tok"}, APIBase: "http://localhost:18800/proxy/v1"},
		},
		Gateway: GatewayConfig{Host: "127.0.0.1", Port: 18790, LogLevel: "warn"},
	}
	if err := WriteConfig(cfg, dest); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}
	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	var out Config
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out.Version != 2 {
		t.Errorf("Version: got %d", out.Version)
	}
	if out.Agents.Defaults.ModelName != "picomaju-proxy" {
		t.Errorf("ModelName: got %q", out.Agents.Defaults.ModelName)
	}
	if len(out.ModelList) != 1 || out.ModelList[0].APIKeys[0] != "tok" {
		t.Errorf("ModelList not preserved: %+v", out.ModelList)
	}
}

func TestWriteConfig_FileMode(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	WriteConfig(Config{Version: 2}, dest)
	info, err := os.Stat(dest)
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
