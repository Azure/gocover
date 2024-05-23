package gittool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	gogitobj "github.com/go-git/go-git/v5/plumbing/object"
)

// NewGitClient creates a git client instance for git diff.
func NewGitClient(
	repositoryPath string,
) (GitClient, error) {
	repository, err := gogit.PlainOpen(repositoryPath)
	if err != nil {
		return nil, err
	}
	return &gitClient{
		repository:     repository,
		repositoryPath: repositoryPath,
	}, nil
}

type GitClient interface {
	// DiffChangesFromCommitted returns the diff changes between HEAD and compared branch commit.
	DiffChangesFromCommitted(compareBranch string) ([]*Change, error)
}

type gitClient struct {
	repository     *gogit.Repository
	repositoryPath string
}

var _ GitClient = (*gitClient)(nil)

func (g *gitClient) DiffChangesFromCommitted(compareBranch string) ([]*Change, error) {
	changes, err := g.diffChanges(compareBranch)
	if err != nil {
		return nil, fmt.Errorf("execute diff: %w", err)
	}

	var diffChanges []*Change
	for _, change := range changes {
		patch, err := change.Patch()
		if err != nil {
			return nil, fmt.Errorf("get patch: %w", err)
		}
		filePatches := patch.FilePatches()
		if len(filePatches) < 1 {
			return nil, errors.New("no patch found")
		}

		diffChange, err := g.buildChangeFromPatch(filePatches[0])
		if err != nil {
			return nil, fmt.Errorf("build change from patch: %w", err)
		}

		// filter nil change because buildChangeFromPatch should return nil as result
		if diffChange != nil {
			diffChanges = append(diffChanges, diffChange)
		}
	}

	return diffChanges, nil
}

// diffChanges get the diff changes between compared branch and HEAD commit.
// It equals to executing command `git diff {comparedBranch}...HEAD`.
//
// It uses package github.com/go-git/go-git to get such output.
func (g *gitClient) diffChanges(comparedBranch string) (gogitobj.Changes, error) {
	// get commit object of HEAD
	head, err := g.repository.Head()
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get HEAD %w", err)
	}

	headCommit, err := g.repository.CommitObject(head.Hash())
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get HEAD commit %w", err)
	}
	headTree, err := headCommit.Tree()
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get HEAD tree object %w", err)
	}

	// get commit object of compared branch
	comparedHash, err := g.repository.ResolveRevision(plumbing.Revision(comparedBranch))
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get %s %w", comparedBranch, err)
	}

	comparedCommit, err := g.repository.CommitObject(*comparedHash)
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get %s commit %w", comparedBranch, err)
	}
	comparedTree, err := comparedCommit.Tree()
	if err != nil {
		return gogitobj.Changes{}, fmt.Errorf("get %s tree object %w", comparedBranch, err)
	}

	return gogitobj.DiffTree(comparedTree, headTree)
}

// buildChangeFromPatch builds the diff change from file patch.
// It's the entrance for building diff change and use different strategy according to different context.
func (g *gitClient) buildChangeFromPatch(filePatch diff.FilePatch) (*Change, error) {

	from, to := filePatch.Files()
	if from == nil && to == nil {
		return nil, nil
	}

	switch {
	// modify or rename file
	case from != nil && to != nil:
		if isGoFile(to) {
			return g.buildChangeFromChunks(to.Path(), filePatch.Chunks())
		}

	// new file
	case from == nil:
		if isGoFile(to) {
			return g.buildChangeFromFile(to.Path())
		}

	// delete file
	case to == nil:
		// we don't care about delete files, omit it
	}

	return nil, nil

}

func isGoFile(fileInfo diff.File) bool {
	return fileInfo.Mode() == filemode.Regular &&
		strings.HasSuffix(fileInfo.Path(), ".go") &&
		!strings.HasSuffix(fileInfo.Path(), "_test.go")
}

// buildChangeFromChunks builds the diff change from git chunks.
// It's used when modify the existing file. and only contains the added chunks which will be used for later diff coverage.
// Input chunks are sorted in sequence and guaranteed by the calling library github.com/go-git/go-git.
func (g *gitClient) buildChangeFromChunks(filename string, chunks []diff.Chunk) (*Change, error) {

	// count the total lines of the file
	// equals lines + added lines should be equal with total lines.
	totalCount := 0
	var sections []*Section

	for _, chunk := range chunks {

		switch chunk.Type() {
		case diff.Equal:
			scanner := bufio.NewScanner(bytes.NewBufferString(chunk.Content()))
			for scanner.Scan() {
				totalCount++
			}

		case diff.Add:
			count := 0
			startLine := totalCount + 1

			scanner := bufio.NewScanner(bytes.NewBufferString(chunk.Content()))
			var contents []string
			for scanner.Scan() {
				count++
				totalCount++
				contents = append(contents, scanner.Text())
			}

			endLine := startLine + count - 1

			sections = append(sections, &Section{
				StartLine: startLine,
				EndLine:   endLine,
				Count:     count,
				Contents:  contents,
				Operation: Add,
			})

		case diff.Delete:
			// we omit delete chunks
		}
	}

	return &Change{
		FileName: filename,
		Sections: sections,
		Mode:     ModifyMode,
	}, nil
}

// buildChangeFromFile builds the diff change from file,
// It's used when add new file compared to specified branch.
// So, it contains all the lines from the file as its contents,
// and be assigned NewMode to indicates that it's the new created file.
func (g *gitClient) buildChangeFromFile(filename string) (*Change, error) {

	fullFilePath := filepath.Join(g.repositoryPath, filename)
	data, err := os.ReadFile(fullFilePath)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	count := 0

	var contents []string

	for scanner.Scan() {
		count++
		contents = append(contents, scanner.Text())
	}

	section := &Section{
		Count:     count,
		StartLine: 1,
		EndLine:   count,
		Contents:  contents,
		Operation: Add,
	}

	return &Change{
		FileName: filename,
		Mode:     NewMode,
		Sections: []*Section{section},
	}, nil
}
