package report

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/gittool"
	"golang.org/x/tools/cover"
)

func TestDiffCoverage(t *testing.T) {
	t.Run("NewDiffCoverage", func(t *testing.T) {
		_, err := NewDiffCoverage([]*cover.Profile{}, []*gittool.Change{}, []string{"**"}, "testbranch", "", "modulePath")
		if err == nil {
			t.Error("should return error")
		}

		diff, err := NewDiffCoverage(
			[]*cover.Profile{},
			[]*gittool.Change{},
			[]string{
				".*github.com/Azure/gocover/report/tool.go",
				"github.com/Azure/gocover/test/.*",
				"github.com/Azure/gocover/mock_*",
			}, "testbranch", "", "github.com/Azure/gocover")
		if err != nil {
			t.Errorf("should not return error: %s", err)
		}
		if diff == nil {
			t.Error("should not nil")
		}
	})

	t.Run("GenerateDiffCoverage", func(t *testing.T) {
		t.Run("generate percent coverage", func(t *testing.T) {
			repositoryPath := t.TempDir()

			diff := &diffCoverage{
				coverageTree:   NewCoverageTree(""),
				comparedBranch: "origin/main",
				repositoryPath: repositoryPath,
				profiles: []*cover.Profile{
					{
						FileName: "github.com/Azure/gocover/report/utils.go",
						Blocks: []cover.ProfileBlock{
							{
								StartLine: 1,
								EndLine:   3,
								NumStmt:   3,
								Count:     1,
							},
						},
					},
					{
						FileName: "github.com/Azure/gocover/report/tool.go",
						Blocks: []cover.ProfileBlock{
							{
								StartLine: 1,
								EndLine:   3,
								NumStmt:   3,
								Count:     1,
							},
							{
								StartLine: 4,
								EndLine:   6,
								NumStmt:   3,
								Count:     0,
							},
						},
					},
					{
						FileName: "github.com/Azure/gocover/report/common.go",
						Blocks: []cover.ProfileBlock{
							{
								StartLine: 1,
								EndLine:   3,
								NumStmt:   3,
								Count:     1,
							},
							{
								StartLine: 4,
								EndLine:   6,
								NumStmt:   3,
								Count:     0,
							},
						},
					},
					{
						FileName: "github.com/Azure/gocover/report/rename.go",
						Blocks: []cover.ProfileBlock{
							{
								StartLine: 1,
								EndLine:   3,
								NumStmt:   3,
								Count:     1,
							},
						},
					},
					{
						FileName: "github.com/Azure/gocover/report/delete.go",
						Blocks: []cover.ProfileBlock{
							{
								StartLine: 1,
								EndLine:   3,
								NumStmt:   3,
								Count:     1,
							},
						},
					},
				},
				changes: []*gittool.Change{
					{
						FileName: "report/tool.go",
						Mode:     gittool.ModifyMode,
						Sections: []*gittool.Section{
							{
								Operation: gittool.Add,
								Count:     3,
								StartLine: 1,
								EndLine:   3,
								Contents:  []string{"line1", "line2", "line3"},
							},
							{
								Operation: gittool.Add,
								Count:     3,
								StartLine: 4,
								EndLine:   6,
								Contents:  []string{"line4", "line5", "line6"},
							},
							{
								Operation: gittool.Add,
								Count:     3,
								StartLine: 7,
								EndLine:   9,
								Contents:  []string{"line7", "line8", "line9"},
							},
						},
					},
					{
						FileName: "report/common.go",
						Mode:     gittool.NewMode,
						Sections: []*gittool.Section{
							{
								Operation: gittool.Add,
								Count:     3,
								StartLine: 4,
								EndLine:   6,
								Contents:  []string{"line4", "line5", "line6"},
							},
						},
					},
					{
						FileName: "report/rename.go",
						Mode:     gittool.RenameMode,
						Sections: []*gittool.Section{
							{
								Operation: gittool.Add,
								Count:     3,
								StartLine: 4,
								EndLine:   6,
								Contents:  []string{"line4", "line5", "line6"},
							},
						},
					},
					{
						FileName: "report/delete.go",
						Mode:     gittool.DeleteMode,
						Sections: []*gittool.Section{
							{
								Operation: gittool.Delete,
								Count:     3,
								StartLine: 4,
								EndLine:   6,
								Contents:  []string{"line4", "line5", "line6"},
							},
						},
					},
				},
			}

			os.MkdirAll(filepath.Join(repositoryPath, "report"), os.ModePerm)
			ioutil.WriteFile(filepath.Join(repositoryPath, "report/tool.go"), []byte("tool"), 0644)
			ioutil.WriteFile(filepath.Join(repositoryPath, "report/common.go"), []byte("common"), 0644)
			ioutil.WriteFile(filepath.Join(repositoryPath, "report/rename.go"), []byte("rename"), 0644)
			ioutil.WriteFile(filepath.Join(repositoryPath, "report/delete.go"), []byte("delete"), 0644)

			statistics, _, err := diff.GenerateDiffCoverage()
			if err != nil {
				t.Errorf("should not error, but get %s", err)
			}
			if statistics == nil {
				t.Error("should return Statistics, but get nil")
			} else {
				if statistics.ComparedBranch != "origin/main" {
					t.Errorf("compare branch should be origin/main, but %s", statistics.ComparedBranch)
				}
				if statistics.TotalLines != 12 {
					t.Errorf("total lines should be 12, but get %d", statistics.TotalLines)
				}
				if statistics.TotalCoveragePercent != 50.0 {
					t.Errorf("coverage percent shoud be 50, but get %f", statistics.TotalCoveragePercent)
				}
				if statistics.TotalViolationLines != 6 {
					t.Errorf("total violation lines should be 6, but get %d", statistics.TotalViolationLines)
				}
				if len(statistics.CoverageProfile) != 2 {
					t.Errorf("should have 2 coverage profile, but get: %d", len(statistics.CoverageProfile))
				}
			}
		})
	})

	t.Run("ignore", func(t *testing.T) {
		t.Run("ignore files", func(t *testing.T) {
			diff := &diffCoverage{
				profiles: []*cover.Profile{
					{
						FileName: "github.com/Azure/gocover/report/tool.go",
					},
					{
						FileName: "github.com/Azure/gocover/mock_interface/a.go",
					},
					{
						FileName: "github.com/Azure/gocover/test/b.go",
					},
					{
						FileName: "github.com/Azure/gocover/utils/common.go",
					},
				},
			}

			for _, p := range []string{
				".*github.com/Azure/gocover/report/tool.go",
				"github.com/Azure/gocover/test/.*",
				"github.com/Azure/gocover/mock_*",
			} {
				reg := regexp.MustCompile(p)
				diff.excludesRegexps = append(diff.excludesRegexps, reg)
			}

			diff.ignore()

			if len(diff.profiles) != 1 {
				t.Errorf("after ignore, should have 1 profile, but get: %d", len(diff.profiles))
			}
			if diff.profiles[0].FileName != "github.com/Azure/gocover/utils/common.go" {
				t.Errorf("after ignore, only common.go is left, but get: %s", diff.profiles[0].FileName)
			}
		})
	})

	t.Run("filter", func(t *testing.T) {
		diff := &diffCoverage{
			profiles: []*cover.Profile{
				{
					FileName: "github.com/Azure/gocover/report/tool.go",
				},
				{
					FileName: "github.com/Azure/gocover/mock_interface/a.go",
				},
				{
					FileName: "github.com/Azure/gocover/test/b.go",
				},
				{
					FileName: "github.com/Azure/gocover/utils/common.go",
				},
			},
			changes: []*gittool.Change{
				{
					FileName: "utils/common.go",
				},
			},
		}

		diff.filter()
		if len(diff.profiles) != 1 {
			t.Errorf("after filter, should have 1 profile, but get: %d", len(diff.profiles))
		}
		if diff.profiles[0].FileName != "github.com/Azure/gocover/utils/common.go" {
			t.Errorf("after filter, only common.go is left, but get: %s", diff.profiles[0].FileName)
		}
	})

	t.Run("percentCovered", func(t *testing.T) {
		diff := &diffCoverage{
			coverageTree:   NewCoverageTree(""),
			comparedBranch: "origin/main",
			profiles: []*cover.Profile{
				{
					FileName: "github.com/Azure/gocover/report/utils.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
					},
				},
				{
					FileName: "github.com/Azure/gocover/report/tool.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
						{
							StartLine: 4,
							EndLine:   6,
							NumStmt:   3,
							Count:     0,
						},
					},
				},
				{
					FileName: "github.com/Azure/gocover/report/common.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
						{
							StartLine: 4,
							EndLine:   6,
							NumStmt:   3,
							Count:     0,
						},
					},
				},
				{
					FileName: "github.com/Azure/gocover/report/rename.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
					},
				},
				{
					FileName: "github.com/Azure/gocover/report/delete.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
					},
				},
				{
					FileName: "github.com/Azure/gocover/report/ignore.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 1,
							EndLine:   3,
							NumStmt:   3,
							Count:     1,
						},
					},
				},
			},
			changes: []*gittool.Change{
				{
					FileName: "report/tool.go",
					Mode:     gittool.ModifyMode,
					Sections: []*gittool.Section{
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 1,
							EndLine:   3,
							Contents:  []string{"line1", "line2", "line3"},
						},
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 4,
							EndLine:   6,
							Contents:  []string{"line4", "line5", "line6"},
						},
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 7,
							EndLine:   9,
							Contents:  []string{"line7", "line8", "line9"},
						},
					},
				},
				{
					FileName: "report/common.go",
					Mode:     gittool.NewMode,
					Sections: []*gittool.Section{
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 4,
							EndLine:   6,
							Contents:  []string{"line4", "line5", "line6"},
						},
					},
				},
				{
					FileName: "report/rename.go",
					Mode:     gittool.RenameMode,
					Sections: []*gittool.Section{
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 4,
							EndLine:   6,
							Contents:  []string{"line4", "line5", "line6"},
						},
					},
				},
				{
					FileName: "report/delete.go",
					Mode:     gittool.DeleteMode,
					Sections: []*gittool.Section{
						{
							Operation: gittool.Delete,
							Count:     3,
							StartLine: 4,
							EndLine:   6,
							Contents:  []string{"line4", "line5", "line6"},
						},
					},
				},
				{
					FileName: "report/ignore.go",
					Mode:     gittool.DeleteMode,
					Sections: []*gittool.Section{
						{
							Operation: gittool.Add,
							Count:     3,
							StartLine: 4,
							EndLine:   6,
							Contents:  []string{"line4", "line5", "line6"},
						},
					},
				},
			},
			ignoreProfiles: map[string]*annotation.IgnoreProfile{
				"report/ignore.go": {Type: annotation.FILE_IGNORE, IgnoreBlocks: make(map[cover.ProfileBlock]*annotation.IgnoreBlock)},
			},
		}

		statistics := diff.percentCovered()
		if statistics == nil {
			t.Error("should return Statistics, but get nil")
		} else {
			if statistics.ComparedBranch != "origin/main" {
				t.Errorf("compare branch should be origin/main, but %s", statistics.ComparedBranch)
			}
			if statistics.TotalLines != 12 {
				t.Errorf("total lines should be 12, but get %d", statistics.TotalLines)
			}
			if statistics.TotalCoveragePercent != 50.0 {
				t.Errorf("coverage percent shoud be 50, but get %f", statistics.TotalCoveragePercent)
			}
			if statistics.TotalViolationLines != 6 {
				t.Errorf("total violation lines should be 6, but get %d", statistics.TotalViolationLines)
			}
			if len(statistics.CoverageProfile) != 2 {
				t.Errorf("should have 2 coverage profile, but get: %d", len(statistics.CoverageProfile))
			}
		}
	})

	t.Run("generateIgnoreProfile", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = ioutil.WriteFile(filepath.Join(tempDir, "foo.go"), []byte(`
		if err != nil {
			//+gocover:ignore:block
			return err
		}`), 0644)

		diff := &diffCoverage{
			repositoryPath: tempDir,
			changes: []*gittool.Change{
				{FileName: "foo.go"},
				{FileName: "bar.go"},
			},
			profiles: []*cover.Profile{
				{
					FileName: "github.com/Azure/gocover/foo.go",
					Blocks: []cover.ProfileBlock{
						{
							StartLine: 3,
							EndLine:   5,
						},
					},
				},
			},
		}

		diff.generateIgnoreProfile()
	})
}

