package role

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Role is a task definition assigned to a Staff member.
// It describes what the agent does and which Tools it uses to do it.
type Role struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Tools       []string `json:"tools,omitempty"` // tool IDs
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) List() ([]Role, error) {
	return s.load()
}

func (s *Store) Get(id string) (*Role, error) {
	roles, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, r := range roles {
		if r.ID == id {
			return &roles[i], nil
		}
	}
	return nil, fmt.Errorf("role %q not found", id)
}

func (s *Store) Create(r *Role) error {
	roles, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range roles {
		if existing.ID == r.ID {
			return fmt.Errorf("role %q already exists", r.ID)
		}
	}
	return s.save(append(roles, *r))
}

func (s *Store) Update(r *Role) error {
	roles, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range roles {
		if existing.ID == r.ID {
			roles[i] = *r
			return s.save(roles)
		}
	}
	return fmt.Errorf("role %q not found", r.ID)
}

func (s *Store) Delete(id string) error {
	roles, err := s.load()
	if err != nil {
		return err
	}
	for i, r := range roles {
		if r.ID == id {
			return s.save(append(roles[:i], roles[i+1:]...))
		}
	}
	return fmt.Errorf("role %q not found", id)
}

func (s *Store) load() ([]Role, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Roles []Role `json:"roles"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Roles, nil
}

func (s *Store) save(roles []Role) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Roles []Role `json:"roles"`
	}{roles}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
