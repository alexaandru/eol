//nolint:cyclop // ok
package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/alexaandru/eol"
)

//go:embed help.txt
var helpText string

func printHeader() { fmt.Println("eol - EndOfLife.date API client") }
func printUsage()  { fmt.Println(helpText) }

func main() {
	var err error

	defer func() {
		if err == nil {
			return
		}

		if errors.Is(err, eol.ErrNeedHelp) {
			printHeader()
			fmt.Println()
			printUsage()

			return
		}

		fmt.Printf("Error: %v!\n\n", err)
		printUsage()
		os.Exit(1)
	}()

	client, err := eol.New()
	if err != nil {
		return
	}

	err = client.Handle()
}
