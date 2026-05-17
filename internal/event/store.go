package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store appends events to per-agent JSONL files under dir.
// Reads are not cached — files are the source of truth.
type Store struct {
	dir string
	mu  sync.Mutex
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

// Append writes e to {dir}/{agentID}.jsonl.
func (s *Store) Append(e Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(s.dir, e.AgentID+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s\n", b)
	return err
}

// ListByAgent returns all events for agentID in chronological order.
func (s *Store) ListByAgent(agentID string) ([]Event, error) {
	return readJSONL(filepath.Join(s.dir, agentID+".jsonl"))
}

// PendingApprovals returns approval_request events with no corresponding approval_result.
// Expired requests (ExpiresAt > 0 && past) are excluded.
func (s *Store) PendingApprovals(agentID string) ([]Event, error) {
	all, err := s.ListByAgent(agentID)
	if err != nil {
		return nil, err
	}
	resolved := map[string]bool{}
	for _, e := range all {
		if e.Type == EventTypeApprovalResult && e.RefID != "" {
			resolved[e.RefID] = true
		}
	}
	now := time.Now().Unix()
	var pending []Event
	for _, e := range all {
		if e.Type != EventTypeApprovalRequest {
			continue
		}
		if resolved[e.ID] {
			continue
		}
		if e.ExpiresAt > 0 && e.ExpiresAt <= now {
			continue
		}
		pending = append(pending, e)
	}
	return pending, nil
}

// RecentAll returns up to n events across all agents, newest first.
func (s *Store) RecentAll(n int) ([]Event, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var all []Event
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		events, _ := readJSONL(filepath.Join(s.dir, entry.Name()))
		all = append(all, events...)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp > all[j].Timestamp
	})
	if len(all) > n {
		all = all[:n]
	}
	return all, nil
}

func readJSONL(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var events []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Event
		if json.Unmarshal(line, &e) == nil {
			events = append(events, e)
		}
	}
	return events, sc.Err()
}
