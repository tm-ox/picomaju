package value

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store is a file-based Value store. Each Value is one .md file with YAML frontmatter.
type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) List() ([]*Value, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var values []*Value
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		v, err := s.readFile(e.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", e.Name(), err)
		}
		values = append(values, v)
	}
	return values, nil
}

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

func (s *Store) Get(id string) (*Value, error) {
	if !validID(id) {
		return nil, fmt.Errorf("invalid value id %q", id)
	}
	v, err := s.readFile(id + ".md")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("value %q not found", id)
		}
		return nil, err
	}
	return v, nil
}

func (s *Store) Create(v *Value) error {
	if !validID(v.ID) {
		return fmt.Errorf("invalid value id %q", v.ID)
	}
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}
	return s.writeFile(v)
}

func (s *Store) Update(v *Value) error {
	if !validID(v.ID) {
		return fmt.Errorf("invalid value id %q", v.ID)
	}
	return s.writeFile(v)
}

func (s *Store) Delete(id string) error {
	if !validID(id) {
		return fmt.Errorf("invalid value id %q", id)
	}
	err := os.Remove(filepath.Join(s.dir, id+".md"))
	if os.IsNotExist(err) {
		return fmt.Errorf("value %q not found", id)
	}
	return err
}

func (s *Store) readFile(filename string) (*Value, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, filename))
	if err != nil {
		return nil, err
	}
	return parseFrontmatter(data)
}

func (s *Store) writeFile(v *Value) error {
	fm, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	content := fmt.Sprintf("---\n%s---\n\n%s\n", string(fm), v.Body)
	return os.WriteFile(filepath.Join(s.dir, v.ID+".md"), []byte(content), 0644)
}

// parseFrontmatter splits ---\nYAML\n---\nbody and unmarshals the Value.
func parseFrontmatter(data []byte) (*Value, error) {
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
	body := strings.TrimPrefix(rest[end+len(close):], "\n")
	body = strings.TrimSpace(body)

	var val Value
	if err := yaml.Unmarshal([]byte(fm), &val); err != nil {
		return nil, err
	}
	val.Body = body
	return &val, nil
}
