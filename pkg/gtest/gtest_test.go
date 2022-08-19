package gtest

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/gocover/pkg/gittool"
)

func TestEnsureGoTestFiles(t *testing.T) {
	t.Run("diff changes failed", func(t *testing.T) {
		dir := t.TempDir()
		gocoverTest := &GocoverTest{
			RepositoryPath: dir,
			CompareBranch:  "origin/master",
			Writer:         &bytes.Buffer{},
			GitClient: &mockGitClient{
				DiffChangesFromCommittedFn: func(compareBranch string) ([]*gittool.Change, error) {
					return nil, errors.New("get changes error")
				},
			},
		}

		err := gocoverTest.EnsureGoTestFiles()
		if err == nil {
			t.Errorf("should fail, but get nil")
		}
	})

	t.Run("check go test files", func(t *testing.T) {
		dir := t.TempDir()
		os.Mkdir(filepath.Join(dir, "foo"), 0777)
		ioutil.WriteFile(filepath.Join(dir, "foo/foo.go"), []byte("package foo"), 0644)
		ioutil.WriteFile(filepath.Join(dir, "foo/foo_test.go"), []byte("package foo"), 0644)
		ioutil.WriteFile(filepath.Join(dir, "mock_foo/foo.go"), []byte("package mock_foo"), 0644)

		os.Mkdir(filepath.Join(dir, "bar"), 0777)
		ioutil.WriteFile(filepath.Join(dir, "bar/bar.go"), []byte("package zoo"), 0644)

		gocoverTest := &GocoverTest{
			RepositoryPath: dir,
			CompareBranch:  "origin/master",
			Writer:         &bytes.Buffer{},
			GitClient: &mockGitClient{
				DiffChangesFromCommittedFn: func(compareBranch string) ([]*gittool.Change, error) {
					return []*gittool.Change{
						{FileName: "foo/foo.go"},
						{FileName: "bar/bar.go"},
					}, nil
				},
			},
		}

		err := gocoverTest.EnsureGoTestFiles()
		if err != nil {
			t.Errorf("should no error, but get error %s", err)
		}

		newTestFile := filepath.Join(dir, "bar/zoo_test.go")
		contents, err := ioutil.ReadFile(newTestFile)
		if err != nil {
			t.Errorf("should no error, but get error %s", err)
		}
		if string(contents) != "package zoo" {
			t.Errorf("package contents should 'package bar', but get '%s'", string(contents))
		}
	})
}

type mockGitClient struct {
	DiffChangesFromCommittedFn func(compareBranch string) ([]*gittool.Change, error)
}

func (gitClient *mockGitClient) DiffChangesFromCommitted(compareBranch string) ([]*gittool.Change, error) {
	return gitClient.DiffChangesFromCommittedFn(compareBranch)
}

func TestCheckTestFileExistence(t *testing.T) {
	t.Run("no test files", func(t *testing.T) {
		dir := t.TempDir()
		ioutil.WriteFile(filepath.Join(dir, "foo.go"), []byte(""), 0644)

		exist, err := checkTestFileExistence(dir)
		if err != nil {
			t.Errorf("should not err, but get: %s", err)
		}
		if exist == true {
			t.Errorf("should no test files, but exist")
		}
	})

	t.Run("has test files", func(t *testing.T) {
		dir := t.TempDir()

		os.Mkdir(filepath.Join(dir, "gocover"), 0777)
		ioutil.WriteFile(filepath.Join(dir, "foo.go"), []byte(""), 0644)
		ioutil.WriteFile(filepath.Join(dir, "foo_test.go"), []byte(""), 0644)

		exist, err := checkTestFileExistence(dir)
		if err != nil {
			t.Errorf("should not err, but get: %s", err)
		}
		if exist == false {
			t.Errorf("should have test files, but does not")
		}
	})
}

func TestTestfileName(t *testing.T) {
	t.Run("testFileName", func(t *testing.T) {
		testSuites := []struct {
			input  string
			expect string
		}{
			{input: "foo", expect: "foo_test.go"},
			{input: "bar", expect: "bar_test.go"},
			{input: "zoo", expect: "zoo_test.go"},
		}

		for _, testCase := range testSuites {
			actual := testFileName(testCase.input)
			if actual != testCase.expect {
				t.Errorf("expect %s, but get %s", testCase.expect, actual)
			}
		}
	})
}

func TestTestFileContents(t *testing.T) {
	t.Run("testFileContents", func(t *testing.T) {
		testSuites := []struct {
			input  string
			expect string
		}{
			{input: "foo", expect: "package foo"},
			{input: "bar", expect: "package bar"},
			{input: "zoo", expect: "package zoo"},
		}

		for _, testCase := range testSuites {
			actual := testFileContents(testCase.input)
			if string(actual) != testCase.expect {
				t.Errorf("expect %s, but get %s", testCase.expect, string(actual))
			}
		}
	})
}

func TestParsePackageName(t *testing.T) {
	t.Run("package name found", func(t *testing.T) {
		dir := t.TempDir()

		fileContents := `// package zoo ...
package zoo

func foo() int { return 1 }
	`
		f := filepath.Join(dir, "foo.go")
		ioutil.WriteFile(f, []byte(fileContents), 0644)

		pkgName, err := parsePackageName(f)
		if err != nil {
			t.Errorf("should not error, but get %s", err)
		}
		if pkgName != "zoo" {
			t.Errorf("package name should be zoo, but get %s", pkgName)
		}
	})

	t.Run("no package name found", func(t *testing.T) {
		dir := t.TempDir()

		fileContents := `// package zoo ...
func foo() int { return 1 }
	`
		f := filepath.Join(dir, "foo.go")
		ioutil.WriteFile(f, []byte(fileContents), 0644)

		_, err := parsePackageName(f)
		if err == nil {
			t.Errorf("should return error, but get nil")
		}
	})
}

func TestPackageRegexp(t *testing.T) {
	t.Run("validate packageRegexp", func(t *testing.T) {
		testSuite := []struct {
			input  string
			expect []string
		}{
			{input: "package foo", expect: []string{"package foo", "foo"}},
			{input: "package foo", expect: []string{"package foo", "foo"}},
			{input: "package  foo", expect: []string{"package  foo", "foo"}},
			{input: "package foo1", expect: []string{"package foo1", "foo1"}},
			{input: "package Foo1", expect: []string{"package Foo1", "Foo1"}},
			{input: "package 1foo1", expect: nil},
			{input: " package foo", expect: nil},
		}

		for _, testCase := range testSuite {
			match := packageRegexp.FindStringSubmatch(testCase.input)
			if len(match) != len(testCase.expect) {
				t.Errorf("expect %d items, but get %d", len(testCase.expect), len(match))
			}
			n := len(match)
			for i := 0; i < n; i++ {
				if match[i] != testCase.expect[i] {
					t.Errorf("expect item %d %s, but %s", i, testCase.expect[i], match[i])
				}
			}
		}
	})
}

func TestMockFileRegexp(t *testing.T) {
	t.Run("validate mockFileRegexp", func(t *testing.T) {
		testSuite := []struct {
			input  string
			expect bool
		}{
			{input: "/mock_foo/foo.go", expect: true},
			{input: "/s/mock_interface/foo/foo.go", expect: true},
		}

		for _, testCase := range testSuite {
			match := goMockFileRegexp.MatchString(testCase.input)
			if match != testCase.expect {
				t.Errorf("expect %t, but get %t for input %s", testCase.expect, match, testCase.input)
			}
		}
	})
}
