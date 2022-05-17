package gittool

// NewGitClient creates a git client instance for git diff.
// TODO: implement it
func NewGitClient() GitClient {
	return nil
}

// TODO: implement GitClient interface
type GitClient interface {
	// DiffCommitted returns the git patch for git diff command between current and compared branch.
	DiffCommitted(compareBranch string) (Patch, error)
}