func TestGenerateCoverageProfileWithModifyMode(t *testing.T) {
	t.Run("generateCoverageProfileWithModifyMode", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
			Blocks: []cover.ProfileBlock{
				{
					StartLine: 1,
					EndLine:   3,
					NumStmt:   3,
					Count:     1,
				},
				{
					StartLine: 4,
					EndLine:   6,
					NumStmt:   3,
					Count:     0,
				},
			},
		}
		change := &gittool.Change{
			FileName: "report/tool.go",
			Mode:     gittool.ModifyMode,
			Sections: []*gittool.Section{
				{
					Operation: gittool.Add,
					Count:     3,
					StartLine: 1,
					EndLine:   3,
					Contents:  []string{"line1", "line2", "line3"},
				},
				{
					Operation: gittool.Add,
					Count:     3,
					StartLine: 4,
					EndLine:   6,
					Contents:  []string{"line4", "line5", "line6"},
				},
				{
					Operation: gittool.Add,
					Count:     3,
					StartLine: 7,
					EndLine:   9,
					Contents:  []string{"line7", "line8", "line9"},
				},
				{
					Operation: gittool.Add,
					Count:     3,
					StartLine: 10,
					EndLine:   12,
					Contents:  []string{"line10", "line11", "line12"},
				},
			},
		}

		coverageProfile := generateCoverageProfileWithModifyMode(profile, change, &annotation.IgnoreProfile{
			Type:         annotation.BLOCK_IGNORE,
			IgnoreBlocks: make(map[cover.ProfileBlock]*annotation.IgnoreBlock),
		})
		if coverageProfile.FileName != "report/tool.go" {
			t.Errorf("expect filename %s, but get %s", "report/tool.go", coverageProfile.FileName)
		}
		c := float64(coverageProfile.CoveredLines) / float64(coverageProfile.TotalLines) * 100
		if c != 50.0 {
			t.Errorf("coverage percent shoud be 50, but get %f", c)
		}
		if coverageProfile.TotalLines != 6 {
			t.Errorf("total lines should be 6, but get %d", coverageProfile.TotalLines)
		}
		if coverageProfile.CoveredLines != 3 {
			t.Errorf("covered lines should be 3, but get %d", coverageProfile.CoveredLines)
		}
		if len(coverageProfile.TotalViolationLines) != 3 {
			t.Errorf("total violation lines should be 3, but get %d", len(coverageProfile.TotalViolationLines))
		}
		if len(coverageProfile.ViolationSections) != 1 {
			t.Errorf("should have 1 violation seciton, but get %d", len(coverageProfile.ViolationSections))
		}
		section := coverageProfile.ViolationSections[0]
		if section.StartLine != 4 {
			t.Errorf("start line should be 4, but %d", section.StartLine)
		}
		if section.EndLine != 6 {
			t.Errorf("end line should be 6, but %d", section.EndLine)
		}
		if strings.Join(section.Contents, ",") != "line4,line5,line6" {
			t.Errorf("content should be line4,line5,line6, but get %s", strings.Join(section.Contents, ","))
		}
	})

	t.Run("generateCoverageProfileWithModifyMode", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
			Blocks:   []cover.ProfileBlock{},
		}
		change := &gittool.Change{
			FileName: "report/tool.go",
			Mode:     gittool.ModifyMode,
			Sections: []*gittool.Section{},
		}

		coverageProfile := generateCoverageProfileWithModifyMode(profile, change, nil)
		if coverageProfile != nil {
			t.Error("should return nil when no lines in the profile")
		}
	})
}

