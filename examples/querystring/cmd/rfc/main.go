// Command rfc is version 0.2.0 of the sample system as a replay subject: the
// same contract as cmd/form, reading one recorded query string on stdin and
// writing what it made of it on stdout. The two differ in one field of one
// function, and the replay between them is what finds out whether that
// difference is a defect, a missing law, or something never promised.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/justinstimatze/germline/examples/querystring"
)

func main() {
	if err := run(querystring.RFCCodec{}, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "rfc:", err)
		os.Exit(1)
	}
}

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
