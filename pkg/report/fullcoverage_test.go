package report

import (
	"regexp"
	"testing"

	"golang.org/x/tools/cover"
)

func TestFullCoverage(t *testing.T) {
	t.Run("NewFullCoverage", func(t *testing.T) {
		_, err := NewFullCoverage([]*cover.Profile{}, "github.com/Azure/gocover", []string{"**"})
		if err == nil {
			t.Error("should return error")
		}

		diff, err := NewFullCoverage(
			[]*cover.Profile{},
			"github.com/Azure/gocover",
			[]string{
				".*github.com/Azure/gocover/report/tool.go",
				"github.com/Azure/gocover/test/.*",
				"github.com/Azure/gocover/mock_*",
			})
		if err != nil {
			t.Errorf("should not return error: %s", err)
		}
		if diff == nil {
			t.Error("should not nil")
		}
	})

	t.Run("ignore", func(t *testing.T) {
		t.Run("ignore files", func(t *testing.T) {
			diff := &fullCoverage{
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

	t.Run("covered", func(t *testing.T) {
		full := &fullCoverage{
			coverageTree: NewCoverageTree(""),
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
		}

		full.covered()
	})

	t.Run("BuildFullCoverageTree", func(t *testing.T) {
		full := &fullCoverage{
			coverageTree: NewCoverageTree(""),
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
		}
		full.BuildFullCoverageTree()
	})

}
