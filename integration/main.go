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

package main

//go:generate go run -modfile=../tools/gen-integration/go.mod ../tools/gen-integration/main.go ./tests/...

import (
	"os"

	"github.com/datastax/astra-db-go/v2/integration/harness"
	"github.com/datastax/astra-db-go/v2/integration/legacy"
)

func main() {
	harness.Init()
	exitCode, skipLegacy := harness.Run()

	if exitCode != 0 || skipLegacy {
		os.Exit(exitCode)
	}

	harness.PrintlnBold(harness.Highlight("...Running legacy integration tests...\n"))
	legacy.Run() // temporary
}
