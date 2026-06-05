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
	"strings"

	"github.com/fatih/color"
)

// Don't have a great way of making this file look nice or have consistent functions
// I'm just adding what I need to add at that very moment ¯\_(ツ)_/¯

var bold = color.New(color.Bold)
var faint = color.New(color.Faint)
var highlight = color.RGB(111, 190, 245)

func Bold(msg string) string {
	return bold.Sprint(msg)
}

func Faint(msg string) string {
	return faint.Sprint(msg)
}

func Highlight(msg string) string {
	return highlight.Sprint(msg)
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
	FprintlnNestedChecklist(os.Stdout, msg)
}

func FprintlnNestedChecklist(w io.Writer, msg string) {
	pipe := Highlight("│")
	prefix := Faint(Highlight("└─> "))

	msg = strings.TrimRight(msg, "\n")
	if !strings.Contains(msg, "\n") {
		_, _ = fmt.Fprintf(w, "%s%s%s\n", pipe, prefix, Faint(msg))
		return
	}

	lines := strings.Split(msg, "\n")
	for i, line := range lines {
		if i != 0 {
			prefix = "    "
		}
		lines[i] = fmt.Sprintf("%s%s%s%s", pipe, prefix, Faint("│ "), Faint(line))
	}
	_, _ = fmt.Fprintln(w, strings.Join(lines, "\n"))
}
