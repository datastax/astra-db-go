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
