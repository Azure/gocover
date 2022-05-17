package gittool

// Operation defines the operation of a diff item.
type Operation int

const (
	// Equal item represents a equals diff.
	Equal Operation = iota
	// Add item represents an insert or modify diff.
	Add
	// Delete item represents a delete diff.
	Delete
)

// Patch represents a collection of steps to transform several files.
type Patch interface {
	// FilePatches returns a slice of patches per file.
	FilePatches() []FilePatch
	// Message returns an optional message that can be at the top of the
	// Patch representation.
	Message() string
}

// FilePatch represents the necessary steps to transform one file to another.
type FilePatch interface {
	// If the patch deletes a file, "to" will be nil.
	File() string
	// Chunks returns a slice of ordered changes to transform "from" File to
	// "to" File. If the file is a binary one, Chunks will be empty.
	Chunks() []Chunk
}

// Chunk represents a chunk for git diff.
// For exmaple:
//
// @@ -21 +21 @@ require (
// -       github.com/projects/protos/a v0.0.25
// +       github.com/projects/protos/a v0.0.5
type Chunk interface {
	// Content contains the portion of the file.
	Content() string
	// Type contains the Operation to do with this Chunk.
	Type() Operation
}
