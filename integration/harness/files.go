package harness

import (
	"os"
	"path/filepath"
	"runtime"
)

var TestsDirectory string

func init() {
	_, file, _, _ := runtime.Caller(0)
	TestsDirectory = filepath.Join(filepath.Dir(filepath.Dir(file)), "tests")

	if _, err := os.Stat(TestsDirectory); err != nil {
		panic("Tests directory not found: " + TestsDirectory)
	}
}

func callerPath(skip int) string {
	_, file, _, _ := runtime.Caller(skip + 1)
	return file
}
