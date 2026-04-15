package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Tool is a capability or integration available to a Role.
type Tool struct {
	ID     string         `json:"id"`
	Label  string         `json:"label"`
	Type   string         `json:"type"`             // e.g. "email", "whatsapp", "mcp"
	Config map[string]any `json:"config,omitempty"` // type-specific config, TBD
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) List() ([]Tool, error) {
	return s.load()
}

func (s *Store) Get(id string) (*Tool, error) {
	tools, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, t := range tools {
		if t.ID == id {
			return &tools[i], nil
		}
	}
	return nil, fmt.Errorf("tool %q not found", id)
}

func (s *Store) Create(t *Tool) error {
	tools, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range tools {
		if existing.ID == t.ID {
			return fmt.Errorf("tool %q already exists", t.ID)
		}
	}
	return s.save(append(tools, *t))
}

func (s *Store) Update(t *Tool) error {
	tools, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range tools {
		if existing.ID == t.ID {
			tools[i] = *t
			return s.save(tools)
		}
	}
	return fmt.Errorf("tool %q not found", t.ID)
}

func (s *Store) Delete(id string) error {
	tools, err := s.load()
	if err != nil {
		return err
	}
	for i, t := range tools {
		if t.ID == id {
			return s.save(append(tools[:i], tools[i+1:]...))
		}
	}
	return fmt.Errorf("tool %q not found", id)
}

func (s *Store) load() ([]Tool, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Tools, nil
}

func (s *Store) save(tools []Tool) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Tools []Tool `json:"tools"`
	}{tools}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
