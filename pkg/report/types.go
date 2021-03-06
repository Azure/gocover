package report

import (
	"html/template"
)

// Statistics represents the total diff coverage for the HEAD commit.
// It contains the total coverage and possible coverage profile.
type Statistics struct {
	// ComparedBranch the branch that diff compared with.
	ComparedBranch string
	// TotalLines represents the total lines that count for coverage.
	TotalLines int
	// TotalCoveredLines indicates total covered lines that count for coverage.
	TotalCoveredLines int
	// TotalViolationLines represents all the lines that miss test coverage.
	TotalViolationLines int
	// TotalCoveragePercent represents the coverage percent for current diff.
	TotalCoveragePercent float64
	// CoverageProfile represents the coverage profile for a specific file.
	CoverageProfile []*CoverageProfile
}

// CoverageProfile represents the test coverage information for a file.
type CoverageProfile struct {
	// FileName indicates which file belongs to this coverage profile.
	FileName string
	// TotalLines indicates total lines of this coverage profile.
	TotalLines int
	// CoveredLines indicates covered lines of this coverage profile.
	CoveredLines int
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
