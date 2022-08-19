package gtest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Azure/gocover/pkg/gittool"
)

const testFileSuffix = "_test.go"

type GocoverTest struct {
	RepositoryPath string
	CompareBranch  string
	GitClient      gittool.GitClient
	Writer         io.Writer
}

func NewGocoverTest(
	repositoryPath string,
	compareBranch string,
	writer io.Writer,
) (*GocoverTest, error) {

	gitClient, err := gittool.NewGitClient(repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("git repository: %w", err)
	}

	return &GocoverTest{
		RepositoryPath: repositoryPath,
		CompareBranch:  compareBranch,
		GitClient:      gitClient,
		Writer:         writer,
	}, nil
}

var goMockFileRegexp = regexp.MustCompile(`.*mock_.*`)

// EnsureGoTestFiles checks the diff changes, and ensures the existence of go test file
// if there are changes about go source files.
func (g *GocoverTest) EnsureGoTestFiles() error {
	changes, err := g.GitClient.DiffChangesFromCommitted(g.CompareBranch)
	if err != nil {
		return fmt.Errorf("git diff: %w", err)
	}

	handled := make(map[string]bool)
	for _, c := range changes {
		// in case of other files, except go files
		if !strings.HasSuffix(c.FileName, ".go") {
			fmt.Fprintf(g.Writer, "skip checking for other file: %s\n", c.FileName)
			continue
		}

		// don't generate test file for go mock files
		if goMockFileRegexp.MatchString(c.FileName) {
			fmt.Fprintf(g.Writer, "skip checking for mock file: %s\n", c.FileName)
			continue
		}

		f := filepath.Join(g.RepositoryPath, c.FileName)

		folder := filepath.Dir(f)
		if _, ok := handled[folder]; ok {
			continue
		}
		handled[folder] = true

		fmt.Fprintf(g.Writer, "checking test files for package: %s\n", folder)

		exist, err := checkTestFileExistence(folder)
		if err != nil {
			return fmt.Errorf("check test file existence: %w", err)
		}
		if exist {
			continue
		}

		pkgName, err := parsePackageName(f)
		if err != nil {
			return fmt.Errorf("parse package name from %s: %w", f, err)
		}

		testFile := filepath.Join(folder, testFileName(pkgName))
		ioutil.WriteFile(
			testFile,
			testFileContents(pkgName),
			0644,
		)
		fmt.Fprintf(g.Writer, "no test files in package %s, create test file for it: %s\n", folder, testFile)
	}

	return nil
}

func testFileName(pkgName string) string {
	return fmt.Sprintf("%s_test.go", pkgName)
}

func testFileContents(pkgName string) []byte {
	return []byte(fmt.Sprintf("package %s", pkgName))
}

func checkTestFileExistence(folder string) (bool, error) {
	files, err := ioutil.ReadDir(folder)
	if err != nil {
		return false, fmt.Errorf("%w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(f.Name()), testFileSuffix) {
			return true, nil
		}
	}

	return false, nil
}

var (
	ErrPackageNameNotFound = errors.New("package not found")
	packageRegexp          = regexp.MustCompile(`^package\s+([a-zA-Z][a-zA-Z0-9]*)`) // regexp for matching "package xxx"
)

func parsePackageName(filename string) (string, error) {
	rd, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer rd.Close()

	s := bufio.NewScanner(rd)
	for s.Scan() {
		match := packageRegexp.FindStringSubmatch(s.Text())
		if match == nil {
			continue
		}

		return match[1], nil
	}

	return "", ErrPackageNameNotFound
}