func TestGenerateCoverageProfileWithNewMode(t *testing.T) {
	t.Run("generateCoverageProfileWithNewMode", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
			Blocks: []cover.ProfileBlock{
				{
					StartLine: 1,
					EndLine:   3,
					NumStmt:   3,
					Count:     1,
				},
				{
					StartLine: 4,
					EndLine:   6,
					NumStmt:   3,
					Count:     0,
				},
				{
					StartLine: 7,
					EndLine:   10,
					NumStmt:   3,
					Count:     0,
				},
			},
		}
		change := &gittool.Change{
			FileName: "report/tool.go",
			Mode:     gittool.NewMode,
			Sections: []*gittool.Section{
				{
					Operation: gittool.Add,
					Count:     3,
					StartLine: 4,
					EndLine:   6,
					Contents:  []string{"line4", "line5", "line6"},
				},
			},
		}

		coverageProfile := generateCoverageProfileWithNewMode(profile, change, &annotation.IgnoreProfile{
			Type: annotation.BLOCK_IGNORE,
			IgnoreBlocks: map[cover.ProfileBlock]*annotation.IgnoreBlock{
				cover.ProfileBlock{StartLine: 7, EndLine: 10, NumStmt: 3, Count: 0}: &annotation.IgnoreBlock{Contents: []string{"line7", "line8", "line9", "line10"}, Lines: []int{7, 8, 9, 10}},
			},
		})
		if coverageProfile.FileName != "report/tool.go" {
			t.Errorf("expect filename %s, but get %s", "report/tool.go", coverageProfile.FileName)
		}
		c := float64(coverageProfile.CoveredLines) / float64(coverageProfile.TotalEffectiveLines) * 100
		if c != 50.0 {
			t.Errorf("coverage percent shoud be 50, but get %f", c)
		}
		if coverageProfile.TotalLines != 9 {
			t.Errorf("total lines should be 9, but get %d", coverageProfile.TotalLines)
		}
		if coverageProfile.TotalLines != 9 {
			t.Errorf("total lines should be 9, but get %d", coverageProfile.TotalEffectiveLines)
		}
		if coverageProfile.CoveredLines != 3 {
			t.Errorf("covered lines should be 3, but get %d", coverageProfile.CoveredLines)
		}
		if len(coverageProfile.TotalViolationLines) != 3 {
			t.Errorf("total violation lines should be 3, but get %d", len(coverageProfile.TotalViolationLines))
		}
		if len(coverageProfile.ViolationSections) != 1 {
			t.Errorf("should have 1 violation seciton, but get %d", len(coverageProfile.ViolationSections))
		}
		section := coverageProfile.ViolationSections[0]
		if section.StartLine != 4 {
			t.Errorf("start line should be 4, but %d", section.StartLine)
		}
		if section.EndLine != 6 {
			t.Errorf("end line should be 6, but %d", section.EndLine)
		}
		if strings.Join(section.Contents, ",") != "line4,line5,line6" {
			t.Errorf("content should be line4,line5,line6, but get %s", strings.Join(section.Contents, ","))
		}
	})

	t.Run("generateCoverageProfileWithNewMode", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
			Blocks:   []cover.ProfileBlock{},
		}
		change := &gittool.Change{
			FileName: "report/tool.go",
			Mode:     gittool.NewMode,
			Sections: []*gittool.Section{},
		}

		coverageProfile := generateCoverageProfileWithNewMode(profile, change, nil)
		if coverageProfile != nil {
			t.Error("should return nil when no lines in the profile")
		}
	})
}

