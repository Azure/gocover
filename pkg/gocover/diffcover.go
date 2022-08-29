package gocover

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gittool"
	"github.com/Azure/gocover/pkg/parser"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
	"golang.org/x/tools/cover"
)

func NewDiffCover(o *DiffOption) (GoCover, error) {
	var (
		dbClient dbclient.DbClient
		err      error
	)

	logger := o.Logger
	if logger == nil {
		logger = logrus.New()
	}

	if o.DbOption.DataCollectionEnabled {
		dbClient, err = o.DbOption.GetDbClient(o.Logger)
		if err != nil {
			return nil, fmt.Errorf("get db client: %w", err)
		}
	}

	p, err := filepath.Abs(o.RepositoryPath)
	if err != nil {
		return nil, err
	}

	modulePath, err := parseGoModulePath(filepath.Join(p, o.ModuleDir))
	if err != nil {
		return nil, fmt.Errorf("parse go module path: %w", err)
	}

	var excludesRegexps []*regexp.Regexp
	for _, ignorePattern := range o.Excludes {
		reg, err := regexp.Compile(ignorePattern)
		if err != nil {
			return nil, fmt.Errorf("compile ignore pattern %s: %w", ignorePattern, err)
		}
		excludesRegexps = append(excludesRegexps, reg)
	}

	return &diffCover{
		comparedBranch:  o.CompareBranch,
		moduleDir:       o.ModuleDir,
		modulePath:      modulePath,
		excludesRegexps: excludesRegexps,
		coverageTree:    report.NewCoverageTree(modulePath),
		repositoryPath:  o.RepositoryPath,
		coverFilenames:  o.CoverProfiles,
		dbClient:        dbClient,
		reportGenerator: report.NewReportGenerator(o.Style, o.Output, o.ReportName, o.Logger),
		logger:          logger.WithField("source", "diffcover"),
	}, nil

}

var _ GoCover = (*diffCover)(nil)

// diffCoverage implements the DiffCoverage interface
// and generate the diff coverage statistics
type diffCover struct {
	comparedBranch  string            // git diff base branch
	profiles        []*cover.Profile  // go unit test coverage profiles
	changes         []*gittool.Change // diff change between compared branch and HEAD commit
	excludesRegexps []*regexp.Regexp  // excludes files regexp patterns
	repositoryPath  string
	ignoreProfiles  map[string]*annotation.IgnoreProfile
	coverProfiles   map[string]*cover.Profile
	moduleDir       string
	modulePath      string
	coverFilenames  []string

	reportGenerator report.ReportGenerator
	coverageTree    report.CoverageTree
	dbClient        dbclient.DbClient

	logger logrus.FieldLogger
}

func (diff *diffCover) Run(ctx context.Context) error {

	statistics, err := diff.generateStatistics()
	if err != nil {
		return fmt.Errorf("full: %s", err)
	}

	if err := diff.reportGenerator.GenerateReport(statistics); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	if err := diff.dump(ctx); err != nil {
		return fmt.Errorf(" %w", err)
	}

	return nil
}

func (diff *diffCover) dump(ctx context.Context) error {
	all := diff.coverageTree.All()

	if diff.dbClient != nil {
		err := store(ctx, diff.dbClient, all, FullCoverage, diff.moduleDir)
		if err != nil {
			return fmt.Errorf("store data: %w", err)
		}
	}

	dump(all, diff.logger)
	return nil
}

func (diff *diffCover) getGitChanges() ([]*gittool.Change, error) {
	gitClient, err := gittool.NewGitClient(diff.repositoryPath)
	if err != nil {
		return nil, fmt.Errorf("git repository: %w", err)
	}
	changes, err := gitClient.DiffChangesFromCommitted(diff.comparedBranch)
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	return changes, nil
}

func (diff *diffCover) generateStatistics() (*report.Statistics, error) {
	changes, err := diff.getGitChanges()
	if err != nil {
		return nil, err
	}

	packages, err := parser.NewParser(diff.coverFilenames, diff.logger).Parse(changes)
	if err != nil {
		return nil, err
	}

	statistics := &report.Statistics{
		StatisticsType: report.DiffStatisticsType,
		ComparedBranch: diff.comparedBranch,
	}
	m := make(map[string]*report.CoverageProfile)
	fileCache := make(fileContentsCache)
	added := make(map[string]*report.CoverageProfile)
	keep := make(map[string]string)
	for _, pkg := range *packages {
		diff.logger.Debugf("package: %s", pkg.Name)

		p, err := build.Import(pkg.Name, ".", build.FindOnly)
		if err != nil {
			return nil, fmt.Errorf("build import %w", err)
		}

		for _, fun := range pkg.Functions {

			// extract into single function
			coverProfile, ok := m[fun.File]
			if !ok {
				coverProfile = &report.CoverageProfile{
					FileName: filepath.Join(diff.modulePath, strings.TrimPrefix(fun.File, p.Root)),
				}
				m[fun.File] = coverProfile
			}

			fileContents, err := findFileContents(fileCache, fun.File)
			if err != nil {
				return nil, fmt.Errorf("find file contents: %w", err)
			}

			section := &report.ViolationSection{
				StartLine: fun.StartLine,
				EndLine:   fun.EndLine,
			}

			for i := fun.StartLine; i <= fun.EndLine; i++ {
				section.Contents = append(section.Contents, fileContents[i-1])
			}

			var total, ignored, covered int
			violated := false
			changed := false
			for _, st := range fun.Statements {
				if st.State == parser.Original {
					continue
				}

				changed = true
				total += 1

				if st.Mode == parser.Ignore {
					diff.logger.Debugf("%s ignore line %d", fun.File, st.StartLine)
					ignored++
					continue
				}
				if st.Reached > 0 {
					covered++
				} else {
					section.ViolationLines = append(section.ViolationLines, st.StartLine)
					violated = true
				}

			}

			if changed {
				coverProfile.TotalLines += total
				coverProfile.CoveredLines += covered
				coverProfile.TotalEffectiveLines += (total - ignored)
				coverProfile.TotalIgnoredLines += ignored
				// coverProfile.TotalViolationLines = append(coverProfile.TotalViolationLines, section.ViolationLines...)
				if violated {
					coverProfile.ViolationSections = append(coverProfile.ViolationSections, section)
				}
				if _, ok := added[fun.File]; !ok {
					statistics.CoverageProfile = append(statistics.CoverageProfile, coverProfile)
					added[fun.File] = coverProfile
					keep[fun.File] = p.Root
				}
			}
		}

	}

	for k, v := range added {
		node := diff.coverageTree.FindOrCreate(strings.TrimPrefix(k, keep[k]))
		node.TotalLines = int64(v.TotalLines)
		node.TotalCoveredLines = int64(v.CoveredLines)
		node.TotalEffectiveLines = int64(v.TotalEffectiveLines)
		node.TotalIgnoredLines = int64(v.TotalIgnoredLines)
	}

	diff.coverageTree.CollectCoverageData()

	for _, p := range statistics.CoverageProfile {
		statistics.TotalLines += p.TotalLines
		statistics.TotalCoveredLines += p.CoveredLines
		statistics.TotalEffectiveLines += p.TotalEffectiveLines
		statistics.TotalIgnoredLines += p.TotalIgnoredLines
	}

	statistics.TotalCoveragePercent = calculateCoverage(
		int64(statistics.TotalCoveredLines),
		int64(statistics.TotalEffectiveLines),
	)

	return statistics, nil
}
