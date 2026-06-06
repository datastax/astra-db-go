package harness

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

var TestsDirectory string

func init() {
	_, file, _, _ := runtime.Caller(0)
	TestsDirectory = filepath.Join(filepath.Dir(filepath.Dir(file)), "tests")

	if _, err := os.Stat(TestsDirectory); err != nil {
		panic("Tests directory not found: " + TestsDirectory)
	}
}

func callerDir(skip int) string {
	_, file, _, _ := runtime.Caller(skip + 1)
	dir, _ := filepath.Rel(TestsDirectory, filepath.Dir(file))
	return dir
}

func lowestNonHelperCallerInfo(mu *sync.RWMutex, helpers map[string]bool) (file string, line int) {
	var pcs [50]uintptr
	n := runtime.Callers(2, pcs[:])
	if n > 0 {
		mu.RLock()
		defer mu.RUnlock()

		frames := runtime.CallersFrames(pcs[:n])
		for {
			frame, more := frames.Next()
			if !helpers[frame.Function] {
				file = frame.File
				line = frame.Line
				break
			}
			if !more {
				break
			}
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, file); err == nil {
			file = rel
		}
	}

	return
}
