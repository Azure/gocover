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

	blocks := profile.Blocks
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].StartLine < blocks[j].StartLine
	})

	violationsMap := make(map[int]bool)
	// NumStmt indicates the number of statements in a code block, it does not means the line, because a statement may have several lines,
	// which means that the value of NumStmt is less or equal tothe total numbers of the code block.
	for _, b := range blocks {
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		} else {
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

	blocks := profile.Blocks
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].StartLine < blocks[j].StartLine
	})

	var total, covered int64
	var totalViolationLines []int
	var violationSections []*ViolationSection

	// for each section contents, find each line in profile block and judge it
	for _, section := range change.Sections {

		var violationLines []int
		for lineNo := section.StartLine; lineNo <= section.EndLine; lineNo++ {
			block := binarySeachForProfileBlock(blocks, 0, len(blocks)-1, lineNo)
			if block == nil {
				continue
			}

			// check line?

			total++
			if block.Count > 0 {
				covered++
			} else {
				violationLines = append(violationLines, lineNo)
			}

		}

		if len(violationLines) != 0 {
			violationSections = append(violationSections, &ViolationSection{
				StartLine:      section.StartLine,
				EndLine:        section.EndLine,
				Contents:       section.Contents,
				ViolationLines: violationLines,
			})

			totalViolationLines = append(totalViolationLines, violationLines...)
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

func binarySeachForProfileBlock(blocks []cover.ProfileBlock, left int, right int, lineNo int) *cover.ProfileBlock {

	var mid int
	for left <= right {
		mid = (left + right) >> 1
		// current line is in this block
		if blocks[mid].StartLine <= lineNo && blocks[mid].EndLine >= lineNo {
			return &blocks[mid]
		}
		if blocks[mid].StartLine > lineNo {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}

	return nil
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
