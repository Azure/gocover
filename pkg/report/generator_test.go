package report

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func TestNewReportGenerator(t *testing.T) {
	t.Run("NewReportGenerator", func(t *testing.T) {
		NewReportGenerator(&Statistics{}, "colorful", "", "", &bytes.Buffer{})
	})
}

func TestGenerateReport(t *testing.T) {
	t.Run("no diff information", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		g := &htmlReportGenerator{
			lexer:      lexers.Get(CodeLanguage),
			style:      styles.Get("colorful"),
			outputPath: path,
			reportName: "corverage.html",
			statistics: &Statistics{
				ComparedBranch: "origin/master",
			},
			writer: &bytes.Buffer{},
		}

		err := g.GenerateReport()
		if err != nil {
			t.Errorf("should not error, but get: %s", err)
		}

		data, err := ioutil.ReadFile(filepath.Join(g.outputPath, finalName(g.reportName)))
		checkError(err)

		reportString := string(data)
		if !strings.Contains(reportString, "No lines with coverage information in this diff.") {
			t.Error("report should contains empty diff information")
		}
		if !strings.Contains(reportString, "origin/master") {
			t.Error("report should contains compared branch 'origin/master'")
		}
	})

	t.Run("have coverage profiles", func(t *testing.T) {
		path, clean := temporalDir()
		defer clean()

		statistics := &Statistics{
			ComparedBranch:       "origin/master",
			TotalLines:           8,
			TotalEffectiveLines:  6,
			TotalIgnoredLines:    2,
			TotalViolationLines:  2,
			TotalCoveragePercent: 70,
			CoverageProfile: []*CoverageProfile{
				{
					FileName:            "foo.txt",
					TotalLines:          20,
					TotalEffectiveLines: 20,
					TotalIgnoredLines:   0,
					CoveredLines:        20,
				},
				{
					FileName:            "bar.txt",
					CoveredLines:        8,
					TotalIgnoredLines:   2,
					TotalEffectiveLines: 10,
					TotalLines:          12,
					TotalViolationLines: []int{2, 10},
					ViolationSections: []*ViolationSection{
						{
							ViolationLines: []int{2},
							StartLine:      1,
							EndLine:        3,
							Contents:       []string{"foo", "bar", "zoo"},
						},
						{
							ViolationLines: []int{10},
							StartLine:      8,
							EndLine:        10,
							Contents:       []string{"text1", "text2", "text3"},
						},
					},
				},
			},
		}

		g := &htmlReportGenerator{
			lexer:      lexers.Get(CodeLanguage),
			style:      styles.Get("colorful"),
			outputPath: path,
			reportName: "corverage.html",
			statistics: statistics,
			writer:     &bytes.Buffer{},
		}

		err := g.GenerateReport()
		if err != nil {
			t.Errorf("should not error, but get: %s", err)
		}

		data, err := ioutil.ReadFile(filepath.Join(g.outputPath, finalName(g.reportName)))
		checkError(err)

		reportString := string(data)
		if !strings.Contains(string(data), "origin/master") {
			t.Error("report should contains compared branch 'origin/master'")
		}
		for _, v := range []string{"foo", "bar", "zoo", "text1", "text2", "text3", "foo.txt", "bar.txt"} {
			if !strings.Contains(reportString, v) {
				t.Errorf("report should contains %s", v)
			}
		}
	})

}

func TestProcessCodeSnippets(t *testing.T) {

	t.Run("processCodeSnippets", func(t *testing.T) {
		statistics := &Statistics{
			CoverageProfile: []*CoverageProfile{
				{
					FileName:     "foo.txt",
					CoveredLines: 20,
					TotalLines:   20,
				},
				{
					FileName:            "bar.txt",
					CoveredLines:        8,
					TotalLines:          10,
					TotalViolationLines: []int{2, 10},
					ViolationSections: []*ViolationSection{
						{
							ViolationLines: []int{2},
							StartLine:      1,
							EndLine:        3,
							Contents:       []string{"foo", "bar", "zoo"},
						},
						{
							ViolationLines: []int{10},
							StartLine:      8,
							EndLine:        10,
							Contents:       []string{"text1", "text2", "text3"},
						},
					},
				},
			},
		}

		g := &htmlReportGenerator{
			lexer:      lexers.Get(CodeLanguage),
			style:      styles.Get("colorful"),
			statistics: statistics,
		}

		for _, profile := range g.statistics.CoverageProfile {
			if len(profile.CodeSnippet) != 0 {
				t.Errorf("should be empty before run processCodeSnippets")
			}
		}

		err := g.processCodeSnippets()
		if err != nil {
			t.Errorf("should not error, but get: %s", err)
		}

		for _, profile := range g.statistics.CoverageProfile {
			if profile.FileName == "foo.txt" {
				if len(profile.CodeSnippet) != 0 {
					t.Error("should no code snippet because coverage percent is 100%")
				}
				continue
			}

			if profile.FileName == "bar.txt" {
				if len(profile.CodeSnippet) != len(profile.ViolationSections) {
					t.Errorf("should get %d code snippets, but get %d", len(profile.ViolationSections), len(profile.CodeSnippet))
				}
			}
		}

	})

}

func TestIntsJoin(t *testing.T) {
	t.Run("intsJoin", func(t *testing.T) {
		testsuites := []struct {
			expected string
			input    []int
		}{
			{input: []int{}, expected: ""},
			{input: []int{1}, expected: "1"},
			{input: []int{1, 2, 3}, expected: "1,2,3"},
		}

		for _, testcase := range testsuites {
			actual := intsJoin(testcase.input)
			if testcase.expected != actual {
				t.Errorf("expected %s, but get %s", testcase.expected, actual)
			}
		}
	})
}

func TestFinalName(t *testing.T) {
	t.Run("", func(t *testing.T) {
		testsuites := []struct {
			expected string
			input    string
		}{
			{input: "a", expected: "a.html"},
			{input: "b", expected: "b.html"},
			{input: "c", expected: "c.html"},
		}
		for _, testcase := range testsuites {
			actual := finalName(testcase.input)
			if testcase.expected != actual {
				t.Errorf("expected %s, but get %s", testcase.expected, actual)
			}
		}
	})
}

func TestNormalizeLines(t *testing.T) {
	t.Run("normalizeLines", func(t *testing.T) {
		testsuites := []struct {
			expected string
			input    int
		}{
			{input: 1, expected: "1 line"},
			{input: 2, expected: "2 lines"},
			{input: 0, expected: "0 line"},
		}

		for _, testcase := range testsuites {
			actual := normalizeLines(testcase.input)
			if testcase.expected != actual {
				t.Errorf("expected %s, but get %s", testcase.expected, actual)
			}
		}
	})
}

func TestPercentCovered(t *testing.T) {
	t.Run("percentCovered", func(t *testing.T) {
		testsuites := []struct {
			expected float64
			total    int
			covered  int
		}{
			{expected: 100.0, total: 10, covered: 10},
			{expected: 50.0, total: 10, covered: 5},
			{expected: 0.0, total: 10, covered: 0},
			{expected: 100.0, total: 0, covered: 0},
		}

		for _, testcase := range testsuites {
			actual := percentCovered(testcase.total, testcase.covered)
			if testcase.expected != actual {
				t.Errorf("expected %f, but get %f", testcase.expected, actual)
			}
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

// checkError checks the error and panic error at preparing testing environment steps.
func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
