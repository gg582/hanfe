package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Database struct {
	entries map[string]string
}

func LoadDatabase(path string) (Database, error) {
	file, err := os.Open(path)
	if err != nil {
		return Database{}, fmt.Errorf("open backend database: %w", err)
	}
	defer file.Close()

	var raw map[string]string
	if err := json.NewDecoder(file).Decode(&raw); err != nil {
		return Database{}, fmt.Errorf("parse backend database: %w", err)
	}

	entries := make(map[string]string, len(raw))
	for key, value := range raw {
		normalized := strings.ToLower(strings.TrimSpace(key))
		if normalized == "" {
			continue
		}
		entries[normalized] = value
	}

	return Database{entries: entries}, nil
}

func (db Database) Lookup(key string) (string, bool) {
	if db.entries == nil {
		return "", false
	}
	normalized := strings.ToLower(strings.TrimSpace(key))
	value, ok := db.entries[normalized]
	return value, ok
}

func (db Database) Available() bool {
	return len(db.entries) > 0
}
