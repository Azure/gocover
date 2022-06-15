package report

import (
	"fmt"
	"regexp"

	"golang.org/x/tools/cover"
)

type FullCoverage interface {
	BuildFullCoverageTree()
}

func NewFullCoverage(
	profiles []*cover.Profile,
	moduleHostPath string,
	excludes []string,
) (FullCoverage, error) {

	var excludesRegexps []*regexp.Regexp
	for _, ignorePattern := range excludes {
		reg, err := regexp.Compile(ignorePattern)
		if err != nil {
			return nil, fmt.Errorf("compile pattern %s: %w", ignorePattern, err)
		}
		excludesRegexps = append(excludesRegexps, reg)
	}

	return &fullCoverage{
		profiles:        profiles,
		excludesRegexps: excludesRegexps,
		coverageTree: &coverageTree{
			ModuleHostPath: moduleHostPath,
			Root:           NewTreeNode(moduleHostPath, false),
		},
	}, nil

}

var _ FullCoverage = (*fullCoverage)(nil)

type fullCoverage struct {
	profiles        []*cover.Profile
	excludesRegexps []*regexp.Regexp
	coverageTree    CoverageTree
}

func (full *fullCoverage) BuildFullCoverageTree() {
	full.ignore()
	full.covered()
	all := full.coverageTree.All()
	for _, v := range all {
		fmt.Println(
			v.Path,
			v.TotalLines,
			v.TotalCoveredLines,
			fmt.Sprintf("%.2f %%", float64(v.TotalCoveredLines)/float64(v.TotalLines)*100),
		)
	}
}

func (full *fullCoverage) ignore() {
	var filteredProfiles []*cover.Profile

	for _, p := range full.profiles {
		filter := false
		for _, reg := range full.excludesRegexps {
			if reg.MatchString(p.FileName) {
				filter = true
				break
			}
		}
		if !filter {
			filteredProfiles = append(filteredProfiles, p)
		}
	}

	full.profiles = filteredProfiles
}

func (full *fullCoverage) covered() {
	for _, p := range full.profiles {
		node := full.coverageTree.FindOrCreate(p.FileName)
		for _, b := range p.Blocks {
			node.TotalLines += int64(b.NumStmt)
			if b.Count > 0 {
				node.TotalCoveredLines += int64(b.NumStmt)
			}
		}
	}
	full.coverageTree.CollectCoverageData()
}
