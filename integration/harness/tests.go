package harness

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/datastax/astra-db-go/internal/testlib"
	"github.com/fatih/color"
)

type T struct {
	*TestObjects
	Name string
	Ctx  context.Context
}

type failSignal struct {
	msg string
}

func (t T) Helper() {}

func (t T) Fatalf(format string, args ...any) {
	panic(failSignal{msg: fmt.Sprintf(format, args...)})
}

func (t T) NoDiff(want, got any) {
	if diff := testlib.Diff(want, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}

type test struct {
	name string
	run  func(T)
}

var suites []*S
var backgroundTests []test

func BackgroundTest(name string, fn func(T)) {
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

func (s *S) Run(name string, fn func(T)) *S {
	s.tests = append(s.tests, test{name, fn})
	return s
}

func Run() int {
	var bgWg sync.WaitGroup
	var bgOut strings.Builder
	var suiteWg sync.WaitGroup
	var failures int32

	for _, t := range backgroundTests {
		runTestParallel(&bgOut, t, &bgWg, &failures)
	}

	for i, t := range suites {
		fmt.Printf("\n%s %s\n", Bold(Highlight(fmt.Sprintf("%d)", i+1))), Bold(t.Name))

		for _, test := range t.tests {
			if t.Parallel {
				runTestParallel(os.Stdout, test, &suiteWg, &failures)
			} else {
				executeTest(nil, test, &failures)
			}
		}
		suiteWg.Wait()
	}

	bgWg.Wait()

	if atomic.LoadInt32(&failures) != 0 {
		PrintlnBold(color.RedString("\n✘ %d test(s) failed.\n", failures))
		return 1
	}

	PrintlnBold(color.GreenString("\n✓ All tests passed.\n"))
	return 0
}

func executeTest(w io.Writer, t test, failed *int32) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(failSignal); ok {
				FprintlnChecklist(w, fmt.Sprintf("%s %s", color.RedString("✘"), t.name))
				//fmt.Printf("  %s\n\n", fs.msg)
			} else {
				FprintlnChecklist(w, fmt.Sprintf("%s %s", color.YellowString("!"), t.name))
				//fmt.Printf("  %v\n\n", r)
				//debug.PrintStack()
			}
			atomic.AddInt32(failed, 1)
		}
	}()

	t.run(mkT(t))

	FprintlnChecklist(w, fmt.Sprintf("%s %s", color.GreenString("✓"), t.name))
}

func runTestParallel(w io.Writer, t test, wg *sync.WaitGroup, failed *int32) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		executeTest(w, t, failed)
	}()
}

func mkT(t test) T {
	return T{NewTestObjects(), t.name, context.Background()}
}
