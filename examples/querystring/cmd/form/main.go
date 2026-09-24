// Command form is version 0.1.0 of the sample system as a replay subject: it
// reads one recorded query string on stdin and writes what it made of it on
// stdout. That contract is the whole port, and it is why a C binary or a
// Python service could stand here instead.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/justinstimatze/germline/examples/querystring"
)

func main() {
	if err := run(querystring.FormCodec{}, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "form:", err)
		os.Exit(1)
	}
}

// run is shared with the other subject so the two differ only in their codec.
func run(c querystring.Codec, in io.Reader, out io.Writer) error {
	raw, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	q, err := c.Parse(string(raw))
	if err != nil {
		return err
	}
	fmt.Fprintln(out, q.Canonical())
	fmt.Fprintln(out, c.Render(q))
	return nil
}
