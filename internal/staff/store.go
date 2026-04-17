package staff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Staff is an agent profile composed of Tasks and Values.
// It is the top-level entity that gets compiled into an agent directive.
type Staff struct {
	ID              string   `json:"id"`
	Label           string   `json:"label"`
	Tasks           []string `json:"tasks,omitempty"`            // task IDs
	ValueCategories []string `json:"value_categories,omitempty"` // bulk inclusion by category
	Values          []string `json:"values,omitempty"`           // individual value IDs
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) List() ([]Staff, error) {
	return s.load()
}

func (s *Store) Get(id string) (*Staff, error) {
	members, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, m := range members {
		if m.ID == id {
			return &members[i], nil
		}
	}
	return nil, fmt.Errorf("staff %q not found", id)
}

func (s *Store) Create(m *Staff) error {
	members, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range members {
		if existing.ID == m.ID {
			return fmt.Errorf("staff %q already exists", m.ID)
		}
	}
	return s.save(append(members, *m))
}

func (s *Store) Update(m *Staff) error {
	members, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range members {
		if existing.ID == m.ID {
			members[i] = *m
			return s.save(members)
		}
	}
	return fmt.Errorf("staff %q not found", m.ID)
}

func (s *Store) Delete(id string) error {
	members, err := s.load()
	if err != nil {
		return err
	}
	for i, m := range members {
		if m.ID == id {
			return s.save(append(members[:i], members[i+1:]...))
		}
	}
	return fmt.Errorf("staff %q not found", id)
}

func (s *Store) load() ([]Staff, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Staff []Staff `json:"staff"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Staff, nil
}

func (s *Store) save(members []Staff) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Staff []Staff `json:"staff"`
	}{members}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
