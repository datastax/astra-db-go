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

package harness

import (
	"flag"
	"fmt"
	"strings"

	"github.com/DeanPDX/dotconfig"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/fatih/color"
)

const (
	DefaultCollectionName = "default_collection"
	DefaultTableName      = "default_table"
	DefaultUDTName        = "example_udt"
)

// stringSlice is necessary to support multiple -f and -F flags for test filtering
type stringSlice []string

func (i *stringSlice) String() string {
	return strings.Join(*i, ",")
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type config struct {
	// flags
	local       bool
	skipPrelude bool
	include     stringSlice
	exclude     stringSlice

	// env
	apiEndpoint      string `env:"API_ENDPOINT,optional"`
	applicationToken string `env:"APPLICATION_TOKEN,optional"`
	backend          string `env:"BACKEND" default:"astra"`
}

var cfg config

func Init() {
	c, err := dotconfig.FromFileName[config]("../integration/.env")

	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	flag.BoolVar(&c.local, "local", false, "run tests against a local HCD/DSE instance")
	flag.BoolVar(&c.skipPrelude, "P", false, "skip the prelude (setup) step")
	flag.Var(&c.include, "f", "filter tests by name (must include)")
	flag.Var(&c.exclude, "F", "filter tests by name (must not include)")
	flag.Parse()

	if c.local {
		c.backend = "hcd"
	}

	if c.backend != "astra" {
		if c.apiEndpoint == "" {
			c.apiEndpoint = "http://127.0.0.1:8181"
		}
		if c.applicationToken == "" {
			c.applicationToken = "Cassandra:Y2Fzc2FuZHJh:Y2Fzc2FuZHJh"
		}
	}

	cfg = c
	GlobalFixtures = NewTestObjects()

	if !cfg.skipPrelude {
		prelude()
	} else {
		PrintlnBold(color.YellowString("⚠️  Skipping prelude"))
	}
}

func APIEndpoint() string {
	return cfg.apiEndpoint
}

func ApplicationToken() string {
	return cfg.applicationToken
}

func Backend() options.DataAPIBackend {
	return options.DataAPIBackend(cfg.backend) // should probably add validation but eh whatever
}

func ShouldRun(suiteName, testName string) bool {
	if len(cfg.include) == 0 && len(cfg.exclude) == 0 {
		return true
	}

	fullName := suiteName + "/" + testName

	for _, inc := range cfg.include {
		if !strings.Contains(fullName, inc) {
			return false
		}
	}

	for _, exc := range cfg.exclude {
		if strings.Contains(fullName, exc) {
			return false
		}
	}

	return true
}
