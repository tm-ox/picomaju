package event

// EventType classifies what kind of event an agent reported.
type EventType string

const (
	EventTypeAction          EventType = "action"           // agent completed an action
	EventTypeApprovalRequest EventType = "approval_request" // agent needs human sign-off
	EventTypeApprovalResult  EventType = "approval_result"  // human responded to a request
	EventTypeMessage         EventType = "message"          // inbound/outbound customer message
	EventTypeError           EventType = "error"            // agent encountered an error
)

// Event is a single timestamped report from a picoclaw agent.
type Event struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id"`
	Type      EventType         `json:"type"`
	Timestamp int64             `json:"timestamp"`
	Summary   string            `json:"summary"`
	Detail    string            `json:"detail,omitempty"`
	Channel   string            `json:"channel,omitempty"` // whatsapp, telegram, shopee, …
	RefID     string            `json:"ref_id,omitempty"`  // approval_result → approval_request ID
	Decision  string            `json:"decision,omitempty"` // "approved" | "denied"
	ExpiresAt int64             `json:"expires_at,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}
