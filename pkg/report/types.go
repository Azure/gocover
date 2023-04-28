package report

import (
	"html/template"
)

type StatisticsType string

const (
	FullStatisticsType StatisticsType = "full"
	DiffStatisticsType StatisticsType = "diff"
)

// Statistics represents the total diff coverage for the HEAD commit.
// It contains the total coverage and possible coverage profile.
type Statistics struct {
	// ComparedBranch the branch that diff compared with.
	ComparedBranch string
	// TotalLines represents the total lines that count for coverage.
	TotalLines int
	// TotalEffectiveLines indicates effective lines for the coverage profile.
	TotalEffectiveLines int
	// TotalIgnoredLines indicates the lines ignored.
	TotalIgnoredLines int
	// TotalCoveredLines indicates total covered lines that count for coverage.
	TotalCoveredLines int
	// CoveredButIgnoredLines indicates the lines that covered but ignored.
	TotalCoveredButIgnoredLines int
	// TotalViolationLines represents all the lines that miss test coverage.
	TotalViolationLines int
	// TotalCoveragePercent represents the coverage percent for current diff.
	TotalCoveragePercent float64
	// TotalCoverageWithoutIgnore represents the coverage percent for current diff without ignorance
	TotalCoverageWithoutIgnore float64
	// CoverageProfile represents the coverage profile for a specific file.
	CoverageProfile []*CoverageProfile
	// StatisticsType indicates which type the Statistics is.
	StatisticsType StatisticsType
	// exclude files that won't take participate to coverage calculation.
	ExcludeFiles []string
}

// CoverageProfile represents the test coverage information for a file.
type CoverageProfile struct {
	// FileName indicates which file belongs to this coverage profile.
	FileName string
	// TotalLines indicates total lines of the entire repo/module.
	TotalLines int
	// TotalEffectiveLines indicates effective lines for the coverage profile.
	TotalEffectiveLines int
	// TotalIgnoredLines indicates the lines ignored.
	TotalIgnoredLines int
	// CoveredLines indicates covered lines of this coverage profile.
	CoveredLines int
	// CoveredButIgnoredLines indicates the lines that covered but ignored.
	CoveredButIgnoredLines int
	// CoveragePercent indicates the diff coverage percent for this file.
	TotalViolationLines []int
	// ViolationSections indicates the violation sections that miss full coverage.
	ViolationSections []*ViolationSection
	// CodeSnippet represents the output of the ViolationSections, it's calculated from ViolationSections.
	CodeSnippet []template.HTML
}

// ViolationSection represents a portion of the change that miss unit test coverage.
type ViolationSection struct {
	// ViolationLines indicates which line miss the coverage.
	ViolationLines []int
	// StartLine indicates the start line of the section.
	StartLine int
	// EndLine indicates the end line of the section.
	EndLine int
	// Contents contains [StartLine..EndLine] lines from the source file.
	Contents []string
}
