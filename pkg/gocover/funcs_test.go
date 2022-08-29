package gocover

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
)

func TestCalculateCoverage(t *testing.T) {
	t.Run("calculateCoverage", func(t *testing.T) {
		testSuites := []struct {
			covered    int64
			effectived int64
			expect     float64
		}{
			{covered: 0, effectived: 0, expect: 100.0},
			{covered: 0, effectived: 100, expect: 0},
			{covered: 10, effectived: 100, expect: 10.0},
			{covered: 50, effectived: 100, expect: 50.0},
			{covered: 30, effectived: 50, expect: float64(30) / float64(50) * 100},
		}

		for _, testCase := range testSuites {
			actual := calculateCoverage(testCase.covered, testCase.effectived)
			if actual != testCase.expect {
				t.Errorf("expect calculateCoverage(%d, %d) = %f, but get %f", testCase.covered, testCase.effectived, testCase.expect, actual)
			}
		}
	})
}

func TestFindFileContents(t *testing.T) {
	t.Run("findFileContents", func(t *testing.T) {
		dir := t.TempDir()
		filename := filepath.Join(dir, "foo.go")
		contents := "" +
			`package foo

func foo() {
}`
		err := ioutil.WriteFile(filename, []byte(contents), 0644)
		if err != nil {
			t.Errorf("prepare test environment failed: %s", err)
		}

		fileCache := make(fileContentsCache)
		result1, err := findFileContents(fileCache, filename)
		if err != nil {
			t.Errorf("should not return error, but get %s", err)
		}
		if strings.Join(result1, "\n") != contents {
			t.Errorf("expect %s, but get %s", contents, strings.Join(result1, "\n"))
		}

		result2, err := findFileContents(fileCache, filename)
		if err != nil {
			t.Errorf("should not return error, but get %s", err)
		}
		if strings.Join(result2, "\n") != contents {
			t.Errorf("expect %s, but get %s", contents, strings.Join(result2, "\n"))
		}
	})

	t.Run("findFileContents", func(t *testing.T) {
		dir := t.TempDir()
		filename := filepath.Join(dir, "foo.go")
		fileCache := make(fileContentsCache)
		_, err := findFileContents(fileCache, filename)
		if err == nil {
			t.Errorf("should return error, but return nil")
		}
	})
}

type mockDbClient struct {
	storeFn func(context context.Context, data *dbclient.Data) error
}

func (client *mockDbClient) Store(context context.Context, data *dbclient.Data) error {
	return client.storeFn(context, data)
}

func TestStore(t *testing.T) {
	t.Run("store successfully", func(t *testing.T) {
		client := &mockDbClient{
			storeFn: func(context context.Context, data *dbclient.Data) error {
				return nil
			},
		}

		all := []*report.AllInformation{
			{TotalLines: 120, TotalEffectiveLines: 100, TotalIgnoredLines: 20, TotalCoveredLines: 80},
		}

		err := store(context.Background(), client, all, FullCoverage, "")
		if err != nil {
			t.Errorf("should return nil, but get error: %s", err)
		}

		err = store(context.Background(), client, nil, FullCoverage, "")
		if err != nil {
			t.Errorf("should return nil, but get error: %s", err)
		}
	})

	t.Run("store failed", func(t *testing.T) {
		client := &mockDbClient{
			storeFn: func(context context.Context, data *dbclient.Data) error {
				return errors.New("unexpected error")
			},
		}

		all := []*report.AllInformation{
			{TotalLines: 120, TotalEffectiveLines: 100, TotalIgnoredLines: 20, TotalCoveredLines: 80},
		}

		err := store(context.Background(), client, all, FullCoverage, "")
		if err == nil {
			t.Errorf("should return error, but no error")
		}
	})
}

func TestParseGoModulePath(t *testing.T) {
	t.Run("parse go module path from go.mod", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "foo"), os.ModePerm)
		os.MkdirAll(filepath.Join(dir, "empty"), os.ModePerm)
		os.MkdirAll(filepath.Join(dir, "nonexist"), os.ModePerm)

		ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/Azure/gocover"), 0644)
		ioutil.WriteFile(filepath.Join(dir, "foo/go.mod"), []byte("module github.com/Azure/gocover/foo"), 0644)
		ioutil.WriteFile(filepath.Join(dir, "empty/go.mod"), []byte(""), 0644)

		var testSuites = []struct {
			input  string
			expect string
			err    error
		}{
			{input: filepath.Join(dir, "."), expect: "github.com/Azure/gocover", err: nil},
			{input: filepath.Join(dir, "foo"), expect: "github.com/Azure/gocover/foo", err: nil},
			{input: filepath.Join(dir, "empty"), expect: "", err: ErrModuleNotFound},
			{input: filepath.Join(dir, "nonexist"), expect: "", err: errors.New("file not exists")},
		}

		for _, testCase := range testSuites {
			actual, err := parseGoModulePath(testCase.input)
			if actual != testCase.expect {
				t.Errorf("should %s, but get %s", testCase.expect, actual)
			}
			if testCase.err == nil && testCase.err != err {
				t.Errorf("error should nil but get %s", err)
			}
			if testCase.err != nil && err == nil {
				t.Errorf("error should be %s, but get nil", testCase.err)
			}
		}
	})
}
