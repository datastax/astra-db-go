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

// TableDescriptor represents the descriptor for a table, including its name and definition.
type TableDescriptor struct {
	// Name of the table.
	Name string `json:"name"`

	// Definition of the table (columns and primary key).
	// Only populated when listTables is called with explain=true.
	Definition table.Definition `json:"definition"`
}
