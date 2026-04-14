package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	BusinessName    string `json:"business_name"`
	BusinessDetails string `json:"business_details"`
	DataDir         string `json:"data_dir"`
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (*Settings, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return &Settings{}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Settings
	return &cfg, json.Unmarshal(data, &cfg)
}

func (s *Store) Save(cfg *Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
