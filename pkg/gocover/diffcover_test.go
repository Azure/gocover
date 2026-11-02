package gocover

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestDiffCoverNoApplicableChanges(t *testing.T) {
	for _, change := range []string{"config.yaml", "sample_test.go", "delete sample.go", "none"} {
		t.Run(change, func(t *testing.T) {
			repoPath, repo, base := newDiffCoverTestRepository(t)
			switch change {
			case "delete sample.go":
				if err := os.Remove(filepath.Join(repoPath, "sample.go")); err != nil {
					t.Fatal(err)
				}
			case "none":
			default:
				if err := os.WriteFile(filepath.Join(repoPath, change), []byte("changed\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if change != "none" {
				commitDiffCoverTestChanges(t, repo)
			}
			profilePath := filepath.Join(t.TempDir(), "coverage.out")
			// Resolving this package would fail: empty diffs must not resolve it.
			if err := os.WriteFile(profilePath,
				[]byte("mode: set\nexample.com/unresolvable/missing.go:1.1,1.2 1 0\n"), 0644); err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&output)
			outputDir := t.TempDir()
			generator, err := NewDiffCover(&DiffOption{
				RepositoryPath:   repoPath,
				CompareBranch:    base,
				CoverProfiles:    []string{profilePath},
				CoverageBaseline: 80,
				OutputDir:        outputDir,
				ReportName:       "empty",
				DbOption:         &dbclient.DBOption{},
				Logger:           logger,
			})
			if err != nil {
				t.Fatal(err)
			}
			assert.NoError(t, generator.Run(context.Background()))
			html, err := os.ReadFile(filepath.Join(outputDir, "empty.html"))
			assert.NoError(t, err)
			assert.Contains(t, string(html), "<html")
			assert.Contains(t, output.String(), "No applicable changed Go code")
			statistics := generator.(*diffCover).coverageTree.Statistics()
			assert.Zero(t, statistics.TotalEffectiveLines)
			assert.Zero(t, statistics.TotalLines)
		})
	}
}

func TestDiffCoverNoChangeErrors(t *testing.T) {
	tests := []struct {
		name     string
		profile  string
		missing  bool
		baseline string
	}{
		{name: "missing profile", missing: true},
		{name: "malformed profile", profile: "invalid\n"},
		{name: "invalid baseline", profile: "mode: set\n", baseline: "missing-branch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repoPath, _, base := newDiffCoverTestRepository(t)
			if test.baseline != "" {
				base = test.baseline
			}
			profilePath := filepath.Join(t.TempDir(), "coverage.out")
			if !test.missing {
				if err := os.WriteFile(profilePath, []byte(test.profile), 0644); err != nil {
					t.Fatal(err)
				}
			}
			outputDir := t.TempDir()
			generator, err := NewDiffCover(&DiffOption{
				RepositoryPath: repoPath,
				CompareBranch:  base,
				CoverProfiles:  []string{profilePath},
				OutputDir:      outputDir,
				ReportName:     "empty",
				DbOption:       &dbclient.DBOption{},
				Logger:         logrus.New(),
			})
			if err != nil {
				t.Fatal(err)
			}
			assert.Error(t, generator.Run(context.Background()))
			_, err = os.Stat(filepath.Join(outputDir, "empty.html"))
			assert.True(t, os.IsNotExist(err))
		})
	}
}

func TestDiffCoverChangedCodeAndFullMode(t *testing.T) {
	repoPath, repo, base := newDiffCoverTestRepository(t)
	source := "package sample\nfunc Value() int {\n\treturn 2\n}\n"
	if err := os.WriteFile(filepath.Join(repoPath, "sample.go"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	commitDiffCoverTestChanges(t, repo)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoPath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Error(err)
		}
	})
	for _, count := range []string{"0", "1"} {
		t.Run("execution count "+count, func(t *testing.T) {
			profilePath := filepath.Join(t.TempDir(), "coverage.out")
			if err := os.WriteFile(profilePath,
				[]byte("mode: set\nexample.com/emptydiff/sample.go:3.2,3.10 1 "+count+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			logger := logrus.New()
			diff := &diffCover{
				repositoryPath:   repoPath,
				comparedBranch:   base,
				modulePath:       "example.com/emptydiff",
				coverFilenames:   []string{profilePath},
				excludeFiles:     make(excludeFileCache),
				coverageTree:     report.NewCoverageTree("example.com/emptydiff"),
				coverageBaseline: 80,
				logger:           logger,
			}
			statistics, err := diff.generateStatistics()
			if err != nil {
				t.Fatal(err)
			}
			assert.NotZero(t, statistics.TotalEffectiveLines)
			if count == "0" {
				assert.Error(t, diff.pass(statistics))
			} else {
				assert.NoError(t, diff.pass(statistics))
			}
			full := &fullCover{
				modulePath:     "example.com/emptydiff",
				coverFilenames: []string{profilePath},
				excludeFiles:   make(excludeFileCache),
				coverageTree:   report.NewCoverageTree("example.com/emptydiff"),
				logger:         logger,
			}
			fullStatistics, err := full.generateStatistics()
			if err != nil {
				t.Fatal(err)
			}
			assert.NotZero(t, fullStatistics.TotalEffectiveLines)
		})
	}
}

func newDiffCoverTestRepository(t *testing.T) (string, *gogit.Repository, string) {
	t.Helper()
	path := t.TempDir()
	repo, err := gogit.PlainInit(path, false)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"go.mod":    "module example.com/emptydiff\n",
		"sample.go": "package sample\nfunc Value() int {\n\treturn 1\n}\n",
	} {
		if err := os.WriteFile(filepath.Join(path, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	base := commitDiffCoverTestChanges(t, repo)
	return path, repo, base
}

func commitDiffCoverTestChanges(t *testing.T, repo *gogit.Repository) string {
	t.Helper()
	worktree, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if err := worktree.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		t.Fatal(err)
	}
	hash, err := worktree.Commit("test change", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Test", Email: "test@example.com", When: time.Now()},
	})
	if err != nil {
		t.Fatal(err)
	}
	return hash.String()
}
