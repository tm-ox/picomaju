package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Write writes compiled workspace files into workspaceDir, creating it if needed.
func Write(out Output, workspaceDir string) error {
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}
	for name, content := range map[string]string{
		"AGENT.md": out.AgentMD,
		"SOUL.md":  out.SoulMD,
		"USER.md":  out.UserMD,
	} {
		if err := os.WriteFile(filepath.Join(workspaceDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// AgentEntry is the minimal picoclaw config.json agent entry picomaju manages.
type AgentEntry struct {
	ID          string `json:"id"`
	Workspace   string `json:"workspace"`
	Description string `json:"description,omitempty"`
}

// InjectConfig upserts an agent entry into picoclaw's config.json.
// Creates a minimal config file if none exists.
func InjectConfig(configPath string, entry AgentEntry) error {
	var root map[string]json.RawMessage
	b, err := os.ReadFile(configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read picoclaw config: %w", err)
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &root); err != nil {
			return fmt.Errorf("parse picoclaw config: %w", err)
		}
	}
	if root == nil {
		root = make(map[string]json.RawMessage)
	}

	type agentsSection struct {
		Defaults json.RawMessage `json:"defaults,omitempty"`
		List     []AgentEntry    `json:"list,omitempty"`
	}
	var agents agentsSection
	if raw, ok := root["agents"]; ok {
		if err := json.Unmarshal(raw, &agents); err != nil {
			return fmt.Errorf("parse agents section: %w", err)
		}
	}

	found := false
	for i, a := range agents.List {
		if a.ID == entry.ID {
			agents.List[i] = entry
			found = true
			break
		}
	}
	if !found {
		agents.List = append(agents.List, entry)
	}

	agentsRaw, err := json.Marshal(agents)
	if err != nil {
		return err
	}
	root["agents"] = agentsRaw

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(configPath, out, 0644)
}
