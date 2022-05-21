package gittool

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestNewGitClient(t *testing.T) {
	t.Run("should new git client fail", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		_, err := NewGitClient(path)
		if err == nil {
			t.Error("should fail")
		}
	})

	t.Run("should new git client successfully", func(t *testing.T) {
		path, _, clean := temporalRepository("")
		defer clean()

		client, err := NewGitClient(path)
		if err != nil {
			t.Errorf("new git client: %s", err)
		}
		if client == nil {
			t.Error("should get git client")
		}
	})
}

func TestDiffChanges(t *testing.T) {
	t.Run("get diff changes between HEAD and specified branch", func(t *testing.T) {
		branch := "foo"
		_, repo, clean := temporalRepository(branch)
		defer clean()

		client := &gitClient{repository: repo}
		_, err := client.diffChanges(branch)
		if err != nil {
			t.Errorf("diff change: %s", err)
		}
	})
}

func TestBuildChangeFromFile(t *testing.T) {
	t.Run("when read file fail", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		g := &gitClient{repositoryPath: path}

		_, err := g.buildChangeFromFile(path)
		if err == nil {
			t.Error("should return error")
		}
	})

	t.Run("when read file successfully", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		filename := "foo"
		texts := `foo
bar
hello world
`
		f := filepath.Join(path, filename)
		err := ioutil.WriteFile(f, []byte(texts), 0644)
		checkError(err)

		g := &gitClient{repositoryPath: path}

		change, err := g.buildChangeFromFile(filename)
		if err != nil {
			t.Errorf("build change from file: %s", err)
		}
		if change.FileName != filename {
			t.Errorf("filename should be %s, but get %s", filename, change.FileName)
		}
		if change.Mode != NewMode {
			t.Errorf("change should be new mode (%d), but get %d", NewMode, change.Mode)
		}
		if len(change.Sections) != 1 {
			t.Errorf("change should contain 1 section, but get %d", len(change.Sections))
		}
		section := change.Sections[0]
		if section.Count != 3 {
			t.Errorf("should have 3 lines, but get %d", section.Count)
		}
		if section.Operation != Add {
			t.Errorf("should be Add(%d) operation, but get %d", Add, section.Operation)
		}
		if section.StartLine != 1 {
			t.Errorf("new mode change start line should start from 1, but get %d", section.StartLine)
		}
		if section.Count != section.EndLine {
			t.Errorf("new mode change end line should be equal with total line count %d, but get %d", section.Count, section.EndLine)
		}
		if section.Count != len(section.Contents) {
			t.Errorf("new mode change contents should have %d lines, but get %d lines", section.Count, len(section.Contents))
		}
		if section.Contents[0] != "foo" {
			t.Errorf("first item should be 'foo', but get: %s", section.Contents[0])
		}
		if section.Contents[1] != "bar" {
			t.Errorf("first item should be 'bar', but get: %s", section.Contents[1])
		}
		if section.Contents[2] != "hello world" {
			t.Errorf("first item should be 'hello world', but get: %s", section.Contents[2])
		}
	})
}

