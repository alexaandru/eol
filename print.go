package eol

import "fmt"

// Printf formats according to a format specifier and writes to c.sink.
func (c *Client) Printf(format string, a ...any) (n int, err error) {
	return fmt.Fprintf(c.sink, format, a...)
}

// Print formats using the default formats for its operands and writes to c.sink.
func (c *Client) Print(a ...any) (n int, err error) {
	return fmt.Fprint(c.sink, a...)
}

// Println formats using the default formats for its operands and writes to c.sink.
func (c *Client) Println(a ...any) (n int, err error) {
	return fmt.Fprintln(c.sink, a...)
}
