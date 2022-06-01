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
) (DiffCoverage, error) {

	var excludesRegexps []*regexp.Regexp
	for _, ignorePattern := range excludes {
		reg, err := regexp.Compile(ignorePattern)
		if err != nil {
			return nil, fmt.Errorf("compile pattern %s: %w", ignorePattern, err)
		}
		excludesRegexps = append(excludesRegexps, reg)
	}

	return &diffCoverage{
		comparedBranch:  comparedBranch,
		profiles:        profiles,
		changes:         changes,
		excludesRegexps: excludesRegexps,
	}, nil
}

var _ DiffCoverage = (*diffCoverage)(nil)

// diffCoverage implements the DiffCoverage interface
// and generate the diff coverage statistics
type diffCoverage struct {
	comparedBranch  string            // git diff base branch
	profiles        []*cover.Profile  // go unit test coverage profiles
	changes         []*gittool.Change // diff change between compared branch and HEAD commit
	excludesRegexps []*regexp.Regexp  // excludes files regexp patterns
}

func (diff *diffCoverage) GenerateDiffCoverage() (*Statistics, error) {
	diff.ignore()
	diff.filter()
	return diff.percentCovered(), nil
}

// ignore files that not accountting for diff coverage
// support standard regular expression
func (diff *diffCoverage) ignore() {
	var filteredProfiles []*cover.Profile

	for _, p := range diff.profiles {
		filter := false
		for _, reg := range diff.excludesRegexps {
			if reg.MatchString(p.FileName) {
				filter = true
				break
			}
		}
		if !filter {
			filteredProfiles = append(filteredProfiles, p)
		}
	}

	diff.profiles = filteredProfiles
}

// filter files that no change in current HEAD commit
func (diff *diffCoverage) filter() {
	var filterProfiles []*cover.Profile
	for _, p := range diff.profiles {
		for _, c := range diff.changes {
			if isSubFolderTo(p.FileName, c.FileName) {
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
		TotalCoveredLines:    int(totalCovered),
		TotalCoveragePercent: float64(totalCovered) / float64(totalLines) * 100,
		TotalViolationLines:  totalViolations,
		CoverageProfile:      coverageProfiles,
	}
}

// generateCoverageProfileWithNewMode generates for new file
func generateCoverageProfileWithNewMode(profile *cover.Profile, change *gittool.Change) *CoverageProfile {
	var total, covered int64

	sort.Sort(blocksByStart(profile.Blocks))

	violationsMap := make(map[int]bool)
	// NumStmt indicates the number of statements in a code block, it does not means the line, because a statement may have several lines,
	// which means that the value of NumStmt is less or equal tothe total numbers of the code block.
	for _, b := range profile.Blocks {
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

	coverageProfile := &CoverageProfile{
		FileName:            change.FileName,
		TotalLines:          int(total),
		CoveredLines:        int(covered),
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

	sort.Sort(blocksByStart(profile.Blocks))

	var total, covered int64
	var totalViolationLines []int
	var violationSections []*ViolationSection

	// for each section contents, find each line in profile block and judge it
	for _, section := range change.Sections {

		var violationLines []int
		for lineNo := section.StartLine; lineNo <= section.EndLine; lineNo++ {

			block := findProfileBlock(profile.Blocks, lineNo)
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

	return &CoverageProfile{
		FileName:            change.FileName,
		TotalLines:          int(total),
		CoveredLines:        int(covered),
		TotalViolationLines: totalViolationLines,
		ViolationSections:   violationSections,
	}
}

// findProfileBlock find the expected profile block by line number.
// as a profile block has start line and end line, we use binary search to search for it using start line first,
// then validate the end line.
func findProfileBlock(blocks []cover.ProfileBlock, lineNo int) *cover.ProfileBlock {
	idx := sort.Search(len(blocks), func(i int) bool {
		return blocks[i].StartLine >= lineNo
	})

	// no suitable block, index is out of range
	if idx == len(blocks) {
		idx--
		if blocks[idx].StartLine <= lineNo && blocks[idx].EndLine >= lineNo {
			return &blocks[idx]
		} else {
			return nil
		}
	}

	for idx >= 0 {
		if blocks[idx].StartLine <= lineNo && blocks[idx].EndLine >= lineNo {
			return &blocks[idx]
		}
		idx--
	}
	return nil
}

// findChange find the expected change by file name.
func findChange(profile *cover.Profile, changes []*gittool.Change) *gittool.Change {
	for _, change := range changes {
		if isSubFolderTo(profile.FileName, change.FileName) {
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

// isSubFolderTo check whether specified filepath is a part of parent path.
func isSubFolderTo(parentDir, filepath string) bool {
	return strings.HasSuffix(parentDir, filepath)
}

// interface for sorting profile block slice by start line
type blocksByStart []cover.ProfileBlock

func (b blocksByStart) Len() int      { return len(b) }
func (b blocksByStart) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b blocksByStart) Less(i, j int) bool {
	bi, bj := b[i], b[j]
	return bi.StartLine < bj.StartLine || bi.StartLine == bj.StartLine && bi.StartCol < bj.StartCol
}
