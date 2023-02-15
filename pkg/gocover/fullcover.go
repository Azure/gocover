package gocover

import (
	"context"
	"fmt"
	"go/build"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/parser"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
	"go.uber.org/multierr"
)

func NewFullCover(o *FullOption) (GoCover, error) {
	var (
		dbClient dbclient.DbClient
		err      error
	)

	logger := o.Logger
	if logger == nil {
		logger = logrus.New()
	}
	logger = logger.WithField("source", "fullcover")

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

	return &fullCover{
		coverFilenames:  o.CoverProfiles,
		modulePath:      modulePath,
		repositoryPath:  repositoryAbsPath,
		excludeFiles:    make(excludeFileCache),
		excludePatterns: o.Excludes,
		moduleDir:       o.ModuleDir,
		coverageTree:    report.NewCoverageTree(modulePath),
		logger:          logger,
		dbClient:        dbClient,
		reportGenerator: report.NewReportGenerator(o.Style, o.OutputDir, o.ReportName, o.Logger),
	}, nil

}

var _ GoCover = (*fullCover)(nil)

// diffCoverage implements the GoCover interface and generate the full coverage statistics.
type fullCover struct {
	coverFilenames  []string
	moduleDir       string
	modulePath      string
	repositoryPath  string
	excludePatterns []string
	ignoreProfiles  []*annotation.IgnoreProfile
	excludeFiles    excludeFileCache
	coverageTree    report.CoverageTree
	reportGenerator report.ReportGenerator
	dbClient        dbclient.DbClient

	logger logrus.FieldLogger
}

func (full *fullCover) Run(ctx context.Context) error {

	statistics, err := full.generateStatistics()
	if err != nil {
		return fmt.Errorf("full: %w", err)
	}

	if err := full.reportGenerator.GenerateReport(statistics); err != nil {
		return fmt.Errorf("generate report: %w", err)
	}

	if err := full.dump(ctx); err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (full *fullCover) dump(ctx context.Context) error {
	all := full.coverageTree.All()

	if full.dbClient != nil {
		err := storeCoverageData(ctx, full.dbClient, all, FullCoverage, full.modulePath)
		if err != nil {
			return fmt.Errorf("store coverage data: %w", err)
		}
		err = storeIgnoreProfileData(ctx, full.dbClient, full.ignoreProfiles, FullCoverage, full.modulePath, full.repositoryPath, full.moduleDir)
		if err != nil {
			return fmt.Errorf("store ignore profile data: %w", err)
		}
	}

	dump(all, full.logger)
	return nil
}

func (full *fullCover) generateStatistics() (*report.Statistics, error) {
	coverParser, err := parser.NewParser(full.coverFilenames, full.logger)
	if err != nil {
		return nil, err
	}
	packages, err := coverParser.Parse(nil)
	if err != nil {
		return nil, err
	}

	lock := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	errorChannel := make(chan error, len(packages))

	coverProfileChannel := make(chan []*report.CoverageProfile, len(packages))

	for _, pkg := range packages {
		wg.Add(1)

		go func(pkg *parser.Package) {
			defer wg.Done()

			m := make(map[string]*report.CoverageProfile)
			fileCache := make(fileContentsCache)

			full.logger.Debugf("package: %s", pkg.Name)
			full.ignoreProfiles = append(full.ignoreProfiles, pkg.IgnoreProfiles...)

			p, err := build.Import(pkg.Name, ".", build.FindOnly)
			if err != nil {
				errorChannel <- fmt.Errorf("build import %w", err)
				return
			}

			var coverProfiles []*report.CoverageProfile

			for _, fun := range pkg.Functions {

				if ok := inExclueds(
					full.excludeFiles,
					full.excludePatterns,
					formatFilePath(p.Root, fun.File, full.modulePath),
					full.logger,
				); ok {
					continue
				}

				// extract into single function
				coverProfile, ok := m[fun.File]
				if !ok {
					coverProfile = &report.CoverageProfile{
						FileName: formatFilePath(p.Root, fun.File, full.modulePath),
					}
					m[fun.File] = coverProfile
					coverProfiles = append(coverProfiles, coverProfile)
				}

				fileContents, err := findFileContents(fileCache, fun.File)
				if err != nil {
					errorChannel <- fmt.Errorf("find file contents: %w", err)
					return
				}

				section := &report.ViolationSection{
					StartLine: fun.StartLine,
					EndLine:   fun.EndLine,
				}

				for i := fun.StartLine; i <= fun.EndLine; i++ {
					section.Contents = append(section.Contents, fileContents[i-1])
				}

				lock.Lock()
				node := full.coverageTree.FindOrCreate(strings.TrimPrefix(fun.File, p.Root))

				var total, ignored, covered int
				violated := false
				for _, st := range fun.Statements {
					total += 1
					node.TotalLines += 1

					if st.Mode == parser.Ignore {
						full.logger.Debugf("%s ignore line %d", fun.File, st.StartLine)
						ignored++
						node.TotalIgnoredLines += 1
					}
					if st.Reached > 0 {
						node.TotalCoveredLines += 1
						covered++
					} else {
						section.ViolationLines = append(section.ViolationLines, st.StartLine)
						violated = true
					}

				}

				node.TotalEffectiveLines = node.TotalLines - node.TotalIgnoredLines
				lock.Unlock()

				coverProfile.TotalLines += total
				coverProfile.CoveredLines += covered
				coverProfile.TotalEffectiveLines += (total - ignored)
				coverProfile.TotalIgnoredLines += ignored
				coverProfile.TotalViolationLines = append(coverProfile.TotalViolationLines, section.ViolationLines...)
				if violated {
					coverProfile.ViolationSections = append(coverProfile.ViolationSections, section)
				}
			}

			errorChannel <- nil
			coverProfileChannel <- coverProfiles

		}(pkg)

	}

	wg.Wait()

	var finalErr error
	for i := 0; i < len(packages); i++ {
		finalErr = multierr.Append(finalErr, <-errorChannel)
	}

	if finalErr != nil {
		return nil, finalErr
	}

	statistics := &report.Statistics{
		StatisticsType: report.FullStatisticsType,
	}

	for i := 0; i < len(packages); i++ {
		statistics.CoverageProfile = append(statistics.CoverageProfile, <-coverProfileChannel...)
	}

	full.coverageTree.CollectCoverageData()

	reBuildStatistics(statistics, full.excludeFiles)

	return statistics, nil
}
