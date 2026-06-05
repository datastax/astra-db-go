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
	"context"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/datastax/astra-db-go/internal/testlib"
	"github.com/fatih/color"
)

// T contains all test-spec data and utilities, similar to testing.T
type T struct {
	*TestObjects
	Name string
	Ctx  context.Context
	logs strings.Builder
}

// failSignal is used internally to differentiate between a true panic, and an intentional test failure (via t.Fatalf).
// Is this abuse of control flow? Maybe. Do I care? no.
type failSignal struct {
	msg string
}

// Helper is just used for interface compatability w/ testlib.HasFatal, and doesn't really have a reason to exist otherwise
func (t *T) Helper() {}

func (t *T) Log(format string) {
	_, _ = fmt.Fprintln(&t.logs, format)
}

func (t *T) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(&t.logs, format, args...)
}

func (t *T) Fatalf(format string, args ...any) {
	panic(failSignal{msg: fmt.Sprintf(format, args...)})
}

func (t *T) NoDiff(want, got any) {
	if diff := testlib.Diff(want, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

type test struct {
	name string
	run  func(*T)
}

var suites []*S
var backgroundTests []test

func BackgroundTest(name string, fn func(*T)) {
	backgroundTests = append(backgroundTests, test{name, fn})
}

type S struct {
	Name     string
	Parallel bool
	tests    []test
}

func Suite(name string) *S {
	s := &S{Name: name}
	suites = append(suites, s)
	return s
}

func ParallelSuite(name string) *S {
	s := &S{Name: name, Parallel: true}
	suites = append(suites, s)
	return s
}

func (s *S) Run(name string, fn func(*T)) *S {
	s.tests = append(s.tests, test{name, fn})
	return s
}

type failure struct {
	testName string
	message  string
	isPanic  bool
}

var (
	failuresMu    sync.Mutex
	suiteFailures = make(map[string][]failure)
)

func Run() int {
	var bgWg sync.WaitGroup
	var bgOut strings.Builder
	var suiteWg sync.WaitGroup

	for _, t := range backgroundTests {
		runTestParallel(&bgOut, "{Background}", t, &bgWg)
	}

	for i, t := range suites {
		fmt.Printf("\n%s %s\n", Bold(Highlight(fmt.Sprintf("%d)", i+1))), Bold(t.Name))

		for _, test := range t.tests {
			if t.Parallel {
				runTestParallel(os.Stdout, t.Name, test, &suiteWg)
			} else {
				executeTest(os.Stdout, t.Name, test)
			}
		}
		suiteWg.Wait()
	}

	bgWg.Wait()

	return printFailures()
}

func printFailures() int {
	failuresMu.Lock()
	defer failuresMu.Unlock()

	if len(suiteFailures) == 0 {
		PrintlnBold(color.GreenString("\n✓ All tests passed.\n"))
		return 0
	}

	totalFailures := 0
	var suiteNames []string
	for name, fs := range suiteFailures {
		totalFailures += len(fs)
		suiteNames = append(suiteNames, name)
	}

	sort.Strings(suiteNames)

	PrintlnBold(color.RedString("\n✘ %d test(s) failed:\n", totalFailures))

	i := 1
	for _, suiteName := range suiteNames {
		for _, f := range suiteFailures[suiteName] {
			fmt.Printf("  %d) %s %s:\n", i, suiteName, f.testName)
			lines := strings.Split(f.message, "\n")
			for _, line := range lines {
				if f.isPanic {
					fmt.Printf("     %s\n", color.YellowString(line))
				} else {
					fmt.Printf("     %s\n", color.RedString(line))
				}
			}
			fmt.Println()
			i++
		}
	}

	return 1
}

func executeTest(w io.Writer, suiteName string, tst test) {
	t := mkT(tst)

	printRes := func(symbol string) {
		FprintlnChecklist(w, fmt.Sprintf("%s %s", symbol, tst.name))
		if t.logs.Len() > 0 {
			FprintlnNestedChecklist(w, t.logs.String())
		}
	}

	defer func() {
		if r := recover(); r != nil {
			failuresMu.Lock()
			defer failuresMu.Unlock()

			if fs, ok := r.(failSignal); ok {
				printRes(color.RedString("✘"))
				suiteFailures[suiteName] = append(suiteFailures[suiteName], failure{
					testName: tst.name,
					message:  fs.msg,
				})
			} else {
				printRes(color.YellowString("!"))
				suiteFailures[suiteName] = append(suiteFailures[suiteName], failure{
					testName: tst.name,
					message:  fmt.Sprintf("%v\n%s", r, debug.Stack()),
					isPanic:  true,
				})
			}
		}
	}()

	tst.run(&t)

	printRes(color.GreenString("✓"))
}

func runTestParallel(w io.Writer, suiteName string, t test, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		executeTest(w, suiteName, t)
	}()
}

func mkT(t test) T {
	return T{NewTestObjects(), t.name, context.Background(), strings.Builder{}}
}
