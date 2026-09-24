// Package decisions is the sample system's decision record, in the same shape
// `germline init` scaffolds for a new project.
//
// It is a Go package because that is what a winze store is: the compiler is
// the consistency checker, a memory is a top-level variable, and a decision
// is never edited — a newer one supersedes it. The example runs with no setup:
//
//	germline -C examples/querystring -store examples/querystring/decisions check
package decisions

// An Entity is one memory: what it is called, and the reason written for a
// reader with zero context.
type Entity struct {
	Name  string
	Brief string
}

// A Concept is a memory whose role is a durable idea rather than a person, a
// hypothesis, or an event.
type Concept struct{ *Entity }

// A DeclinedPromise is one observable this system does not promise: the four
// fields BOUNDARY.md publishes, with the memory that carries the reason in
// full held by the Declines claim.
type DeclinedPromise struct {
	System     string
	Observable string
	Since      string
	Reason     string
}

// Declines says the memory in Subject is the reason the system does not
// promise the observable in Object. BOUNDARY.md is generated from these, so
// an entry with no reason cannot be written at all.
//
// The real store names these types the same way and declares them over a
// generic relation with provenance and a temporal marker. germline reads the
// shape rather than the names — an Object carrying an Observable — which is
// what lets this eleven-line stand-in be read by the same code.
type Declines struct {
	Subject Concept
	Object  *DeclinedPromise
}

// QueryPlusDecodingIsNotPromised licenses the difference the two versions
// disagree about.
var QueryPlusDecodingIsNotPromised = Concept{&Entity{
	Name: "Query Plus Decoding Is Not Promised",
	Brief: "2026-09-02: how \"+\" is decoded in a query string is not promised. HTML " +
		"form submission has always decoded it as a space; RFC 3986 treats it as an " +
		"ordinary character. Both readings ship in real software and neither is a " +
		"defect, so a caller who needs a literal plus percent-encodes it and a caller " +
		"who needs a space percent-encodes that. The alternative considered was " +
		"promising the form reading, which is what most callers expect: declined " +
		"because it would make every RFC-conforming implementation a violation and " +
		"there is no way to satisfy both. What would reverse this: if the corpus shows " +
		"callers relying on the form reading in numbers, it becomes a promise in a " +
		"minor version and the RFC codec is the one that has to change.",
}}

// QueryPercentEncodingCaseIsNotPromised covers the hex digits in output.
var QueryPercentEncodingCaseIsNotPromised = Concept{&Entity{
	Name: "Query Percent Encoding Case Is Not Promised",
	Brief: "2026-09-02: whether a rendered escape reads %2F or %2f is not promised. " +
		"RFC 3986 says the two are equivalent and that producers should prefer " +
		"uppercase, which this implementation does, but nothing in the laws pins it " +
		"and a reader accepts both. Declining it keeps byte-comparison of rendered " +
		"output out of the contract, which is where it belongs: compare parsed " +
		"queries. What would reverse this: a caller signing a rendered URL, where the " +
		"bytes are the thing and the case becomes load-bearing.",
}}

// QueryEmptyValueShapeIsNotPromised covers "a" against "a=".
var QueryEmptyValueShapeIsNotPromised = Concept{&Entity{
	Name: "Query Empty Value Shape Is Not Promised",
	Brief: "2026-09-02: whether a key with no \"=\" is distinguishable from a key with " +
		"an empty value is not promised. This implementation parses \"a\" and \"a=\" to " +
		"the same query and renders both as \"a=\". Keeping them apart would need a " +
		"third state in the value type, which every caller would then have to handle " +
		"to get at the common case. The alternative considered was preserving the " +
		"input form through a render: declined because it makes the type harder for " +
		"every caller in order to serve one that has not appeared. What would reverse " +
		"this: a recorded input where the distinction changes what a caller does.",
}}

// The declined promises, in the order BOUNDARY.md publishes them. A new one
// goes at the end: a boundary is appended to, and the projection writes them
// out in the order they are declared here.

// DeclinesPlusDecoding is the first entry in BOUNDARY.md.
var DeclinesPlusDecoding = Declines{
	Subject: QueryPlusDecodingIsNotPromised,
	Object: &DeclinedPromise{
		System:     "querystring",
		Observable: "How `+` is decoded",
		Since:      "0.1.0",
		Reason: "Form submission reads it as a space and RFC 3986 reads it " +
			"literally; both ship, and no rule satisfies both.",
	},
}

// DeclinesPercentEncodingCase is the second.
var DeclinesPercentEncodingCase = Declines{
	Subject: QueryPercentEncodingCaseIsNotPromised,
	Object: &DeclinedPromise{
		System:     "querystring",
		Observable: "The case of hex digits in a rendered escape",
		Since:      "0.1.0",
		Reason: "RFC 3986 makes `%2F` and `%2f` equivalent; compare parsed " +
			"queries, not rendered bytes.",
	},
}

// DeclinesEmptyValueShape is the third, and the one a recorded input would
// reopen first.
var DeclinesEmptyValueShape = Declines{
	Subject: QueryEmptyValueShapeIsNotPromised,
	Object: &DeclinedPromise{
		System:     "querystring",
		Observable: "Whether a key with no `=` differs from one with an empty value",
		Since:      "0.1.0",
		Reason: "Keeping them apart needs a third state in the value type that " +
			"every caller would have to handle.",
	},
}
