package ini

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type File struct {
	sections map[string]*Section
}

type Section struct {
	keys map[string]string
}

type Key struct {
	value string
}

func Load(path string) (*File, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	current := &Section{keys: make(map[string]string)}
	sections := map[string]*Section{"": current}
	name := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name = strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				name = ""
			}
			if _, ok := sections[name]; !ok {
				sections[name] = &Section{keys: make(map[string]string)}
			}
			current = sections[name]
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, errors.New("ini: malformed line")
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		current.keys[strings.ToLower(key)] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &File{sections: sections}, nil
}

func (f *File) Section(name string) *Section {
	if f == nil {
		return &Section{keys: map[string]string{}}
	}
	key := strings.ToLower(name)
	if section, ok := f.sections[key]; ok {
		return section
	}
	if section, ok := f.sections[""]; ok {
		return section
	}
	return &Section{keys: map[string]string{}}
}

func (s *Section) Key(name string) *Key {
	if s == nil {
		return &Key{}
	}
	value, ok := s.keys[strings.ToLower(name)]
	if !ok {
		return &Key{}
	}
	return &Key{value: value}
}

func (k *Key) String() string {
	return k.value
}

func (k *Key) MustString(def string) string {
	if k.value == "" {
		return def
	}
	return k.value
}
