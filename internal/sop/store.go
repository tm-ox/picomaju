package sop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store is a file-based SOP store. Each SOP is one .md file with YAML frontmatter.
type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) List() ([]*SOP, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var sops []*SOP
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		sop, err := s.readFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		sops = append(sops, sop)
	}
	return sops, nil
}

func (s *Store) Get(id string) (*SOP, error) {
	sop, err := s.readFile(id + ".md")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("sop %q not found", id)
		}
		return nil, err
	}
	return sop, nil
}

func (s *Store) Create(sop *SOP) error {
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	return s.writeFile(sop)
}

func (s *Store) Update(sop *SOP) error {
	return s.writeFile(sop)
}

func (s *Store) Delete(id string) error {
	err := os.Remove(filepath.Join(s.dir, id+".md"))
	if os.IsNotExist(err) {
		return fmt.Errorf("sop %q not found", id)
	}
	return err
}

func (s *Store) readFile(filename string) (*SOP, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, filename))
	if err != nil {
		return nil, err
	}
	return parseFrontmatter(data)
}

func (s *Store) writeFile(sop *SOP) error {
	fm, err := yaml.Marshal(sop)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), sop.Body)
	return os.WriteFile(filepath.Join(s.dir, sop.ID+".md"), []byte(content), 0644)
}

// parseFrontmatter splits ---\nYAML\n---\nbody and unmarshals the SOP.
func parseFrontmatter(data []byte) (*SOP, error) {
	const open = "---\n"
	const close = "\n---"
	str := string(data)
	if !strings.HasPrefix(str, open) {
		return nil, fmt.Errorf("missing opening frontmatter delimiter")
	}
	rest := str[len(open):]
	end := strings.Index(rest, close)
	if end == -1 {
		return nil, fmt.Errorf("missing closing frontmatter delimiter")
	}
	fm := rest[:end]
	body := strings.TrimSpace(rest[end+len(close):])
	// Strip leading newline after closing ---
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimSpace(body)

	var s SOP
	if err := yaml.Unmarshal([]byte(fm), &s); err != nil {
		return nil, err
	}
	s.Body = body
	return &s, nil
}
