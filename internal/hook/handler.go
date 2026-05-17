// Package hook implements the `picomaju hook` subcommand.
// picoclaw spawns this process for each checkpoint and communicates via JSON-RPC 2.0 over stdio.
package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      json.RawMessage `json:"id"`
}

// hookParams contains the fields picoclaw sends for each checkpoint.
// The formal schema is not publicly documented; fields are best-effort.
type hookParams struct {
	Checkpoint string         `json:"checkpoint"`
	ToolName   string         `json:"tool_name"`
	ToolArgs   map[string]any `json:"tool_args"`
	AgentID    string         `json:"agent_id"`
	Context    string         `json:"context"`
	Error      string         `json:"error,omitempty"`
}

type rpcResult struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  any             `json:"result"`
	ID      json.RawMessage `json:"id"`
}

// Run is called by main when os.Args[1] == "hook".
// Reads one JSON-RPC request from stdin, processes it, writes the result to stdout.
func Run() {
	apiURL := os.Getenv("PICOMAJU_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:18800"
	}
	token := os.Getenv("PICOMAJU_TOKEN")
	client := &http.Client{Timeout: 30 * time.Second}
	run(os.Stdin, os.Stdout, apiURL, token, client)
}

func run(in io.Reader, out io.Writer, apiURL, token string, client *http.Client) {
	var req rpcRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		respond(out, nil, map[string]any{"ok": true})
		return
	}

	var params hookParams
	json.Unmarshal(req.Params, &params) //nolint:errcheck

	if params.AgentID == "" {
		respond(out, req.ID, map[string]any{"ok": true})
		return
	}

	if params.Checkpoint == "approve_tool" {
		decision := handleApproval(client, apiURL, token, params)
		respond(out, req.ID, map[string]string{"decision": decision})
		return
	}

	// Observer checkpoint — report event and return immediately.
	postEvent(client, apiURL, token, params)
	respond(out, req.ID, map[string]any{"ok": true})
}

func postEvent(client *http.Client, apiURL, token string, p hookParams) {
	evType := "action"
	if p.Error != "" {
		evType = "error"
	} else if p.Checkpoint == "before_llm" || p.Checkpoint == "after_llm" {
		evType = "message"
	}

	summary := p.Checkpoint
	if p.ToolName != "" {
		summary = fmt.Sprintf("%s: %s", p.Checkpoint, p.ToolName)
	}

	body, _ := json.Marshal(map[string]any{
		"type":    evType,
		"summary": summary,
		"detail":  p.Context,
	})
	req, err := http.NewRequest(http.MethodPost, apiURL+"/agents/"+p.AgentID+"/events", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func handleApproval(client *http.Client, apiURL, token string, p hookParams) string {
	summary := "Approval required"
	if p.ToolName != "" {
		summary = fmt.Sprintf("Approval required: %s", p.ToolName)
	}
	body, _ := json.Marshal(map[string]any{
		"type":    "approval_request",
		"summary": summary,
		"detail":  p.Context,
	})
	req, err := http.NewRequest(http.MethodPost, apiURL+"/agents/"+p.AgentID+"/events", bytes.NewReader(body))
	if err != nil {
		return "denied"
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "denied"
	}
	var stored struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&stored) //nolint:errcheck
	resp.Body.Close()

	if stored.ID == "" {
		return "denied"
	}

	return pollForDecision(apiURL, token, p.AgentID, stored.ID)
}

// pollForDecision long-polls picomaju's approval endpoint until a decision is made.
// picomaju returns 408 on each 25s timeout window; we retry immediately.
func pollForDecision(apiURL, token, agentID, eventID string) string {
	for {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest(http.MethodGet, apiURL+"/agents/"+agentID+"/approvals/"+eventID, nil)
		if err != nil {
			return "denied"
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if resp.StatusCode == http.StatusRequestTimeout {
			resp.Body.Close()
			continue
		}
		var result struct {
			Decision string `json:"decision"`
		}
		json.NewDecoder(resp.Body).Decode(&result) //nolint:errcheck
		resp.Body.Close()
		if result.Decision == "approved" || result.Decision == "denied" {
			return result.Decision
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func respond(out io.Writer, id json.RawMessage, result any) {
	json.NewEncoder(out).Encode(rpcResult{ //nolint:errcheck
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	})
}
