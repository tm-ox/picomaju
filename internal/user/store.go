package user

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

func (s *Store) List() ([]User, error) {
	return s.load()
}

func (s *Store) Get(id string) (*User, error) {
	users, err := s.load()
	if err != nil {
		return nil, err
	}
	for i, u := range users {
		if u.ID == id {
			return &users[i], nil
		}
	}
	return nil, fmt.Errorf("user %q not found", id)
}

func (s *Store) Create(u *User) error {
	users, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range users {
		if existing.ID == u.ID {
			return fmt.Errorf("user %q already exists", u.ID)
		}
	}
	return s.save(append(users, *u))
}

func (s *Store) Update(u *User) error {
	users, err := s.load()
	if err != nil {
		return err
	}
	for i, existing := range users {
		if existing.ID == u.ID {
			users[i] = *u
			return s.save(users)
		}
	}
	return fmt.Errorf("user %q not found", u.ID)
}

func (s *Store) Delete(id string) error {
	users, err := s.load()
	if err != nil {
		return err
	}
	for i, u := range users {
		if u.ID == id {
			return s.save(append(users[:i], users[i+1:]...))
		}
	}
	return fmt.Errorf("user %q not found", id)
}

func (s *Store) Count() (int, error) {
	users, err := s.load()
	return len(users), err
}

func (s *Store) load() ([]User, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var payload struct {
		Users []User `json:"users"`
	}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload.Users, nil
}

func (s *Store) save(users []User) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(struct {
		Users []User `json:"users"`
	}{users}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0644)
}