func TestBuildChangeFromChunks(t *testing.T) {
	t.Run("build change from chunks", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		g := &gitClient{repositoryPath: path}

		equalChunk := &mockChunk{
			ContentFn: func() string {
				return `line1
line2`
			},
			TypeFn: func() diff.Operation {
				return diff.Equal
			},
		}
		addChunk := &mockChunk{
			ContentFn: func() string {
				return `line3
line4`
			},
			TypeFn: func() diff.Operation {
				return diff.Add
			},
		}
		deleteChunk := &mockChunk{
			ContentFn: func() string {
				return `line5
line6`
			},
			TypeFn: func() diff.Operation {
				return diff.Delete
			},
		}

		filename := "foo"
		change, err := g.buildChangeFromChunks(filename, []diff.Chunk{equalChunk, addChunk, deleteChunk})
		if err != nil {
			t.Errorf("build change from chunks: %s", err)
		}
		if change.FileName != filename {
			t.Errorf("filename should be %s, but get %s", filename, change.FileName)
		}
		if change.Mode != ModifyMode {
			t.Errorf("change should be new mode (%d), but get %d", ModifyMode, change.Mode)
		}

		if len(change.Sections) != 1 {
			t.Errorf("change should contain 1 section, but get %d", len(change.Sections))
		}
		section := change.Sections[0]
		if section.Count != 2 {
			t.Errorf("should have 2 lines, but get %d", section.Count)
		}
		if section.Operation != Add {
			t.Errorf("should be Add(%d) operation, but get %d", Add, section.Operation)
		}
		if section.StartLine != 3 {
			t.Errorf("modify mode change start line should start from 3, but get %d", section.StartLine)
		}
		if section.EndLine != 4 {
			t.Errorf("modify mode change end line should be 4, but get %d", section.EndLine)
		}
		if section.Count != len(section.Contents) {
			t.Errorf("modify mode change contents should have %d lines, but get %d lines", section.Count, len(section.Contents))
		}
		if section.Contents[0] != "line3" {
			t.Errorf("first item should be 'line3', but get: %s", section.Contents[0])
		}
		if section.Contents[1] != "line4" {
			t.Errorf("first item should be 'line4', but get: %s", section.Contents[1])
		}
	})
}

func TestBuildChangeFromPatch(t *testing.T) {
	t.Run("both from and to files are nil", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		g := &gitClient{repositoryPath: path}

		change, err := g.buildChangeFromPatch(&mockFilePatch{
			FilesFn: func() (from diff.File, to diff.File) {
				return nil, nil
			},
		})

		if err != nil {
			t.Error("should return nil error")
		}
		if change != nil {
			t.Error("should return nil as change")
		}
	})

	t.Run("both from and to files are not nil", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		equalChunk := &mockChunk{
			ContentFn: func() string {
				return `line1
line2`
			},
			TypeFn: func() diff.Operation {
				return diff.Equal
			},
		}
		addChunk := &mockChunk{
			ContentFn: func() string {
				return `line3
line4`
			},
			TypeFn: func() diff.Operation {
				return diff.Add
			},
		}
		deleteChunk := &mockChunk{
			ContentFn: func() string {
				return `line5
line6`
			},
			TypeFn: func() diff.Operation {
				return diff.Delete
			},
		}

		filename := "foo"
		g := &gitClient{repositoryPath: path}

		change, err := g.buildChangeFromPatch(&mockFilePatch{
			FilesFn: func() (from diff.File, to diff.File) {
				return &mockFile{
						PathFn: func() string {
							return filename
						},
					}, &mockFile{
						PathFn: func() string {
							return filename
						},
					}
			},
			ChunksFn: func() []diff.Chunk {
				return []diff.Chunk{
					equalChunk, addChunk, deleteChunk,
				}
			},
		})

		if err != nil {
			t.Errorf("build change from patch: %s", err)
		}
		if change.FileName != filename {
			t.Errorf("filename should be %s, but get %s", filename, change.FileName)
		}
		if change.Mode != ModifyMode {
			t.Errorf("change should be new mode (%d), but get %d", ModifyMode, change.Mode)
		}

	})

	t.Run("only from file is nil", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		filename := "foo"
		texts := `foo
bar
hello world
`
		f := filepath.Join(path, filename)
		err := ioutil.WriteFile(f, []byte(texts), 0644)
		checkError(err)

		g := &gitClient{repositoryPath: path}

		change, err := g.buildChangeFromPatch(&mockFilePatch{
			FilesFn: func() (from diff.File, to diff.File) {
				return nil, &mockFile{
					PathFn: func() string {
						return filename
					},
				}
			},
		})

		if err != nil {
			t.Errorf("build change from file: %s", err)
		}
		if change.FileName != filename {
			t.Errorf("filename should be %s, but get %s", filename, change.FileName)
		}
		if change.Mode != NewMode {
			t.Errorf("change should be new mode (%d), but get %d", NewMode, change.Mode)
		}
	})

	t.Run("only to file is nil", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		g := &gitClient{repositoryPath: path}

		change, err := g.buildChangeFromPatch(&mockFilePatch{
			FilesFn: func() (from diff.File, to diff.File) {
				return &mockFile{}, nil
			},
		})

		if err != nil {
			t.Error("should return nil error")
		}
		if change != nil {
			t.Error("should return nil as change")
		}
	})
}

