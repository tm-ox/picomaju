package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func validID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// Task is a task definition assigned to a Staff member.
// It describes what the agent does and which Tools it uses to do it.
type Task struct {
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

func (s *Store) List() ([]Task, error) {
	return s.load()
}

func (s *Store) Get(id string) (*Task, error) {
	tasks, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, t := range tasks {
		if t.ID == id {
			return &tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %q not found", id)
}

func (s *Store) Create(t *Task) error {
	if !validID(t.ID) {
		return fmt.Errorf("invalid task id %q: use only letters, digits, hyphens, underscores", t.ID)
	}
	tasks, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range tasks {
		if existing.ID == t.ID {
			return fmt.Errorf("task %q already exists", t.ID)
		}
	}
	return s.save(append(tasks, *t))
}

func (s *Store) Update(t *Task) error {
	tasks, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range tasks {
		if existing.ID == t.ID {
			tasks[i] = *t
			return s.save(tasks)
		}
	}
	return fmt.Errorf("task %q not found", t.ID)
}

func (s *Store) Delete(id string) error {
	tasks, err := s.load()
	if err != nil {
		return err
	}
	for i, t := range tasks {
		if t.ID == id {
			return s.save(append(tasks[:i], tasks[i+1:]...))
		}
	}
	return fmt.Errorf("task %q not found", id)
}

func (s *Store) load() ([]Task, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Tasks, nil
}

func (s *Store) save(tasks []Task) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Tasks []Task `json:"tasks"`
	}{tasks}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
