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
func (c command) MarshalJSON() ([]byte, error) {
	if len(c.name) > 0 {
		data := make(map[string]any)
		data[c.name] = c.payload
		//return json.Marshal(data)
		return serdes.Serialize(data, serdes.TargetUnknown)
	}
	return serdes.Serialize(c.payload, serdes.TargetUnknown)
}

func (c command) MarshalAstraRaw(_ serdes.Target, data []byte) ([]byte, error) {
	b, err := c.MarshalJSON()
	if err != nil {
		return data, err
	}
	return append(data, b...), nil
}
