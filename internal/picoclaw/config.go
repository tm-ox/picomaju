package picoclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the picoclaw agent runtime configuration written to config.json.
type Config struct {
	AgentID      string          `json:"agent_id"`
	WorkspaceDir string          `json:"workspace_dir"`
	Tools        []ToolConfig    `json:"tools,omitempty"`
	LLMProxy     *LLMProxyConfig `json:"llm_proxy,omitempty"`
}

// ToolConfig mirrors tool.Tool config for picoclaw consumption.
type ToolConfig struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

// LLMProxyConfig tells picoclaw where to route LLM calls and how to authenticate.
type LLMProxyConfig struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// WriteConfig serialises cfg to {workspaceDir}/config.json.
func WriteConfig(cfg Config, workspaceDir string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(workspaceDir, "config.json"), data, 0600)
}
