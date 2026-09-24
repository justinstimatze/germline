// Package store stands in for a winze decision store: a Go module whose
// top-level variable declarations are its memories. Under testdata, so the go
// tool never builds it.
package store

// Entity is the shape a winze memory carries.
type Entity struct {
	Name  string
	Brief string
}

// Concept wraps an entity, the way a real store's role types do.
type Concept struct{ *Entity }

// GermlineThreeWayClose is one memory.
var GermlineThreeWayClose = Concept{&Entity{
	Name:  "Germline Three Way Close",
	Brief: "Every replay difference closes exactly one of three ways.",
}}

// GermlineCorpusInputsAppendOnlyOutputsPerVersion is another.
var GermlineCorpusInputsAppendOnlyOutputsPerVersion = Concept{&Entity{
	Name:  "Germline Corpus Inputs Append Only Outputs Per Version",
	Brief: "Inputs are append-only; outputs are re-derived per version.",
}}

// unexportedHelper is not a memory and must not be listed.
var unexportedHelper = "ignored"
