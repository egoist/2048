//go:build !js

package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
)

// store keeps saved in a JSON file.
type store struct {
	path string
}

// openStore returns the store in the app's data directory.
func openStore() (*store, error) {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return nil, err
	}
	return &store{path: filepath.Join(dir, "state.json")}, nil
}

func (s *store) load() saved {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return saved{}
	}
	return decodeSaved(data)
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
