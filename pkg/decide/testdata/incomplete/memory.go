// Package incomplete stands in for a record whose decline is missing the one
// field that decides which system it belongs to. Under testdata, so the go
// tool never builds it.
package incomplete

type Entity struct{ Name string }

type Concept struct{ *Entity }

type DeclinedPromise struct {
	System     string
	Observable string
	Since      string
	Reason     string
}

type Declines struct {
	Subject *Entity
	Object  *DeclinedPromise
}

var SomeReason = Concept{&Entity{Name: "Some Reason"}}

// DeclinesWithoutASystem is the failure the reader has to be loud about. No
// system means no boundary file ever carries it, so the promise is withdrawn
// in the record and still published in the file.
var DeclinesWithoutASystem = Declines{
	Subject: SomeReason.Entity,
	Object: &DeclinedPromise{
		Observable: "Something nobody will hear about",
		Since:      "0.1.0",
		Reason:     "Recorded, and invisible to every projection.",
	},
}
