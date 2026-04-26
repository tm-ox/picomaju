package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Message struct {
	Role      string `json:"role"` // "user" | "assistant"
	Content   string `json:"content"`
	Timestamp int64  `json:"ts"`
}

type Chat struct {
	ID        string    `json:"id"`
	StaffID   string    `json:"staff_id"`
	Title     string    `json:"title"`
	CreatedAt int64     `json:"created_at"`
	Messages  []Message `json:"messages,omitempty"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) ListByStaff(staffID string) ([]Chat, error) {
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	var out []Chat
	for _, c := range all {
		if c.StaffID == staffID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Store) Get(id string) (*Chat, error) {
	all, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, c := range all {
		if c.ID == id {
			return &all[i], nil
		}
	}
	return nil, fmt.Errorf("chat %q not found", id)
}

func (s *Store) Create(c *Chat) error {
	all, err := s.load()
	if err != nil {
		return err
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	return s.save(append(all, *c))
}

func (s *Store) Update(c *Chat) error {
	all, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range all {
		if existing.ID == c.ID {
			all[i] = *c
			return s.save(all)
		}
	}
	return fmt.Errorf("chat %q not found", c.ID)
}

func (s *Store) Delete(id string) error {
	all, err := s.load()
	if err != nil {
		return err
	}
	for i, c := range all {
		if c.ID == id {
			return s.save(append(all[:i], all[i+1:]...))
		}
	}
	return fmt.Errorf("chat %q not found", id)
}

func (s *Store) load() ([]Chat, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Chats []Chat `json:"chats"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Chats, nil
}

func (s *Store) save(chats []Chat) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Chats []Chat `json:"chats"`
	}{chats}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
