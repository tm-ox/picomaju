package compiler

import (
	"strings"
	"testing"

	"picomaju/internal/settings"
	"picomaju/internal/staff"
	"picomaju/internal/task"
	"picomaju/internal/tool"
	"picomaju/internal/value"
)

func baseStaff() *staff.Staff {
	return &staff.Staff{ID: "s1", Label: "Support Agent", Description: "Handles customer queries."}
}

func TestBuildAgent_FrontmatterAndRole(t *testing.T) {
	out := Compile(Input{Staff: baseStaff()})
	if !strings.Contains(out.AgentMD, "name: Support Agent") {
		t.Error("missing name in frontmatter")
	}
	if !strings.Contains(out.AgentMD, "description:") {
		t.Error("missing description in frontmatter")
	}
	if !strings.Contains(out.AgentMD, "# Role") {
		t.Error("missing Role section")
	}
	if !strings.Contains(out.AgentMD, "Handles customer queries.") {
		t.Error("missing role body")
	}
}

func TestBuildAgent_FallbackRoleFromBusiness(t *testing.T) {
	s := &staff.Staff{ID: "s1", Label: "Agent"}
	out := Compile(Input{
		Staff:    s,
		Settings: &settings.Settings{BusinessName: "Acme Co"},
	})
	if !strings.Contains(out.AgentMD, "Acme Co") {
		t.Error("expected business name fallback in role")
	}
}

func TestBuildAgent_NoDescriptionNoFallbackWithoutSettings(t *testing.T) {
	s := &staff.Staff{ID: "s1", Label: "Agent"}
	out := Compile(Input{Staff: s})
	// No description, no settings — Role section exists but body is empty.
	if !strings.Contains(out.AgentMD, "# Role") {
		t.Error("missing Role section")
	}
}

func TestBuildAgent_TasksSection(t *testing.T) {
	tl := tool.Tool{ID: "t1", Label: "WhatsApp", Type: "whatsapp"}
	tk := task.Task{ID: "tk1", Label: "Reply to customers", Description: "Answer inbound messages.", Tools: []string{"t1"}}
	out := Compile(Input{
		Staff: baseStaff(),
		Tasks: []task.Task{tk},
		Tools: []tool.Tool{tl},
	})
	if !strings.Contains(out.AgentMD, "## Reply to customers") {
		t.Error("missing task heading")
	}
	if !strings.Contains(out.AgentMD, "Answer inbound messages.") {
		t.Error("missing task description")
	}
	if !strings.Contains(out.AgentMD, "**Tools:** WhatsApp") {
		t.Error("missing tool label in task")
	}
}

func TestBuildAgent_MissingToolWarning(t *testing.T) {
	tk := task.Task{ID: "tk1", Label: "Send report", Tools: []string{"ghost-tool"}}
	out := Compile(Input{
		Staff: baseStaff(),
		Tasks: []task.Task{tk},
	})
	if len(out.Warnings) == 0 {
		t.Fatal("expected a warning for missing tool ref")
	}
	if !strings.Contains(out.Warnings[0], "ghost-tool") {
		t.Errorf("warning should name the missing tool, got: %s", out.Warnings[0])
	}
}

func TestBuildAgent_NoTasksSectionWhenEmpty(t *testing.T) {
	out := Compile(Input{Staff: baseStaff()})
	if strings.Contains(out.AgentMD, "# Tasks") {
		t.Error("Tasks section should be absent when no tasks provided")
	}
}

func TestBuildAgent_IntegrationsSection(t *testing.T) {
	// whatsapp is in the catalog
	tl := tool.Tool{ID: "t1", Label: "WA Biz", Type: "whatsapp"}
	out := Compile(Input{Staff: baseStaff(), Tools: []tool.Tool{tl}})
	if !strings.Contains(out.AgentMD, "# Integrations") {
		t.Error("missing Integrations section")
	}
	if !strings.Contains(out.AgentMD, "WA Biz") {
		t.Error("missing tool label in integrations")
	}
	if !strings.Contains(out.AgentMD, "messaging") {
		t.Error("missing category from catalog")
	}
}

