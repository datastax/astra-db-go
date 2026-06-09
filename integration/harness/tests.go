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
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/datastax/astra-db-go/v2/astra/datatypes"
	"github.com/fatih/color"
)

// T contains all test-spec data and utilities, similar to testing.T
type T struct {
	*TestObjects
	Name    string
	Ctx     context.Context
	logs    strings.Builder
	mu      sync.RWMutex
	helpers map[string]bool
	key     string
}

// Helper marks the calling function as a test helper function.
// When printing file and line information, that function will be skipped.
func (t *T) Helper() {
	t.mu.Lock()
	defer t.mu.Unlock()

	var pcs [1]uintptr
	n := runtime.Callers(2, pcs[:])
	if n == 0 {
		return
	}

	frames := runtime.CallersFrames(pcs[:n])
	frame, _ := frames.Next()

	t.helpers[frame.Function] = true
}

func (t *T) Log(items ...any) {
	_, _ = fmt.Fprintln(&t.logs, items...)
}

func (t *T) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(&t.logs, format, args...)
}

// failSignal is used internally to differentiate between a true panic, and an intentional test failure (via t.Fatalf).
// Is this abuse of control flow? Maybe. Do I care? no.
type failSignal struct {
	msg  string
	file string
	line int
}

func (t *T) Fatalf(format string, args ...any) {
	file, line := lowestNonHelperCallerInfo(&t.mu, t.helpers)

	panic(failSignal{
		msg:  fmt.Sprintf(format, args...),
		file: file,
		line: line,
	})
}

func (t *T) Key(seed int) string {
	return fmt.Sprintf("k_%d", seed)
}

var suites = []*S{
	{Name: "{Background}", Type: suiteBackground},
}

type S struct {
	Name   string
	Type   suiteType
	dir    string
	tests  []test
	before func(*T)
	after  func(*T)
}

type suiteType int

const (
	suiteParallel suiteType = iota
	suiteSequential
	suiteBackground
)

type test struct {
	name string
	run  func(*T)
}

func ParallelSuite(name string) *S {
	return appendSuite(name, suiteParallel)
}

func SequentialSuite(name string) *S {
	return appendSuite(name, suiteSequential)
}

func BackgroundSuite() *S {
	return suites[0]
}

var kebabRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func appendSuite(name string, typ suiteType) *S {
	if !kebabRegex.MatchString(name) {
		panic(fmt.Sprintf("suite name '%s' is not in kebab-case", name))
	}
	s := &S{Name: name, Type: typ, dir: callerDir(2)}
	suites = append(suites, s)
	return s
}

func (s *S) Run(name string, fn func(*T)) *S {
	s.tests = append(s.tests, test{name, fn})
	return s
}

func (s *S) Before(fn func(*T)) *S {
	old := s.before
	s.before = func(t *T) {
		if old != nil {
			old(t)
		}
		fn(t)
	}
	return s
}

func (s *S) After(fn func(*T)) *S {
	old := s.after
	s.after = func(t *T) {
		fn(t)
		if old != nil {
			old(t)
		}
	}
	return s
}

func Run() int {
	suitesToRun, testsRun := filterTests()

	var bgWg sync.WaitGroup
	var bgOut strings.Builder

	if len(suitesToRun) > 0 && suitesToRun[0].Type == suiteBackground {
		bgWg.Add(1)
		go func() {
			defer bgWg.Done()
			executeSuite(&bgOut, suitesToRun[0], len(suitesToRun))
		}()
		suitesToRun = suitesToRun[1:]
	}

	for i, s := range suitesToRun {
		executeSuite(os.Stdout, s, i+1)
	}

	bgWg.Wait()
	fmt.Print(bgOut.String())

	return printResults(testsRun)
}

func filterTests() (suitesToRun []*S, testsRun int) {
	for _, s := range suites {
		var tests []test
		for _, t := range s.tests {
			if ShouldRun(s, t.name) {
				tests = append(tests, t)
			}
		}
		if len(tests) > 0 {
			s.tests = tests
			suitesToRun = append(suitesToRun, s)
			testsRun += len(tests)
		}
	}
	return
}

func executeSuite(w io.Writer, s *S, i int) {
	_, _ = fmt.Fprintf(w, "\n%s %s\n", Bold(Highlight(fmt.Sprintf("%d)", i))), Bold(filepath.Base(s.dir)+"/"+s.Name))

	beforeSucceeded := true
	if s.before != nil {
		beforeSucceeded = executeTestSync(w, s.Name, test{"{Before}", s.before}, true)
	}

	if beforeSucceeded {
		var internalWg sync.WaitGroup
		for _, test := range s.tests {
			if s.Type == suiteSequential {
				executeTestSync(w, s.Name, test, false)
			} else {
				executeTestAsync(w, s.Name, test, &internalWg)
			}
		}
		internalWg.Wait()
	}

	if s.after != nil {
		executeTestSync(w, s.Name, test{"{After}", s.after}, true)
	}
}

type failure struct {
	testName string
	message  string
	isPanic  bool
}

var (
	resultsMu     sync.Mutex
	suiteFailures = make(map[string][]failure)
)

func executeTestSync(w io.Writer, suiteName string, tst test, silent bool) (success bool) {
	success = true
	t := mkT(tst, NewTestObjects())

	printRes := func(symbol string) {
		if silent && symbol == color.GreenString("✓") && t.logs.Len() == 0 {
			return
		}

		resultsMu.Lock()
		defer resultsMu.Unlock()

		FprintlnChecklist(w, fmt.Sprintf("%s %s", symbol, tst.name))
		if t.logs.Len() > 0 {
			FprintlnNestedChecklist(w, t.logs.String())
		}
	}

	defer func() {
		if r := recover(); r != nil {
			success = false

			var f failure
			if fs, ok := r.(failSignal); ok {
				printRes(color.RedString("✘"))
				f = failure{
					testName: tst.name + Faint(fmt.Sprintf(" (%s:%d)", fs.file, fs.line)),
					message:  fs.msg,
				}
			} else {
				printRes(color.YellowString("!"))
				f = failure{
					testName: tst.name,
					message:  fmt.Sprintf("%v\n%s", r, debug.Stack()),
					isPanic:  true,
				}
			}

			resultsMu.Lock()
			suiteFailures[suiteName] = append(suiteFailures[suiteName], f)
			resultsMu.Unlock()
		}
	}()

	tst.run(&t)

	printRes(color.GreenString("✓"))
	return
}

func executeTestAsync(w io.Writer, suiteName string, t test, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		executeTestSync(w, suiteName, t, false)
	}()
}

func mkT(t test, fixtures *TestObjects) T {
	return T{
		fixtures, t.name, context.Background(), strings.Builder{},
		sync.RWMutex{}, make(map[string]bool), datatypes.NewObjectId().String(),
	}
}

func printResults(testsRun int) int {
	if testsRun == 0 {
		PrintlnBold(color.YellowString("\n! No tests were run.\n"))
		return 0
	}

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
			fmt.Printf("  %s – %s\n", Bold(fmt.Sprintf("%d) %s", i, suiteName)), f.testName)
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
