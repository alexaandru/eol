//nolint:cyclop // ok
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alexaandru/eol"
)

//go:embed help.txt
var helpText string

func printHeader() { fmt.Println("eol - EndOfLife.date API client") }
func printUsage()  { fmt.Println(helpText) }

func main() {
	var err error

	defer func() {
		switch {
		case err == nil:
		case errors.Is(err, eol.ErrNeedHelp):
			printHeader()
			fmt.Println()
			printUsage()
		case errors.Is(err, eol.ErrUsage):
			msg := err.Error()
			msg, _ = strings.CutPrefix(msg, "usage error: ")
			fmt.Printf("Error: %v!\n\n", msg)
			printUsage()
			os.Exit(1)
		default:
			fmt.Printf("Error: %v!\n", err)
			os.Exit(2)
		}
	}()

	client, err := eol.New()
	if err != nil {
		return
	}

	err = client.Handle()
}
