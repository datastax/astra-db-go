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

package results

import "github.com/datastax/astra-db-go/astra/table"

// UDTDescriptor represents the descriptor for a user-defined type, including its name and definition.
type UDTDescriptor struct {
	// Name of the user-defined type.
	Name string `json:"name"`

	// Definition of the user-defined type.
	// Only populated when listTypes is called with explain=true.
	Definition table.UDTDefinition `json:"definition"`
}