func TestFindCoverProfile(t *testing.T) {
	t.Run("find cover profile", func(t *testing.T) {
		profiles := []*cover.Profile{
			{
				FileName: "github.com/Azure/gocover/report/tool.go",
			},
		}
		change := &gittool.Change{
			FileName: "report/tool.go",
			Mode:     gittool.NewMode,
		}

		profile := findCoverProfile(change, profiles)
		if profile == nil {
			t.Errorf("profile should not be nil")
		}
	})

	t.Run("cannot find cover profile", func(t *testing.T) {
		profiles := []*cover.Profile{
			{
				FileName: "github.com/Azure/gocover/report/tool.go",
			},
		}
		change := &gittool.Change{
			FileName: "report/foo.go",
			Mode:     gittool.NewMode,
		}

		profile := findCoverProfile(change, profiles)
		if profile != nil {
			t.Errorf("profile should be nil")
		}
	})
}

func TestFindChange(t *testing.T) {
	t.Run("find change", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
		}
		changes := []*gittool.Change{
			{
				FileName: "report/generator.go",
				Mode:     gittool.ModifyMode,
			},
			{
				FileName: "report/tool.go",
				Mode:     gittool.NewMode,
			},
		}

		change := findChange(profile, changes)
		if change != changes[1] {
			t.Errorf("should return expect change")
		}
	})

	t.Run("not find change", func(t *testing.T) {
		profile := &cover.Profile{
			FileName: "github.com/Azure/gocover/report/tool.go",
		}
		changes := []*gittool.Change{
			{
				FileName: "report/generator.go",
				Mode:     gittool.ModifyMode,
			},
			{
				FileName: "gitdiff/tool.go",
				Mode:     gittool.NewMode,
			},
		}

		if findChange(profile, changes) != nil {
			t.Error("change should nil")
		}
	})
}

