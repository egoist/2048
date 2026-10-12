package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// saved is what the app keeps between launches: the best score, and the
// game in progress, if any.
type saved struct {
	Best int        `json:"best"`
	Game *savedGame `json:"game,omitempty"`
}

// store keeps saved in a JSON file.
type store struct {
	path string
}

// load returns what was saved, or nothing when the file is missing or
// unreadable: a damaged save starts a new game rather than failing.
func (s *store) load() saved {
	var v saved
	data, err := os.ReadFile(s.path)
	if err != nil {
		return saved{}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return saved{}
	}
	return v
}

// save writes v to a temporary file and renames it over the last save,
// so a crash midway leaves the last save whole.
func (s *store) save(v saved) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), ".state-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}
