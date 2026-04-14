package category

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

// Seed writes the default categories if the file does not exist yet.
func (s *Store) Seed() error {
	if _, err := os.Stat(s.path); err == nil {
		return nil
	}
	return s.save(Defaults)
}

func (s *Store) List() ([]Category, error) {
	cats, err := s.load()
	if err != nil {
		return nil, err
	}
	return cats, nil
}

func (s *Store) Get(id string) (*Category, error) {
	cats, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, c := range cats {
		if c.ID == id {
			return &cats[i], nil
		}
	}
	return nil, fmt.Errorf("category %q not found", id)
}

func (s *Store) Create(c *Category) error {
	cats, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range cats {
		if existing.ID == c.ID {
			return fmt.Errorf("category %q already exists", c.ID)
		}
	}
	return s.save(append(cats, *c))
}

func (s *Store) Delete(id string) error {
	cats, err := s.load()
	if err != nil {
		return err
	}
	for i, c := range cats {
		if c.ID == id {
			if c.System {
				return fmt.Errorf("cannot delete system category %q", id)
			}
			return s.save(append(cats[:i], cats[i+1:]...))
		}
	}
	return fmt.Errorf("category %q not found", id)
}

func (s *Store) load() ([]Category, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults, nil
		}
		return nil, err
	}
	var payload struct {
		Categories []Category `json:"categories"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Categories, nil
}

func (s *Store) save(cats []Category) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Categories []Category `json:"categories"`
	}{cats}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
