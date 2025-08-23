package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
)

//go:embed help.txt
var helpText string

func main() {
	var (
		c   *client
		err error
	)

	defer func() {
		switch {
		case err == nil:
		case errors.Is(err, ErrUsage):
			msg := err.Error()
			msg, _ = strings.CutPrefix(msg, "usage error: ")
			fmt.Printf("Error: %v!\n\n", msg)
			c.printUsage()
			os.Exit(1)
		default:
			fmt.Printf("Error: %v!\n", err)
			os.Exit(2)
		}
	}()

	c, err = New(os.Args[1:])
	if err != nil {
		return
	}

	err = c.Handle()
}
