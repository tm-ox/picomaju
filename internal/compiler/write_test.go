package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	out := Output{AgentMD: "# Agent", SoulMD: "# Soul", UserMD: "# User"}
	if err := Write(out, dir); err != nil {
		t.Fatalf("Write: %v", err)
	}
	for _, name := range []string{"AGENT.md", "SOUL.md", "USER.md"} {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("missing %s: %v", name, err)
			continue
		}
		if !strings.Contains(string(b), "#") {
			t.Errorf("%s appears empty", name)
		}
	}
}

func TestWrite_CreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "workspace")
	out := Output{AgentMD: "a", SoulMD: "b", UserMD: "c"}
	if err := Write(out, dir); err != nil {
		t.Fatalf("Write should create nested dir: %v", err)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected dir to exist: %v", err)
	}
}

func TestWrite_CorrectContent(t *testing.T) {
	dir := t.TempDir()
	out := Output{AgentMD: "agent content", SoulMD: "soul content", UserMD: "user content"}
	Write(out, dir)
	b, _ := os.ReadFile(filepath.Join(dir, "SOUL.md"))
	if string(b) != "soul content" {
		t.Errorf("SOUL.md content mismatch: %q", string(b))
	}
}

func TestWriteUserMD(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the dir (WriteUserMD does not create it)
	if err := WriteUserMD("# Updated User", dir); err != nil {
		t.Fatalf("WriteUserMD: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "USER.md"))
	if err != nil {
		t.Fatalf("read USER.md: %v", err)
	}
	if string(b) != "# Updated User" {
		t.Errorf("unexpected content: %q", string(b))
	}
}

func TestInjectConfig_NewFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	entry := AgentEntry{ID: "a1", Workspace: "/ws/a1", Description: "Support"}
	if err := InjectConfig(path, entry); err != nil {
		t.Fatalf("InjectConfig: %v", err)
	}
	b, _ := os.ReadFile(path)
	var root map[string]json.RawMessage
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := root["agents"]; !ok {
		t.Error("expected agents key in config")
	}
}

func TestInjectConfig_AddsToExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	InjectConfig(path, AgentEntry{ID: "a1", Workspace: "/ws/a1"})
	if err := InjectConfig(path, AgentEntry{ID: "a2", Workspace: "/ws/a2"}); err != nil {
		t.Fatalf("InjectConfig: %v", err)
	}
	b, _ := os.ReadFile(path)
	var root struct {
		Agents struct {
			List []AgentEntry `json:"list"`
		} `json:"agents"`
	}
	json.Unmarshal(b, &root)
	if len(root.Agents.List) != 2 {
		t.Errorf("expected 2 agents, got %d", len(root.Agents.List))
	}
}

func TestInjectConfig_UpdatesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	InjectConfig(path, AgentEntry{ID: "a1", Workspace: "/ws/old"})
	if err := InjectConfig(path, AgentEntry{ID: "a1", Workspace: "/ws/new"}); err != nil {
		t.Fatalf("InjectConfig: %v", err)
	}
	b, _ := os.ReadFile(path)
	var root struct {
		Agents struct {
			List []AgentEntry `json:"list"`
		} `json:"agents"`
	}
	json.Unmarshal(b, &root)
	if len(root.Agents.List) != 1 {
		t.Errorf("expected 1 agent after upsert, got %d", len(root.Agents.List))
	}
	if root.Agents.List[0].Workspace != "/ws/new" {
		t.Errorf("expected updated workspace, got %q", root.Agents.List[0].Workspace)
	}
}

func TestInjectConfig_PreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte(`{"version":"1","other":"data"}`), 0644)
	InjectConfig(path, AgentEntry{ID: "a1", Workspace: "/ws/a1"})
	b, _ := os.ReadFile(path)
	var root map[string]json.RawMessage
	json.Unmarshal(b, &root)
	if _, ok := root["version"]; !ok {
		t.Error("existing keys should be preserved")
	}
	if _, ok := root["other"]; !ok {
		t.Error("existing keys should be preserved")
	}
}

func TestInjectConfig_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("not json {{{"), 0644)
	if err := InjectConfig(path, AgentEntry{ID: "a1"}); err == nil {
		t.Error("expected error for corrupt config file")
	}
}