func TestBuildAgent_UnknownToolTypeInIntegrations(t *testing.T) {
	tl := tool.Tool{ID: "t1", Label: "Mystery Box", Type: "unknown_xyz"}
	out := Compile(Input{Staff: baseStaff(), Tools: []tool.Tool{tl}})
	if !strings.Contains(out.AgentMD, "- **Mystery Box**") {
		t.Error("unknown tool type should still appear without category")
	}
}

func TestBuildSoul_CategoryOrder(t *testing.T) {
	vals := []*value.Value{
		{ID: "v1", Title: "Custom Rule", Category: "custom", Priority: 50, Body: "do this"},
		{ID: "v2", Title: "Be Honest", Category: "core_values", Priority: 80, Body: "always"},
		{ID: "v3", Title: "Speak Clearly", Category: "communication", Priority: 60, Body: "use plain words"},
	}
	out := Compile(Input{Staff: baseStaff(), Values: vals})
	coreIdx := strings.Index(out.SoulMD, "# Core Values")
	commIdx := strings.Index(out.SoulMD, "# Communication")
	custIdx := strings.Index(out.SoulMD, "# Custom")
	if coreIdx < 0 || commIdx < 0 || custIdx < 0 {
		t.Fatal("missing one or more category sections")
	}
	if !(coreIdx < commIdx && commIdx < custIdx) {
		t.Error("category sections not in expected order: core_values, communication, custom")
	}
}

func TestBuildSoul_PriorityOrderWithinCategory(t *testing.T) {
	vals := []*value.Value{
		{ID: "v1", Title: "Low", Category: "core_values", Priority: 10, Body: "low"},
		{ID: "v2", Title: "High", Category: "core_values", Priority: 90, Body: "high"},
	}
	out := Compile(Input{Staff: baseStaff(), Values: vals})
	highIdx := strings.Index(out.SoulMD, "## High")
	lowIdx := strings.Index(out.SoulMD, "## Low")
	if highIdx > lowIdx {
		t.Error("higher priority value should appear before lower priority")
	}
}

func TestBuildSoul_EmptyFallback(t *testing.T) {
	out := Compile(Input{Staff: baseStaff()})
	if !strings.Contains(out.SoulMD, "no configured values") {
		t.Error("expected fallback text when no values provided")
	}
}

func TestBuildUser_WithUser(t *testing.T) {
	out := Compile(Input{
		Staff: baseStaff(),
		User:  &UserContext{Name: "Alice", Role: "Owner", Description: "Runs the front desk."},
	})
	if !strings.Contains(out.UserMD, "# Current User") {
		t.Error("missing Current User section")
	}
	if !strings.Contains(out.UserMD, "Alice") {
		t.Error("missing user name")
	}
	if !strings.Contains(out.UserMD, "Owner") {
		t.Error("missing user role")
	}
	if !strings.Contains(out.UserMD, "Runs the front desk.") {
		t.Error("missing user description")
	}
}

func TestBuildUser_NoUserSection(t *testing.T) {
	out := Compile(Input{Staff: baseStaff()})
	if strings.Contains(out.UserMD, "# Current User") {
		t.Error("Current User section should be absent when no user provided")
	}
}

func TestBuildUser_BusinessContext(t *testing.T) {
	out := Compile(Input{
		Staff: baseStaff(),
		Settings: &settings.Settings{
			BusinessName:    "Bali Surf Co",
			BusinessDetails: "Surf lessons and rentals.",
			Timezone:        "Asia/Makassar",
			Hours:           "8am–6pm",
			Languages:       []string{"en", "id"},
		},
	})
	if !strings.Contains(out.UserMD, "# Business Context") {
		t.Error("missing Business Context section")
	}
	if !strings.Contains(out.UserMD, "Bali Surf Co") {
		t.Error("missing business name")
	}
	if !strings.Contains(out.UserMD, "Asia/Makassar") {
		t.Error("missing timezone")
	}
	if !strings.Contains(out.UserMD, "en, id") {
		t.Error("missing languages")
	}
}

func TestBuildUser_NilSettings(t *testing.T) {
	out := Compile(Input{Staff: baseStaff(), Settings: nil})
	// Should not panic; Business Context header present but empty.
	if !strings.Contains(out.UserMD, "# Business Context") {
		t.Error("missing Business Context section even with nil settings")
	}
}
