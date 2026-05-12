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

import (
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/datastax/astra-db-go/internal/integrationtests/harness"
	// This import triggers the init() functions in tests. If you put tests
	// in other packages, be sure to side-effect import them.
	_ "github.com/datastax/astra-db-go/internal/integrationtests/tests"
)

// Get the test environment as well as all tests, then run them.
// Tests should be of type [harness.IntegrationTest]:
//
//	func CreateCollection(e *harness.TestEnv) error { //... }
//
// Tests should register themselves with the test harness in an `init`
// function:
//
//	func init() {
//		// Register a single test that doesn't depent on prior tests
//		harness.Register(harness.IntegrationTest{
//			Name: "ListCollections", Run: ListCollections,
//		})
//		// Register multiple tests (order will be preserved).
//		t := []harness.IntegrationTest{
//			{Name: "CreateCollection", Run: CreateCollection},
//			{Name: "GetCollection", Run: GetCollection},
//			{Name: "DropCollection", Run: DropCollection},
//		}
//		harness.Register(t...)
//	 }
func main() {
	// Create a file to log to and ensure we close it on exit
	logFile := createLogFile()
	defer logFile.Close()
	// We will log to file and stdout
	setupLogging(os.Stdout, logFile)
	// Get the test environment
	e := harness.Environment()
	// And all tests
	tests := harness.Tests()
	ran := 0
	skipped := 0

	totalStart := time.Now()
	// Then run them
	for _, test := range tests {
		if len(e.TestPrefix) > 0 && !strings.HasPrefix(test.Name, e.TestPrefix) {
			slog.Info(test.Name, "status", "SKIPPED")
			skipped++
			continue
		}
		slog.Info(test.Name, "status", "RUNNING")
		start := time.Now()
		// Run the test, paying attention to elapsed time.
		err := test.Run(&e)
		elapsed := time.Since(start)
		// Show a result
		if err == nil {
			ran++
			slog.Info(test.Name, "status", "PASS", "elapsed", elapsed)
		} else {
			slog.Error(test.Name, "status", "FAIL", "elapsed", elapsed, "error", err)
			slog.Error("Tests failed")
			os.Exit(1)
		}
	}
	slog.Info("All tests passed", "elapsed", time.Since(totalStart), "ran", ran, "skipped", skipped)
}

// createLogFile ensures ./logs exists and creates a timestamped log file
func createLogFile() *os.File {
	// Ensure logs dir exists
	os.Mkdir(path.Join(".", "logs"), os.ModePerm)
	// Create a log file
	logFilePath := path.Join(".", "logs", time.Now().Format("2006-01-02_150405.log"))
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("os.OpenFile", "error", err)
		return nil
	}
	return logFile
}

// Set up logging for our test run. Will log to any writers in variadic input.
func setupLogging(writers ...io.Writer) {
	multiWriter := io.MultiWriter(writers...)
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // We want debug logs
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the time format
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				return slog.String(slog.TimeKey, t.Format("2006/01/02 15:04:05.000"))
			}
			return a
		},
	}
	handler := slog.NewTextHandler(multiWriter, opts)
	// Create our structured logger and set it as default
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
