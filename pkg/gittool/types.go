package gittool

// DiffMode values represent the kind of things a Change can represent:
// creations, modifications, deletions or renaming of files.
type DiffMode int

// The set of possible diff mode in a change.
const (
	_ DiffMode = iota
	NewMode
	ModifyMode
	DeleteMode
	RenameMode
)

// DiffOperation defines the operation of a diff item for a git chunk.
type DiffOperation int

const (
	// Equal item represents a equals diff chunk.
	Equal DiffOperation = iota
	// Add item represents an insert diff chunk.
	Add
	// Delete item represents a delete diff chunk.
	Delete
	// There is no operation called Modify, since it can be fullfilled by delete a line and add a line.
)

// Section represents a portion of a file that contains the changes
// that made in the HEAD commit.
type Section struct {
	// Operation indicates how this section operates compared to the specified branch.
	// Equal means no change
	// Add means contents of the section are added into the compared branch.
	// Delete means contents of the section are deleted from the compared branch.
	Operation DiffOperation
	// Count indicates how many lines this section object contains in total
	Count int
	// StartLine indicates where this section starts from the source file.
	StartLine int
	// EndLine indicates where thie section ends from the source file.
	EndLine int
	// Contents contains [StartLine..EndLine] lines from the source file.
	Contents []string
}

// Change contains all the changes made to a specific file
// that made in the HEAD commit.
type Change struct {
	// FileName indicates which file the change are applied against.
	FileName string
	// Mode indicates what kind of the change, whether it's a new created file,
	// or a modified file, or deleted file, or renamed file.
	Mode DiffMode
	// Sections indicates the change details.
	// For NewMode and RenameMode it contains all the contents of the new file
	// For ModifyMode it contains the each change sections made to compared branch
	// For DeleteMode it's empty
	Sections []*Section
}
