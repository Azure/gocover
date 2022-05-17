package report

import (
	"github.com/Azure/gocover/pkg/gittool"
	"golang.org/x/tools/cover"
)

// NewDiffCoverageReport creates a diff coverage report instance.
// TODO: implement it
func NewDiffCoverageReport() DiffCoverageReport {
	return nil
}

// TODO: implement DiffCoverageReport interface
type DiffCoverageReport interface {
	// DiffCoverage calculate diff coverage
	DiffCoverage(profile []*cover.Profile, path gittool.Patch) error
	// ApplyFilter filters the cover profile with filters
	ApplyFilter(profile []*cover.Profile, filters []string) error
	// generate diff coverage report
	GenerateReport(profile []*cover.Profile) error
}