func TestSortLines(t *testing.T) {
	t.Run("sortLines", func(t *testing.T) {
		m := make(map[int]bool)
		m[3] = true
		m[1] = true
		m[2] = true
		m[3] = true

		lines := sortLines(m)
		if sort.IntsAreSorted(lines) != true {
			t.Error("should sort in increasing order")
		}
	})
}

func TestIsSubFolderTo(t *testing.T) {
	t.Run("isSubFolderTo", func(t *testing.T) {
		testsuites := []struct {
			parentDir string
			filepath  string
			expected  bool
		}{
			{parentDir: "github.com/Azure/gocover/report/tool.go", filepath: "utils/common.go", expected: false},
			{parentDir: "github.com/Azure/gocover/report/tool.go", filepath: "report/tool.go", expected: true},
		}

		for _, testcase := range testsuites {
			actual := isSubFolderTo(testcase.parentDir, testcase.filepath)
			if actual != testcase.expected {
				t.Errorf("expected %t, but get %t", testcase.expected, actual)
			}
		}
	})
}

// 1  package foo
// 2
// 3  import "fmt"
// 4
// 5 func plus(a, b int) int {
// 6 	fmt.Println("plus")
// 7
// 8 	if a > b {
// 9 		fmt.Println(a + b)
// 10 		return a + b
// 11}
// 12
// 13 	fmt.Println(b + a)
// 14 	return b + a
// 15}
//
// Above code would generate the cover profile following:
// github.com/Azure/gocover/pkg/annotation/foo.go:5.25,8.11 2 0
// github.com/Azure/gocover/pkg/annotation/foo.go:8.11,11.3 2 0
// github.com/Azure/gocover/pkg/annotation/foo.go:13.2,14.14 2 0
func TestFindProfileBlock(t *testing.T) {

	t.Run("findProfileBlock", func(t *testing.T) {
		block0 := cover.ProfileBlock{StartLine: 5, StartCol: 25, EndLine: 8, EndCol: 11}
		block1 := cover.ProfileBlock{StartLine: 8, StartCol: 11, EndLine: 11, EndCol: 3}
		block2 := cover.ProfileBlock{StartLine: 13, StartCol: 2, EndLine: 14, EndCol: 14}

		blocks := []cover.ProfileBlock{block0, block1, block2}

		testsuites := []struct {
			lineNumber int
			line       string
			expect     *cover.ProfileBlock
		}{
			{lineNumber: 4, line: ``, expect: nil},
			{lineNumber: 5, line: `func plus(a, b int) int {`, expect: nil},
			{lineNumber: 6, line: `	fmt.Println("plus")`, expect: &blocks[0]},
			{lineNumber: 7, line: ``, expect: &blocks[0]},
			{lineNumber: 8, line: `	if a > b {`, expect: &blocks[0]},
			{lineNumber: 9, line: `		fmt.Println(a + b)`, expect: &blocks[1]},
			{lineNumber: 10, line: `		return a + b`, expect: &blocks[1]},
			{lineNumber: 11, line: `	}`, expect: &blocks[1]},
			{lineNumber: 12, line: ``, expect: nil},
			{lineNumber: 13, line: `	fmt.Println(b + a)`, expect: &blocks[2]},
			{lineNumber: 14, line: `	return b + a`, expect: &blocks[2]},
			{lineNumber: 15, line: `}`, expect: nil},
		}

		for _, testcase := range testsuites {
			actual := findProfileBlock(blocks, testcase.lineNumber, testcase.line)
			if actual != testcase.expect {
				t.Errorf("expect %v, but get %v", testcase.expect, actual)
			}
		}
	})

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
