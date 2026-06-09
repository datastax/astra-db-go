// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testlib

import (
	"github.com/datastax/astra-db-go/v2/astra/serdes"
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

// Check out rationale for this in main astra package.
func (c command) MarshalAstraRaw(ctx serdes.EncodeCtx, dst []byte) ([]byte, error) {
	if len(c.name) > 0 {
		data := make(map[string]any)
		data[c.name] = c.payload
		return serdes.SerializeInto(data, serdes.TargetNone, dst, ctx.Flags)
	}
	return serdes.SerializeInto(c.payload, serdes.TargetNone, dst, ctx.Flags)
}
