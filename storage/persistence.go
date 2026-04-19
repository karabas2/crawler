package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SaveState persists the current storage state to a JSON file.
func (s *Storage) SaveState(path string) error {
	s.Mu.RLock()
	defer s.Mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadState restores the storage state from a JSON file.
func (s *Storage) LoadState(path string) error {
	s.Mu.Lock()
	defer s.Mu.Unlock()

	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(file, s)
}
