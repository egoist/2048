package main

import "encoding/json"

// saved is what the app keeps between launches: the best score, and the
// game in progress, if any.
type saved struct {
	Best int        `json:"best"`
	Game *savedGame `json:"game,omitempty"`
}

// decodeSaved returns what data saved, or nothing when it is damaged: a
// damaged save starts a new game rather than failing.
func decodeSaved(data []byte) saved {
	var v saved
	if err := json.Unmarshal(data, &v); err != nil {
		return saved{}
	}
	return v
}
