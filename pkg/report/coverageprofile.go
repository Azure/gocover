package report

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/Azure/gocover/pkg/gittool"
	"golang.org/x/tools/cover"
)

// DiffCoverage expose the diff coverage statistics
type DiffCoverage interface {
	GenerateDiffCoverage() (*Statistics, error)
}

func NewDiffCoverage(
	profiles []*cover.Profile,
	changes []*gittool.Change,
	excludes []string,
	comparedBranch string,
) DiffCoverage {
	return &diffCoverage{
		comparedBranch: comparedBranch,
		excludes:       excludes,
		profiles:       profiles,
		changes:        changes,
	}
}

var _ DiffCoverage = (*diffCoverage)(nil)

// diffCoverage implements the DiffCoverage interface
// and generate the diff coverage statistics
type diffCoverage struct {
	comparedBranch string            // git diff base branch
	excludes       []string          // excludes files
	profiles       []*cover.Profile  // go unit test coverage profiles
	changes        []*gittool.Change // diff change between compared branch and HEAD commit
}

func (diff *diffCoverage) GenerateDiffCoverage() (*Statistics, error) {
	if err := diff.ignore(); err != nil {
		return nil, err
	}
	diff.filter()
	return diff.percentCovered(), nil
}

// ignore files that not accountting for diff coverage
// support standard regular expression
func (diff *diffCoverage) ignore() error {
	var filterProfiles []*cover.Profile

	for _, p := range diff.profiles {
		find := false
		for _, ignorePattern := range diff.excludes {
			reg, err := regexp.Compile(ignorePattern)
			if err != nil {
				return fmt.Errorf("compile pattern %s: %w", ignorePattern, err)
			}
			if reg.MatchString(p.FileName) {
				find = true
				break
			}
		}
		if !find {
			filterProfiles = append(filterProfiles, p)
		}
	}

	diff.profiles = filterProfiles
	return nil
}

// filter files that no change in current HEAD commit
func (diff *diffCoverage) filter() {
	var filterProfiles []*cover.Profile
	for _, p := range diff.profiles {
		for _, c := range diff.changes {
			if strings.HasSuffix(p.FileName, c.FileName) {
				filterProfiles = append(filterProfiles, p)
			}
		}
	}
	diff.profiles = filterProfiles
}

// percentCovered generate diff coverage profile
// using go unit test covreage profile and diff changes between two commits.
func (diff *diffCoverage) percentCovered() *Statistics {

	var totalLines int64 = 0
	var totalCovered int64 = 0
	totalViolations := 0

	var coverageProfiles []*CoverageProfile
	for _, p := range diff.profiles {

		change := findChange(p, diff.changes)
		if change == nil {
			continue
		}

		switch change.Mode {
		case gittool.NewMode:

			if coverageProfile := generateCoverageProfileWithNewMode(p, change); coverageProfile != nil {
				coverageProfiles = append(coverageProfiles, coverageProfile)

				totalViolations += len(coverageProfile.TotalViolationLines)
				totalLines += int64(coverageProfile.TotalLines)
				totalCovered += int64(coverageProfile.CoveredLines)
			}

		case gittool.ModifyMode:

			coverageProfile := generateCoverageProfileWithModifyMode(p, change)
			coverageProfiles = append(coverageProfiles, coverageProfile)

			totalViolations += len(coverageProfile.TotalViolationLines)
			totalLines += int64(coverageProfile.TotalLines)
			totalCovered += int64(coverageProfile.CoveredLines)

		case gittool.RenameMode:
		case gittool.DeleteMode:
		}
	}

	return &Statistics{
		ComparedBranch:       diff.comparedBranch,
		TotalLines:           int(totalLines),
		TotalCoveragePercent: int(float64(totalCovered) / float64(totalLines) * 100),
		TotalViolationLines:  totalViolations,
		CoverageProfile:      coverageProfiles,
	}
}

// generateCoverageProfileWithNewMode generates for new file
func generateCoverageProfileWithNewMode(profile *cover.Profile, change *gittool.Change) *CoverageProfile {
	var total, covered int64

	violationsMap := make(map[int]bool)
	// NumStmt indicates the number of statements in a code block, it not includes non statements,
	// which means that the value of NumStmt is less or equal tothe total numbers of the code block.
	for _, b := range profile.Blocks {
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		} else {
			// TODO:
			// This part does not reflect the accurate the violcation lines.
			for i := b.StartLine; i <= b.EndLine; i++ {
				violationsMap[i] = true
			}
		}
	}

	// it actually should not happen
	if total == 0 {
		return nil
	}

	violationLines := sortLines(violationsMap)

	coveredPercent := int(float64(covered) / float64(total) * 100)

	coverageProfile := &CoverageProfile{
		FileName:            change.FileName,
		TotalLines:          int(total),
		CoveredLines:        int(covered),
		CoveragePercent:     coveredPercent,
		TotalViolationLines: violationLines,
		ViolationSections: []*ViolationSection{
			{
				ViolationLines: violationLines,
				StartLine:      change.Sections[0].StartLine,
				EndLine:        change.Sections[0].EndLine,
				Contents:       change.Sections[0].Contents,
			},
		},
	}

	return coverageProfile
}

// generateCoverageProfileWithModifyMode generates for modify file
func generateCoverageProfileWithModifyMode(profile *cover.Profile, change *gittool.Change) *CoverageProfile {

	var total, covered int64
	var totalViolationLines []int

	var violationSections []*ViolationSection

	for _, b := range profile.Blocks {

		for _, s := range change.Sections {

			p1 := s.StartLine <= b.StartLine && s.EndLine >= b.EndLine
			p2 := s.StartLine >= b.StartLine && s.EndLine >= b.EndLine && b.EndLine > s.StartLine
			p3 := s.StartLine <= b.StartLine && s.EndLine <= b.EndLine && s.EndLine >= b.StartLine
			p4 := s.StartLine >= b.StartLine && s.EndLine <= b.EndLine

			if !(p1 || p2 || p3 || p4) {
				continue
			}

			total += int64(b.NumStmt)
			if b.Count > 0 {
				covered += int64(b.NumStmt)
			} else {
				violationsMap := make(map[int]bool)

				start := maxInt(s.StartLine, b.StartLine)
				end := minInt(s.EndLine, b.EndLine)

				for i := start; i <= end; i++ {
					violationsMap[i] = true
				}

				violationLines := sortLines(violationsMap)

				violationSections = append(violationSections, &ViolationSection{
					StartLine:      s.StartLine,
					EndLine:        s.EndLine,
					Contents:       s.Contents,
					ViolationLines: violationLines,
				})

				totalViolationLines = append(totalViolationLines, violationLines...)
			}
		}

	}

	if total == 0 {
		return nil
	}

	coveredPercent := int(float64(covered) / float64(total) * 100)

	return &CoverageProfile{
		FileName:            change.FileName,
		TotalLines:          int(total),
		CoveredLines:        int(covered),
		CoveragePercent:     coveredPercent,
		TotalViolationLines: totalViolationLines,
		ViolationSections:   violationSections,
	}

}

// findChange find the expected change by file name
func findChange(profile *cover.Profile, changes []*gittool.Change) *gittool.Change {
	for _, change := range changes {
		if strings.HasSuffix(profile.FileName, change.FileName) {
			return change
		}
	}
	return nil
}

// sortLines returns the lines in increasing order.
func sortLines(m map[int]bool) []int {
	var lines []int
	for k := range m {
		lines = append(lines, k)
	}
	sort.Ints(lines)
	return lines
}

// returns the maximum value between two int values.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// returns the minimum value between two int values.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
