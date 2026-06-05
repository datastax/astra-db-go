package harness

import (
	"fmt"

	"github.com/fatih/color"
)

func PrintlnBold(msg string) {
	_, _ = color.New(color.Bold).Println(msg)
}

func PrintlnChecklist(msg string) {
	_, _ = color.RGB(121, 200, 255).Print("└─> ")
	fmt.Println(msg)
}

func PrintlnNestedChecklist(msg string) {
	_, _ = color.RGB(121, 200, 255).Print("│")
	_, _ = color.RGB(121, 200, 255).Add(color.Faint).Print("└─> ")
	_, _ = color.New(color.Faint).Println(msg)
}