func TestDiffChangesFromCommitted(t *testing.T) {
	t.Run("execute diffChanges fail", func(t *testing.T) {
		path, repo, clean := temporalRepository("")
		defer clean()

		g := &gitClient{repositoryPath: path, repository: repo}
		_, err := g.DiffChangesFromCommitted("foo")
		if err == nil {
			t.Error("should return error")
		}
	})

	t.Run("execute diffChanges success", func(t *testing.T) {
		newBranch := "foo"
		path, repo, clean := temporalRepository(newBranch)
		defer clean()

		g := &gitClient{repositoryPath: path, repository: repo}
		_, err := g.DiffChangesFromCommitted("foo")
		if err != nil {
			t.Errorf("should not return error, but get: %s", err)
		}
	})
}

// temporalDir creates a temp directory for testing.
func temporalDir() (path string, clean func()) {
	tmpDir, err := ioutil.TempDir("", "gocover")
	checkError(err)

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

// temporalDir creates a temp git repository for testing.
func temporalRepository(newBranch string) (string, *gogit.Repository, func()) {
	// temp directory
	tmpDir, err := ioutil.TempDir("", "gocover")
	checkError(err)

	// init repository in temp directory
	repo, err := gogit.PlainInit(tmpDir, false)
	checkError(err)

	// first init commit
	worktree, err := repo.Worktree()
	checkError(err)

	filename := filepath.Join(tmpDir, "example-git-file")
	err = ioutil.WriteFile(filename, []byte("hello world!"), 0644)
	checkError(err)

	_, err = worktree.Add("example-git-file")
	checkError(err)

	_, err = worktree.Commit("init commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "foo",
			Email: "foo@bar.org",
			When:  time.Now(),
		},
	})
	checkError(err)

	// create new branch and checkout if needed
	if newBranch != "" {
		err = worktree.Checkout(&gogit.CheckoutOptions{
			Branch: plumbing.ReferenceName(newBranch),
			Create: true,
		})
		checkError(err)
	}

	return tmpDir, repo, func() {
		os.RemoveAll(tmpDir)
	}
}

// checkError checks the error and panic error at preparing testing environment steps.
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

// mock the interface for github.com/go-git/go-git/v5/plumbing/format/diff

type mockFilePatch struct {
	IsBinaryFn func() bool
	FilesFn    func() (from, to diff.File)
	ChunksFn   func() []diff.Chunk
}

type mockFile struct {
	HashFn func() plumbing.Hash
	ModeFn func() filemode.FileMode
	PathFn func() string
}

type mockChunk struct {
	ContentFn func() string
	TypeFn    func() diff.Operation
}

func (patch *mockFilePatch) IsBinary() bool {
	return patch.IsBinaryFn()
}

func (patch *mockFilePatch) Files() (from, to diff.File) {
	return patch.FilesFn()
}

func (patch *mockFilePatch) Chunks() []diff.Chunk {
	return patch.ChunksFn()
}

func (file *mockFile) Hash() plumbing.Hash {
	return file.HashFn()
}

func (file *mockFile) Mode() filemode.FileMode {
	return file.ModeFn()
}

func (file *mockFile) Path() string {
	return file.PathFn()
}

func (chunk *mockChunk) Content() string {
	return chunk.ContentFn()
}

func (chunk *mockChunk) Type() diff.Operation {
	return chunk.TypeFn()
}
