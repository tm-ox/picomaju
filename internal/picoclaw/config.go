package picoclaw

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the picoclaw configuration written to ~/.picoclaw/config.json.
type Config struct {
	Version   int           `json:"version"`
	Agents    AgentsConfig  `json:"agents"`
	ModelList []ModelEntry  `json:"model_list"`
	Gateway   GatewayConfig `json:"gateway"`
	Hooks     *HooksConfig  `json:"hooks,omitempty"`
}

type AgentsConfig struct {
	Defaults AgentDefaults `json:"defaults"`
}

// AgentDefaults sets runtime defaults for all agents picoclaw manages.
type AgentDefaults struct {
	Workspace string `json:"workspace"`
	ModelName string `json:"model_name"`
	MaxTokens int    `json:"max_tokens,omitempty"`
}

// ModelEntry registers an LLM provider in picoclaw's model_list.
// picoclaw uses the OpenAI chat-completions protocol; set APIBase to override
// the endpoint (e.g. to point at picomaju's /proxy/v1).
type ModelEntry struct {
	ModelName string   `json:"model_name"`
	Model     string   `json:"model"`
	APIKeys   []string `json:"api_keys"`
	APIBase   string   `json:"api_base,omitempty"`
}

type GatewayConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level,omitempty"`
}

// HooksConfig configures picoclaw's hook system.
// picoclaw spawns the hook Command for each checkpoint event, sends JSON-RPC 2.0 over stdin,
// and reads the result from stdout.
type HooksConfig struct {
	Enabled bool        `json:"enabled"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry defines a single hook process picoclaw invokes at the listed checkpoints.
// Supported checkpoints: before_llm, after_llm, before_tool, after_tool, approve_tool.
type HookEntry struct {
	ID          string            `json:"id"`
	Enabled     bool              `json:"enabled"`
	Checkpoints []string          `json:"checkpoints"`
	Type        string            `json:"type"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

// ConfigPath returns the canonical picoclaw config file path (~/.picoclaw/config.json).
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".picoclaw", "config.json")
}

// WriteConfig serialises cfg as JSON to dest, creating parent directories as needed.
func WriteConfig(cfg Config, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0600)
}
