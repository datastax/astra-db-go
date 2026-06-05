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
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

func Bold(msg string) string {
	return color.New(color.Bold).Sprint(msg)
}

func Highlight(msg string) string {
	return color.RGB(121, 200, 255).Sprint(msg)
}

func PrintlnBold(msg string) {
	fmt.Println(Bold(msg))
}

func PrintlnChecklist(msg string) {
	FprintlnChecklist(os.Stdout, msg)
}

func FprintlnChecklist(w io.Writer, msg string) {
	prefix := color.RGB(121, 200, 255).Sprint("└─> ")
	_, _ = fmt.Fprintf(w, "%s%s\n", prefix, msg)
}

func PrintlnNestedChecklist(msg string) {
	pipe := Highlight("│")
	prefix := color.New(color.Faint).Sprint(Highlight("└─> "))
	text := color.New(color.Faint).Sprintln(msg)
	fmt.Printf("%s%s%s", pipe, prefix, text)
}
