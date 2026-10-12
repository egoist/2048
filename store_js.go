//go:build js

package main

import (
	"encoding/json"
	"errors"
	"syscall/js"
)

// store keeps saved in the browser's localStorage, as browsers have no
// data directory for a file.
type store struct {
	storage js.Value
	key     string
}

func openStore() (s *store, err error) {
	// Reading localStorage throws where the browser blocks site data.
	defer func() {
		if recover() != nil {
			s, err = nil, errors.New("localStorage is not available")
		}
	}()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return nil, errors.New("localStorage is not available")
	}
	return &store{storage: storage, key: "2048"}, nil
}

func (s *store) load() (v saved) {
	defer func() {
		if recover() != nil {
			v = saved{}
		}
	}()
	item := s.storage.Call("getItem", s.key)
	if item.Type() != js.TypeString {
		return saved{}
	}
	return decodeSaved([]byte(item.String()))
}

func (s *store) save(v saved) (err error) {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	// Writing throws when the storage is full or blocked.
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("localStorage refused the save")
		}
	}()
	s.storage.Call("setItem", s.key, string(data))
	return nil
}
