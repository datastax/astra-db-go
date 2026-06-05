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

	"github.com/DeanPDX/dotconfig"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/fatih/color"
)

const (
	DefaultCollectionName = "default_collection"
	DefaultTableName      = "default_table"
	DefaultUDTName        = "example_udt"
)

type config struct {
	// flags
	local       bool
	skipPrelude bool

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
