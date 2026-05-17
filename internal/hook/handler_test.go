package hook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// buildRPC encodes a JSON-RPC 2.0 request for stdin.
func buildRPC(t *testing.T, checkpoint, agentID, toolName, context string) *bytes.Buffer {
	t.Helper()
	params, _ := json.Marshal(hookParams{
		Checkpoint: checkpoint,
		AgentID:    agentID,
		ToolName:   toolName,
		Context:    context,
	})
	req := rpcRequest{
		JSONRPC: "2.0",
		Method:  checkpoint,
		Params:  params,
		ID:      json.RawMessage(`1`),
	}
	b, _ := json.Marshal(req)
	return bytes.NewBuffer(b)
}

func decodeResult(t *testing.T, out *bytes.Buffer) rpcResult {
	t.Helper()
	var r rpcResult
	if err := json.NewDecoder(out).Decode(&r); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	return r
}

// ── observer checkpoints ──────────────────────────────────────────────────────

func TestRun_ObserverCheckpoint_PostsEvent(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			received = buf.Bytes()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"evt1"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	in := buildRPC(t, "before_tool", "agent1", "send_message", "sending reply")
	out := &bytes.Buffer{}
	run(in, out, srv.URL, "tok", srv.Client())

	if len(received) == 0 {
		t.Fatal("expected event to be posted")
	}
	var ev map[string]any
	json.Unmarshal(received, &ev)
	if ev["type"] != "action" {
		t.Errorf("expected type=action, got %q", ev["type"])
	}
	if !strings.Contains(ev["summary"].(string), "send_message") {
		t.Errorf("summary should include tool name: %q", ev["summary"])
	}

	result := decodeResult(t, out)
	if string(result.ID) != "1" {
		t.Errorf("expected id=1, got %s", result.ID)
	}
}

func TestRun_LLMCheckpoint_PostsMessageEvent(t *testing.T) {
	var evType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev map[string]any
		json.NewDecoder(r.Body).Decode(&ev)
		evType = ev["type"].(string)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e1"}`))
	}))
	defer srv.Close()

	in := buildRPC(t, "after_llm", "a1", "", "model responded")
	out := &bytes.Buffer{}
	run(in, out, srv.URL, "tok", srv.Client())

	if evType != "message" {
		t.Errorf("expected message event for after_llm, got %q", evType)
	}
}

func TestRun_ErrorInParams_PostsErrorEvent(t *testing.T) {
	var evType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ev map[string]any
		json.NewDecoder(r.Body).Decode(&ev)
		evType, _ = ev["type"].(string)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e1"}`))
	}))
	defer srv.Close()

	params, _ := json.Marshal(hookParams{
		Checkpoint: "after_tool",
		AgentID:    "a1",
		ToolName:   "fetch_url",
		Error:      "timeout",
	})
	req := rpcRequest{JSONRPC: "2.0", Method: "after_tool", Params: params, ID: json.RawMessage(`1`)}
	b, _ := json.Marshal(req)
	out := &bytes.Buffer{}
	run(bytes.NewBuffer(b), out, srv.URL, "tok", srv.Client())

	if evType != "error" {
		t.Errorf("expected error event when params.Error set, got %q", evType)
	}
}

func TestRun_MissingAgentID_ReturnOK(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"e1"}`))
	}))
	defer srv.Close()

	params, _ := json.Marshal(hookParams{Checkpoint: "before_tool"})
	req := rpcRequest{JSONRPC: "2.0", Params: params, ID: json.RawMessage(`2`)}
	b, _ := json.Marshal(req)
	out := &bytes.Buffer{}
	run(bytes.NewBuffer(b), out, srv.URL, "tok", srv.Client())

	if called {
		t.Error("should not post event when agent_id is missing")
	}
	result := decodeResult(t, out)
	if string(result.ID) != "2" {
		t.Errorf("id not echoed: %s", result.ID)
	}
}

func TestRun_BadJSON_ReturnOK(t *testing.T) {
	out := &bytes.Buffer{}
	run(strings.NewReader("{bad json}"), out, "http://localhost", "tok", &http.Client{})
	// Should not panic; should write a valid response
	if out.Len() == 0 {
		t.Error("expected output even on bad JSON input")
	}
}

// ── approval flow ─────────────────────────────────────────────────────────────

func TestRun_ApprovalCheckpoint_Approved(t *testing.T) {
	eventPosted := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") {
			eventPosted = true
			var ev map[string]any
			json.NewDecoder(r.Body).Decode(&ev)
			if ev["type"] != "approval_request" {
				t.Errorf("expected approval_request event, got %q", ev["type"])
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"evt-approve"}`))
			return
		}
		// Long-poll approval endpoint
		if strings.Contains(r.URL.Path, "/approvals/") {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"decision":"approved"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	in := buildRPC(t, "approve_tool", "agent1", "delete_record", "about to delete")
	out := &bytes.Buffer{}
	run(in, out, srv.URL, "tok", srv.Client())

	if !eventPosted {
		t.Error("approval_request event should be posted")
	}
	result := decodeResult(t, out)
	m, _ := result.Result.(map[string]any)
	if m["decision"] != "approved" {
		t.Errorf("expected decision=approved, got %v", m["decision"])
	}
}

func TestRun_ApprovalCheckpoint_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/events") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"evt-deny"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"decision":"denied"}`))
	}))
	defer srv.Close()

	in := buildRPC(t, "approve_tool", "agent1", "send_bulk_message", "")
	out := &bytes.Buffer{}
	run(in, out, srv.URL, "tok", srv.Client())

	result := decodeResult(t, out)
	m, _ := result.Result.(map[string]any)
	if m["decision"] != "denied" {
		t.Errorf("expected decision=denied, got %v", m["decision"])
	}
}

func TestRun_ApprovalCheckpoint_EventPostFails_Denied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	in := buildRPC(t, "approve_tool", "agent1", "dangerous_op", "")
	out := &bytes.Buffer{}
	run(in, out, srv.URL, "tok", srv.Client())

	result := decodeResult(t, out)
	m, _ := result.Result.(map[string]any)
	if m["decision"] != "denied" {
		t.Errorf("expected denied on event-post failure, got %v", m["decision"])
	}
}

func TestPollForDecision_RetriesOn408(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusRequestTimeout)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"decision":"approved"}`))
	}))
	defer srv.Close()

	result := pollForDecision(srv.URL, "tok", "a1", "e1")
	if result != "approved" {
		t.Errorf("expected approved, got %q", result)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 calls for retry behaviour, got %d", calls)
	}
}

func TestHandleApproval_BearerTokenSent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if strings.HasSuffix(r.URL.Path, "/events") {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"e1"}`))
			return
		}
		w.Write([]byte(`{"decision":"approved"}`))
	}))
	defer srv.Close()

	p := hookParams{Checkpoint: "approve_tool", AgentID: "a1", ToolName: "op"}
	handleApproval(srv.Client(), srv.URL, "secret-tok", p)

	if !strings.Contains(gotAuth, "secret-tok") {
		t.Errorf("Bearer token not sent, got %q", gotAuth)
	}
}

func TestRun_ObserverCheckpoint_APIUnavailable_NoError(t *testing.T) {
	in := buildRPC(t, "before_tool", "agent1", "tool", "")
	out := &bytes.Buffer{}
	// Point at a port nothing is listening on.
	client := &http.Client{Timeout: 100 * time.Millisecond}
	run(in, out, "http://127.0.0.1:19999", "tok", client)
	// Should still write a result (not panic or block).
	if out.Len() == 0 {
		t.Error("expected output even when API is unreachable")
	}
}
