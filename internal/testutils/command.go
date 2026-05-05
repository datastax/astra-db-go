package testutils

import (
	"github.com/datastax/astra-db-go/serdes"
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
func (c command) MarshalAstraRaw(_ serdes.Target, dst []byte) ([]byte, error) {
	if len(c.name) > 0 {
		data := make(map[string]any)
		data[c.name] = c.payload
		return serdes.SerializeInto(data, serdes.TargetUnknown, dst)
	}
	return serdes.SerializeInto(c.payload, serdes.TargetUnknown, dst)
}
