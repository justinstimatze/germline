// Package declining stands in for a decision record that carries its
// system's declined promises, in the two shapes a hand-written store gets
// written in. Under testdata, so the go tool never builds it.
package declining

type Entity struct {
	Name  string
	Brief string
}

type Concept struct{ *Entity }

type DeclinedPromise struct {
	System     string
	Observable string
	Since      string
	Reason     string
}

// Declines is the claim. A real store declares it over a generic relation
// with provenance; the reader recognises the shape, not the declaration.
type Declines struct {
	Subject *Entity
	Object  *DeclinedPromise
}

// ValueDeclines is the same claim by value, with the payload inline rather
// than behind a pointer. Both shapes read the same.
type ValueDeclines struct {
	Subject Concept
	Object  DeclinedPromise
}

var PlusDecodingIsNotPromised = Concept{&Entity{
	Name:  "Plus Decoding Is Not Promised",
	Brief: "Form submission and RFC 3986 disagree and both ship.",
}}

var EscapeCaseIsNotPromised = Concept{&Entity{
	Name:  "Escape Case Is Not Promised",
	Brief: "%2F and %2f are equivalent; compare parsed queries.",
}}

// DeclinesPlusDecoding takes the subject through a selector and the payload
// through an address-of, and splits its reason across two literals the way
// gofmt leaves a sentence that outgrew a line.
var DeclinesPlusDecoding = Declines{
	Subject: PlusDecodingIsNotPromised.Entity,
	Object: &DeclinedPromise{
		System:     "querystring",
		Observable: "How `+` is decoded",
		Since:      "0.1.0",
		Reason: "Form submission reads it as a space and RFC 3986 reads it " +
			"literally; both ship.",
	},
}

// DeclinesEscapeCase names its subject bare, holds its payload by value, and
// writes its reason as one raw string.
var DeclinesEscapeCase = ValueDeclines{
	Subject: EscapeCaseIsNotPromised,
	Object: DeclinedPromise{
		System:     "querystring",
		Observable: "The case of hex digits in a rendered escape",
		Since:      "0.1.0",
		Reason:     `RFC 3986 makes the two equivalent.`,
	},
}

// DeclinesForAnotherSystem shares the record with the two above. One store
// serves several systems, and a projection takes only its own.
var DeclinesForAnotherSystem = Declines{
	Subject: PlusDecodingIsNotPromised.Entity,
	Object: &DeclinedPromise{
		System:     "somethingelse",
		Observable: "How `+` is decoded",
		Since:      "2.0",
		Reason:     "A different system, and the same observable means something else in it.",
	},
}

// declinesDraft is unexported: not a memory, and so not a decline. A promise
// nobody can cite has not been withdrawn.
var declinesDraft = Declines{
	Subject: EscapeCaseIsNotPromised.Entity,
	Object: &DeclinedPromise{
		System:     "querystring",
		Observable: "Something still being thought about",
		Since:      "0.1.0",
		Reason:     "Not decided yet.",
	},
}
