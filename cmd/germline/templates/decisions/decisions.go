// Package decisions is this project's decision record: one memory per
// decision, superseded rather than edited, written for a reader with no
// context.
//
// It is Go source because germline reads it as Go source. A memory is an
// exported top-level variable; the compiler checks that every citation between
// memories resolves, and germline checks that every citation from BOUNDARY.md
// and from an accepted replay difference names a memory declared here. Nothing
// here is imported by the project, and a project in another language can keep
// this directory without building it.
//
// A memory:
//
//	// RetryBudgetIsPerRequest records why retries are budgeted per request.
//	var RetryBudgetIsPerRequest = Concept{&Entity{
//		Name: "Retry Budget Is Per Request",
//		Brief: "2026-01-15: the reason, the alternatives seen, and what " +
//			"would reverse it.",
//	}}
//
// A declined promise, which `germline boundary -write` projects into
// BOUNDARY.md. Once the record carries one, the record is the source of the
// boundary and the file is its projection. Declare them in the order they are
// published: a boundary is appended to, never reordered.
//
//	var DeclinesHeaderOrder = Declines{
//		Subject: HeaderOrderIsNotPromised,
//		Object: &DeclinedPromise{
//			System:     "myservice",
//			Observable: "The order of response headers",
//			Since:      "0.1.0",
//			Reason:     "Clients compare headers by name; the order is incidental.",
//		},
//	}
//
// To change a decision, declare a newer memory whose Brief says what it
// supersedes and why. Do not edit the old one.
package decisions

// An Entity is one memory: what it is called, and the reason written for a
// reader with zero context.
type Entity struct {
	Name  string
	Brief string
}

// A Concept is a memory whose role is a durable idea.
type Concept struct{ *Entity }

// A DeclinedPromise is one observable this system does not promise: the four
// fields BOUNDARY.md publishes.
type DeclinedPromise struct {
	System     string
	Observable string
	Since      string
	Reason     string
}

// Declines says the memory in Subject is the reason the system does not
// promise the observable in Object.
type Declines struct {
	Subject Concept
	Object  *DeclinedPromise
}
