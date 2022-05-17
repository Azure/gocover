package report

import (
	"github.com/Azure/gocover/pkg/gittool"
	"golang.org/x/tools/cover"
)

// NewDiffCoverageReport creates a diff coverage report instance.
// TODO: implement it
func NewDiffCoverageReport(
	filters []string,
) DiffCoverageReport {
	return nil
}

// TODO: implement DiffCoverageReport interface
type DiffCoverageReport interface {
	// DiffCoverage calculate diff coverage
	DiffCoverage(profile []*cover.Profile, path gittool.Patch) error
	// generate diff coverage report
	GenerateReport(profile []*cover.Profile) error
}
