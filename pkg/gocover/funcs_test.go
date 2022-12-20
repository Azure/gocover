package gocover

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
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

func TestInExclueds(t *testing.T) {
	t.Run("inExclueds", func(t *testing.T) {
		cache := make(excludeFileCache)
		logger := logrus.New()
		testSuites := []struct {
			input  string
			expect bool
		}{
			{input: "mock_client/dbclient.go", expect: true},
			{input: "/mock_client/dbclient.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/mock_client/interface.go", expect: true}, // first match
			{input: "github.com/Azure/gocover/pkg/mock_client/interface.go", expect: true}, // second match, hit cache
			{input: "github.com/Azure/gocover/pkg/mock_client/dbclient.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/mock_client.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/mock.go", expect: true},
			{input: "zz_generated.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/api/v1/zz_generated.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/api/v1/zz_generated.deepcopy.go", expect: true},
			{input: "gocover.pb.go", expect: true},
			{input: "github.com/Azure/gocover/protos/v1/gocover.pb.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/foo", expect: true},
			{input: "github.com/Azure/gocover/pkg/foo/foo.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/foo/a/foo.go", expect: true},
			{input: "github.com/Azure/gocover/pkg/client/dbclient.go", expect: false},
		}

		for _, testCase := range testSuites {
			acutal := inExclueds(cache, []string{"**/mock_*/**", "**/zz_generated*.go", "**/*.pb.go", "**/mock*.go", "github.com/Azure/gocover/pkg/foo/**"}, testCase.input, logger)
			// only care about linux os
			if runtime.GOOS == "linux" {
				if acutal != testCase.expect {
					t.Errorf("for input %s, expcet %t, but get %t", testCase.input, testCase.expect, acutal)
				}
			}
		}
	})
}

func TestFormatFilePath(t *testing.T) {
	t.Run("formatFilePath", func(t *testing.T) {
		testSuites := []struct {
			input  string
			expect string
		}{
			{input: "/home/user/go/src/Azure/gocover/pkg/foo/foo.go", expect: "github.com/Azure/gocover/pkg/foo/foo.go"},
			{input: "/src/pkg/foo/foo.go", expect: "github.com/Azure/gocover/src/pkg/foo/foo.go"},
		}

		for _, testCase := range testSuites {
			acutal := formatFilePath("/home/user/go/src/Azure/gocover", testCase.input, "github.com/Azure/gocover")
			// only care about linux os
			if runtime.GOOS == "linux" {
				if acutal != testCase.expect {
					t.Errorf("expcet %s, but get %s", testCase.expect, acutal)
				}
			}
		}
	})
}

func TestReBuildStatistics(t *testing.T) {
	t.Run("reBuildStatistics", func(t *testing.T) {
		s := &report.Statistics{
			CoverageProfile: []*report.CoverageProfile{
				{TotalLines: 50, CoveredLines: 30, TotalEffectiveLines: 40, TotalIgnoredLines: 10},
				{TotalLines: 50, CoveredLines: 15, TotalEffectiveLines: 50, TotalIgnoredLines: 0},
			},
		}
		cache := excludeFileCache{"github.com/Azure/gocover/pkg/foo/foo.go": true}
		reBuildStatistics(s, cache)

		expectCoveragePercent := calculateCoverage(30+15, 40+50)
		if s.TotalCoveragePercent != expectCoveragePercent {
			t.Errorf("expect coverage percent %f, but get %f", expectCoveragePercent, s.TotalCoveragePercent)
		}
		expectCoveragePercentWithoutIgnore := calculateCoverage(30+15, 50+50)
		if s.TotalCoverageWithoutIgnore != expectCoveragePercentWithoutIgnore {
			t.Errorf("expect coverage percent %f, but get %f", expectCoveragePercentWithoutIgnore, s.TotalCoverageWithoutIgnore)
		}
		expectTotal := 50 + 50
		if s.TotalLines != expectTotal {
			t.Errorf("expect total %d, but get %d", expectTotal, s.TotalLines)
		}
		expectTotalCover := 30 + 15
		if s.TotalCoveredLines != expectTotalCover {
			t.Errorf("expect total covered %d, but get %d", expectTotalCover, s.TotalCoveredLines)
		}
		expectTotalEffective := 40 + 50
		if s.TotalEffectiveLines != expectTotalEffective {
			t.Errorf("expect effectived %d, but get %d", expectTotalEffective, s.TotalEffectiveLines)
		}
		expectTotalIgnore := 10 + 0
		if s.TotalIgnoredLines != expectTotalIgnore {
			t.Errorf("expect ignored %d, but get %d", expectTotalIgnore, s.TotalIgnoredLines)
		}

		if len(cache) != len(s.ExcludeFiles) {
			t.Errorf("should have %d exclude file, but get %d", len(cache), len(s.ExcludeFiles))
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
	storeCoverageDataFn              func(ctx context.Context, data *dbclient.CoverageData) error
	storeIgnoreProfileDataFn         func(ctx context.Context, data *dbclient.IgnoreProfileData) error
	storeCoverageDataFromFileFn      func(ctx context.Context, data []*dbclient.CoverageData) error
	storeIgnoreProfileDataFromFileFn func(ctx context.Context, data []*dbclient.IgnoreProfileData) error
}

func (client *mockDbClient) StoreCoverageData(context context.Context, data *dbclient.CoverageData) error {
	return client.storeCoverageDataFn(context, data)
}

func (client *mockDbClient) StoreIgnoreProfileData(context context.Context, data *dbclient.IgnoreProfileData) error {
	return client.storeIgnoreProfileDataFn(context, data)
}

func (client *mockDbClient) StoreCoverageDataFromFile(ctx context.Context, data []*dbclient.CoverageData) error {
	return client.storeCoverageDataFromFileFn(ctx, data)
}

func (client *mockDbClient) StoreIgnoreProfileDataFromFile(ctx context.Context, data []*dbclient.IgnoreProfileData) error {
	return client.storeIgnoreProfileDataFromFileFn(ctx, data)
}

func TestStore(t *testing.T) {
	t.Run("store successfully", func(t *testing.T) {
		client := &mockDbClient{
			storeCoverageDataFromFileFn: func(ctx context.Context, data []*dbclient.CoverageData) error {
				return nil
			},
		}

		all := []*report.AllInformation{
			{TotalLines: 120, TotalEffectiveLines: 100, TotalIgnoredLines: 20, TotalCoveredLines: 80},
		}

		err := storeCoverageData(context.Background(), client, all, FullCoverage, "")
		if err != nil {
			t.Errorf("should return nil, but get error: %s", err)
		}
	})

	t.Run("store failed", func(t *testing.T) {
		client := &mockDbClient{
			storeCoverageDataFromFileFn: func(ctx context.Context, data []*dbclient.CoverageData) error {
				return errors.New("unexpected error")
			},
		}

		all := []*report.AllInformation{
			{TotalLines: 120, TotalEffectiveLines: 100, TotalIgnoredLines: 20, TotalCoveredLines: 80},
		}

		err := storeCoverageData(context.Background(), client, all, FullCoverage, "")
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
