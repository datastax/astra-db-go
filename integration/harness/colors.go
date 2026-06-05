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
