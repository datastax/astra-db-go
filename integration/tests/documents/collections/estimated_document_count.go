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

package collections

import (
	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/internal/testlib"
)

func init() {
	s := harness.SequentialSuite("estimated-document-count")

	s.Run("roughly works", func(t *harness.T) {
		count, err := t.Collection.EstimatedDocumentCount(t.Ctx)
		testlib.FailIfErr(t, err, "EstimatedDocumentCount failed: %v", err)
		testlib.FailIf(t, count < 0, "expected count to be >= 0, got %d", count)
	})
}
