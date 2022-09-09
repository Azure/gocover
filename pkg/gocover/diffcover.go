package gocover

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"strings"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/gittool"
	"github.com/Azure/gocover/pkg/parser"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
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
	logger = logger.WithField("source", "diffcover")

	if o.DbOption.DataCollectionEnabled {
		dbClient, err = o.DbOption.GetDbClient(o.Logger)
		if err != nil {
			return nil, fmt.Errorf("get db client: %w", err)
		}
	}

	repositoryAbsPath, err := filepath.Abs(o.RepositoryPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path of repo: %w", err)
	}

	modulePath, err := parseGoModulePath(filepath.Join(repositoryAbsPath, o.ModuleDir))
	if err != nil {
		return nil, fmt.Errorf("parse go module path: %w", err)
	}

	logger.Debugf("repository path: %s, module path: %s, output dir: %s, exclude patterns: %s",
		repositoryAbsPath, modulePath, o.OutputDir, o.Excludes)

	return &diffCover{
		repositoryPath:   repositoryAbsPath,
		comparedBranch:   o.CompareBranch,
		moduleDir:        o.ModuleDir,
		modulePath:       modulePath,
		excludeFiles:     make(excludeFileCache),
		excludePatterns:  o.Excludes,
		coverageTree:     report.NewCoverageTree(modulePath),
		coverFilenames:   o.CoverProfiles,
		coverageBaseline: o.CoverageBaseline,
		dbClient:         dbClient,
		reportGenerator:  report.NewReportGenerator(o.Style, o.OutputDir, o.ReportName, o.Logger),
		logger:           logger,
	}, nil

}

var _ GoCover = (*diffCover)(nil)

// diffCoverage implements the GoCover interface and generate the diff coverage statistics.
type diffCover struct {
	comparedBranch   string // git diff base branch
	repositoryPath   string
	excludePatterns  []string
	ignoreProfiles   []*annotation.IgnoreProfile
	excludeFiles     excludeFileCache
	moduleDir        string
	modulePath       string
	coverFilenames   []string
	coverageBaseline float64

	reportGenerator report.ReportGenerator
	coverageTree    report.CoverageTree
	dbClient        dbclient.DbClient

	logger logrus.FieldLogger
}

func (diff *diffCover) Run(ctx context.Context) error {

	statistics, err := diff.generateStatistics()
	if err != nil {
		return fmt.Errorf("diff: %w", err)
	}

	if err := diff.reportGenerator.GenerateReport(statistics); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	if err := diff.dump(ctx); err != nil {
		return fmt.Errorf("%w", err)
	}

	if err := diff.pass(statistics); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (diff *diffCover) pass(statistics *report.Statistics) error {
	if statistics.TotalCoveragePercent < diff.coverageBaseline {
		return fmt.Errorf("the coverage baseline pass rate is %.2f, currently is %.2f",
			diff.coverageBaseline,
			statistics.TotalCoveragePercent,
		)
	}
	return nil
}

func (diff *diffCover) dump(ctx context.Context) error {
	all := diff.coverageTree.All()

	if diff.dbClient != nil {
		err := storeCoverageData(ctx, diff.dbClient, all, FullCoverage, diff.modulePath)
		if err != nil {
			return fmt.Errorf("store coverage data: %w", err)
		}
		err = storeIgnoreProfileData(ctx, diff.dbClient, diff.ignoreProfiles, FullCoverage, diff.modulePath, diff.repositoryPath, diff.moduleDir)
		if err != nil {
			return fmt.Errorf("store ignore profile data: %w", err)
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
		diff.ignoreProfiles = append(diff.ignoreProfiles, pkg.IgnoreProfiles...)

		p, err := build.Import(pkg.Name, ".", build.FindOnly)
		if err != nil {
			return nil, fmt.Errorf("build import %w", err)
		}

		for _, fun := range pkg.Functions {

			// extract into single function
			coverProfile, ok := m[fun.File]
			if !ok {
				coverProfile = &report.CoverageProfile{
					FileName: formatFilePath(p.Root, fun.File, diff.modulePath),
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

				if ok := inExclueds(
					diff.excludeFiles,
					diff.excludePatterns,
					formatFilePath(p.Root, fun.File, diff.modulePath),
					diff.logger,
				); ok {
					continue
				}

				coverProfile.TotalLines += total
				coverProfile.CoveredLines += covered
				coverProfile.TotalEffectiveLines += (total - ignored)
				coverProfile.TotalIgnoredLines += ignored
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

	reBuildStatistics(statistics, diff.excludeFiles)

	return statistics, nil
}
