package hometpl

import (
	"strings"
	"testing"
	"time"
)

func TestGreeting_WithName(t *testing.T) {
	if got := greeting("Alice"); got != "Welcome back, Alice" {
		t.Errorf("got %q", got)
	}
}

func TestGreeting_NoName(t *testing.T) {
	if got := greeting(""); got != "Welcome back" {
		t.Errorf("got %q", got)
	}
}

func TestRelativeTime_JustNow(t *testing.T) {
	got := relativeTime(time.Now().Unix())
	if got != "just now" {
		t.Errorf("got %q", got)
	}
}

func TestRelativeTime_Minutes(t *testing.T) {
	got := relativeTime(time.Now().Add(-5 * time.Minute).Unix())
	if !strings.HasSuffix(got, "m ago") {
		t.Errorf("got %q, expected Xm ago", got)
	}
	if !strings.HasPrefix(got, "5") {
		t.Errorf("got %q, expected 5m ago", got)
	}
}

func TestRelativeTime_Hours(t *testing.T) {
	got := relativeTime(time.Now().Add(-3 * time.Hour).Unix())
	if !strings.HasSuffix(got, "h ago") {
		t.Errorf("got %q, expected Xh ago", got)
	}
	if !strings.HasPrefix(got, "3") {
		t.Errorf("got %q, expected 3h ago", got)
	}
}

func TestRelativeTime_Days(t *testing.T) {
	got := relativeTime(time.Now().Add(-48 * time.Hour).Unix())
	// Older than 24h → "Jan 2" format
	if strings.Contains(got, "ago") {
		t.Errorf("expected date format for old timestamp, got %q", got)
	}
}
