package report

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Azure/gocover/pkg/annotation"
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
	repositoryPath string,
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
		coverageTree:    NewCoverageTree(""),
		repositoryPath:  repositoryPath,
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
	repositoryPath  string
	ignoreProfiles  map[string]*annotation.IgnoreProfile
	coverProfiles   map[string]*cover.Profile
	coverageTree    CoverageTree
}

func (diff *diffCoverage) GenerateDiffCoverage() (*Statistics, error) {
	diff.ignore()
	diff.filter()
	diff.generateIgnoreProfile()
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

func (diff *diffCoverage) generateIgnoreProfile() {
	ignoreProfiles := make(map[string]*annotation.IgnoreProfile)
	coverProfiles := make(map[string]*cover.Profile)

	for _, c := range diff.changes {
		p := findCoverProfile(c, diff.profiles)
		if p == nil {
			continue
		}

		sort.Sort(blocksByStart(p.Blocks))
		coverProfiles[c.FileName] = p

		ignoreProfile, err := annotation.ParseIgnoreProfiles(filepath.Join(diff.repositoryPath, c.FileName), p)
		if err == nil {
			ignoreProfiles[c.FileName] = ignoreProfile
			for _, b := range ignoreProfile.IgnoreBlocks {
				fmt.Fprintln(os.Stdout, b.Annotation)
				for i := 0; i < len(b.Contents); i++ {
					fmt.Fprintf(os.Stdout, "%d %s\n", b.Lines[i], b.Contents[i])
				}
			}
		} else {
			fmt.Println(err)
		}
	}

	diff.ignoreProfiles = ignoreProfiles
	diff.coverProfiles = coverProfiles
}

// findCoverProfile find the expected cover profile by file name.
func findCoverProfile(change *gittool.Change, profiles []*cover.Profile) *cover.Profile {
	for _, profile := range profiles {
		if isSubFolderTo(profile.FileName, change.FileName) {
			return profile
		}
	}
	return nil
}

// percentCovered generate diff coverage profile
// using go unit test covreage profile and diff changes between two commits.
func (diff *diffCoverage) percentCovered() *Statistics {

	var coverageProfiles []*CoverageProfile
	for _, p := range diff.profiles {

		change := findChange(p, diff.changes)
		if change == nil {
			continue
		}

		ignoreProfile, ok := diff.ignoreProfiles[change.FileName]
		if ok && ignoreProfile.Type == annotation.FILE_IGNORE {
			continue
		}

		switch change.Mode {
		case gittool.NewMode:

			if coverageProfile := generateCoverageProfileWithNewMode(p, change, ignoreProfile); coverageProfile != nil {
				coverageProfiles = append(coverageProfiles, coverageProfile)

				node := diff.coverageTree.FindOrCreate(change.FileName)
				node.TotalLines += int64(coverageProfile.TotalLines)
				node.TotalCoveredLines += int64(coverageProfile.CoveredLines)
				node.TotalViolationLines += int64(len(coverageProfile.TotalViolationLines))
				node.CoverageProfile = coverageProfile
			}

		case gittool.ModifyMode:

			if coverageProfile := generateCoverageProfileWithModifyMode(p, change, ignoreProfile); coverageProfile != nil {
				coverageProfiles = append(coverageProfiles, coverageProfile)

				node := diff.coverageTree.FindOrCreate(change.FileName)
				node.TotalLines += int64(coverageProfile.TotalLines)
				node.TotalCoveredLines += int64(coverageProfile.CoveredLines)
				node.TotalViolationLines += int64(len(coverageProfile.TotalViolationLines))
				node.CoverageProfile = coverageProfile
			}

		case gittool.RenameMode:
		case gittool.DeleteMode:
		}
	}

	diff.coverageTree.CollectCoverageData()
	all := diff.coverageTree.Statistics()

	return &Statistics{
		ComparedBranch:       diff.comparedBranch,
		TotalLines:           int(all.TotalLines),
		TotalCoveredLines:    int(all.TotalCoveredLines),
		TotalCoveragePercent: float64(all.TotalCoveredLines) / float64(all.TotalLines) * 100,
		TotalViolationLines:  int(all.TotalViolationLines),
		CoverageProfile:      coverageProfiles,
	}
}

// generateCoverageProfileWithNewMode generates for new file
func generateCoverageProfileWithNewMode(profile *cover.Profile, change *gittool.Change, ignoreProfile *annotation.IgnoreProfile) *CoverageProfile {
	var total, covered, ignored int64

	violationsMap := make(map[int]bool)
	// NumStmt indicates the number of statements in a code block, it does not means the line, because a statement may have several lines,
	// which means that the value of NumStmt is less or equal to the total numbers of the code block.
	for _, b := range profile.Blocks {

		if ignoreProfile.IgnoreBlocks != nil {
			if _, ok := ignoreProfile.IgnoreBlocks[&b]; ok {
				total += int64(b.NumStmt)
				ignored += int64(b.NumStmt)
				continue
			}
		}

		total += int64(b.NumStmt)
		// effective += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		} else {

			for lineNum := b.StartLine; lineNum <= b.EndLine; lineNum++ {
				violationsMap[lineNum] = true
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
func generateCoverageProfileWithModifyMode(profile *cover.Profile, change *gittool.Change, ignoreProfile *annotation.IgnoreProfile) *CoverageProfile {

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

			if ignoreProfile != nil {
				if _, ok := ignoreProfile.IgnoreBlocks[block]; ok {
					continue
				}
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
