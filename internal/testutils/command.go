package testutils

import (
	"encoding/json"
)

// command is a proxy for unexported command in main package
type command struct {
	name    string
	payload any
}

// NewTestCmd creates a new command for testing JSON serialization and such.
func NewTestCmd(name string, payload any) command {
	return command{
		name:    name,
		payload: payload,
	}
}

// Check out rationale for this in main astradb package.
func (c command) MarshalJSON() ([]byte, error) {
	if len(c.name) > 0 {
		data := make(map[string]any)
		data[c.name] = c.payload
		return json.Marshal(data)
	}
	return json.Marshal(c.payload)
}
