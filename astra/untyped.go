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

package astra

import (
	"github.com/datastax/astra-db-go/v2/astra/internal/untyped"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/table"
)

// Document represents an untyped document used for a collection operation (as opposed to using a specific struct).
type Document = untyped.Document

// Row represents an untyped row used for a table operation (as opposed to using a specific struct).
type Row = untyped.Row

// NewDocument is a map-based implementation of [Document], primarily used for insertion.
type NewDocument = untyped.NewDocument

// NewRow is a map-based implementation of [Row], primarily used for insertion.
type NewRow = untyped.NewRow

func NewDocumentTargetCtx() serdes.TargetDecodeCtx {
	return untyped.NewDocumentTargetCtx()
}

func NewRowTargetCtx(cols table.Columns) serdes.TargetDecodeCtx {
	return untyped.NewRowTargetCtx(cols)
}
