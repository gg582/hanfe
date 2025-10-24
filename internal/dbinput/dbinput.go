package dbinput

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode"
)

type Dictionary struct {
	entries map[string][]string
}

func LoadDictionary(path string) (*Dictionary, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	defer file.Close()

	dict := &Dictionary{entries: make(map[string][]string)}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		dict.entries[strings.ToLower(key)] = append(dict.entries[strings.ToLower(key)], value)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read database %s: %w", path, err)
	}
	return dict, nil
}

func (d *Dictionary) Lookup(key string) (string, bool) {
	if d == nil {
		return "", false
	}
	values, ok := d.entries[strings.ToLower(key)]
	if !ok || len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func (d *Dictionary) NewSession() *Session {
	return &Session{dict: d}
}

type Session struct {
	dict   *Dictionary
	buffer []rune
}

func (s *Session) Append(text string) string {
	if s == nil {
		return ""
	}
	for _, r := range text {
		s.buffer = append(s.buffer, unicode.ToLower(r))
	}
	return string(s.buffer)
}

func (s *Session) Backspace() (string, bool) {
	if s == nil || len(s.buffer) == 0 {
		return "", false
	}
	s.buffer = s.buffer[:len(s.buffer)-1]
	return string(s.buffer), true
}

func (s *Session) Commit() string {
	if s == nil || len(s.buffer) == 0 {
		return ""
	}
	key := string(s.buffer)
	s.buffer = nil
	if s.dict == nil {
		return key
	}
	if value, ok := s.dict.Lookup(key); ok {
		return value
	}
	return key
}

func (s *Session) Flush() string {
	return s.Commit()
}

func (s *Session) Reset() {
	if s != nil {
		s.buffer = nil
	}
}

func (s *Session) Preedit() string {
	if s == nil {
		return ""
	}
	return string(s.buffer)
}
