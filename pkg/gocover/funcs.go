package gocover

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/gocover/pkg/annotation"
	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
)

const (
	DefaultReportFormat     = "html"
	DefaultCompareBranch    = "origin/master"
	DefaultCoverageBaseline = 80.0
)

// excludeFileCache cache contains exclude file
type excludeFileCache map[string]bool

func inExclueds(cache excludeFileCache, excludesPattern []string, fileName string, logger logrus.FieldLogger) bool {
	if _, ok := cache[fileName]; ok {
		return true
	}

	for _, pattern := range excludesPattern {
		match, err := doublestar.PathMatch(pattern, fileName)
		if err != nil {
			logger.Warnf("path match [%s, %s] %w", pattern, fileName, err)
			continue
		}
		if match {
			cache[fileName] = true
			logger.Debugf("exculde file: [%s, %s]", fileName, pattern)
			return true
		}
	}
	return false
}

// calculateCoverage calculate coverage proportion
func calculateCoverage(covered int64, effectived int64) float64 {
	if effectived == 0 {
		return 100.0
	}
	return float64(covered) / float64(effectived) * 100
}

// reBuildStatistics rebuild fields of Statistics from its CoverageProfile
func reBuildStatistics(s *report.Statistics, cache excludeFileCache) {
	for _, p := range s.CoverageProfile {
		s.TotalLines += p.TotalLines
		s.TotalEffectiveLines += p.TotalEffectiveLines
		s.TotalIgnoredLines += p.TotalIgnoredLines
		s.TotalCoveredLines += p.CoveredLines
		s.TotalCoveredButIgnoredLines += p.CoveredButIgnoredLines
	}

	s.TotalCoveragePercent = calculateCoverage(
		int64(s.TotalCoveredLines-s.TotalCoveredButIgnoredLines),
		int64(s.TotalEffectiveLines),
	)
	s.TotalCoverageWithoutIgnore = calculateCoverage(
		int64(s.TotalCoveredLines),
		int64(s.TotalLines),
	)

	for f := range cache {
		s.ExcludeFiles = append(s.ExcludeFiles, f)
	}
}

// formatFilePath format filename that strip root path and adds module path
// fileNamePath is the absolute path of the file, modulePath is the module path of go module
// for example:
//
//	rootRepoPath: /home/User/go/src/Azure/gocover
//	fileNamePath: /home/User/go/src/Azure/gocover/pkg/foo/foo.go
//	modulePath: github.com/Azure/gocover
//
// it returns github.com/Azure/gocover/foo/foo.go
func formatFilePath(rootRepoPath, fileNamePath, modulePath string) string {
	return filepath.Join(modulePath,
		strings.TrimPrefix(fileNamePath, rootRepoPath),
	)
}

// storeCoverageData send all coverage results to db store
func storeCoverageData(ctx context.Context, dbClient dbclient.DbClient, all []*report.AllInformation, coverageMode CoverageMode, modulePath string) error {
	now := time.Now().UTC()

	var data []*dbclient.CoverageData
	for _, info := range all {
		d := &dbclient.CoverageData{
			PreciseTimestamp:       now,
			TotalLines:             info.TotalLines,
			EffectiveLines:         info.TotalEffectiveLines,
			IgnoredLines:           info.TotalIgnoredLines,
			CoveredLines:           info.TotalCoveredLines,
			CoveredButIgnoredLines: info.TotalCoveredButIgnoreLines,
			ModulePath:             modulePath,
			FilePath:               info.Path,
			Coverage:               calculateCoverage(info.TotalCoveredLines, info.TotalLines),
			CoverageWithIgnored:    calculateCoverage(info.TotalCoveredLines-info.TotalCoveredButIgnoreLines, info.TotalEffectiveLines),
			CoverageMode:           string(coverageMode),
		}
		data = append(data, d)
	}

	return dbClient.StoreCoverageDataFromFile(ctx, data)
}

func storeIgnoreProfileData(ctx context.Context, dbClient dbclient.DbClient, ignoreProfiles []*annotation.IgnoreProfile, coverageMode CoverageMode, modulePath string, repositoryPath string, moduleDir string) error {
	now := time.Now().UTC()

	var data []*dbclient.IgnoreProfileData
	for _, profile := range ignoreProfiles {
		formattedFilePath := filepath.Join(modulePath, strings.TrimPrefix(profile.Filename, filepath.Join(repositoryPath, moduleDir)))
		if profile.Type == annotation.FILE_IGNORE {

			d := &dbclient.IgnoreProfileData{
				PreciseTimestamp: now,
				FilePath:         formattedFilePath,
				ModulePath:       modulePath,
				IgnoreType:       string(profile.Type),
				Comments:         profile.Comments,
				Annotation:       profile.Annotation,
			}
			data = append(data, d)

			continue
		}

		for _, block := range profile.IgnoreBlocks {
			d := &dbclient.IgnoreProfileData{
				PreciseTimestamp: now,
				FilePath:         formattedFilePath,
				ModulePath:       modulePath,
				IgnoreType:       string(profile.Type),
				LineNumber:       block.AnnotationLineNumber,
				StartLine:        block.Lines[0],
				EndLine:          block.Lines[len(block.Lines)-1],
				Comments:         block.Comments,
				Annotation:       block.Annotation,
				Contents:         strings.Join(block.Contents, "\n"),
			}
			data = append(data, d)
		}
	}

	return dbClient.StoreIgnoreProfileDataFromFile(ctx, data)
}

// dump outputs all coverage results
func dump(all []*report.AllInformation, logger logrus.FieldLogger) {
	logger.Debug("Summary of coverage:")
	logger.Debug("Path EffectiveLines CoveredLines IgnoredLines TotalLines CoveredButIgnoredLines Coverage CoverageWithIgnorance")

	for _, info := range all {
		logger.Debugf("%s %d %d %d %d %d %.1f%% %.1f%%",
			info.Path,
			info.TotalEffectiveLines,
			info.TotalCoveredLines,
			info.TotalIgnoredLines,
			info.TotalLines,
			info.TotalCoveredButIgnoreLines,
			calculateCoverage(info.TotalCoveredLines, info.TotalLines),
			calculateCoverage(info.TotalCoveredLines-info.TotalCoveredButIgnoreLines, info.TotalEffectiveLines),
		)
	}
}

type fileContentsCache map[string][]string

// findFileContents finds the contents of specific file. filename is the absolute path of the file.
func findFileContents(fileCache fileContentsCache, filename string) ([]string, error) {
	result, ok := fileCache[filename]
	if !ok {
		fd, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		s := bufio.NewScanner(fd)
		for s.Scan() {
			result = append(result, s.Text())
		}
		fileCache[filename] = result
	}
	return result, nil
}

var (
	ErrModuleNotFound = errors.New("cannot find module path")
)

// parseGoModulePath uses modfile package to parse go module path
func parseGoModulePath(goModDir string) (string, error) {
	goModFilename := filepath.Join(goModDir, "go.mod")
	bs, err := os.ReadFile(goModFilename)
	if err != nil {
		return "", err
	}

	result := modfile.ModulePath(bs)
	if result == "" {
		return "", fmt.Errorf("%w: %s", ErrModuleNotFound, goModFilename)
	}

	return result, nil
}

// createGoCoverTempDirectory creates temp directory that used for gocover outputs
func createGoCoverTempDirectory() (string, error) {
	tmpDir, err := os.MkdirTemp("", "gocover")
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(tmpDir, fs.ModePerm); err != nil {
		return "", err
	}
	return tmpDir, nil
}
