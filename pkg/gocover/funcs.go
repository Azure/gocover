package gocover

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/Azure/gocover/pkg/dbclient"
	"github.com/Azure/gocover/pkg/report"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
)

const (
	DefaultReportFormat     = "html"
	DefaultCompareBranch    = "origin/master"
	DefaultCoverageBaseline = 80.0
)

// calculateCoverage calculate coverage proportion
func calculateCoverage(covered int64, effectived int64) float64 {
	if effectived == 0 {
		return 100.0
	}
	return float64(covered) / float64(effectived) * 100
}

// store send all coverage results to db store
func store(ctx context.Context, dbClient dbclient.DbClient, all []*report.AllInformation, coverageMode CoverageMode, modulePath string) error {
	now := time.Now().UTC()
	for _, info := range all {
		err := dbClient.Store(ctx, &dbclient.Data{
			PreciseTimestamp: now,
			TotalLines:       info.TotalLines,
			EffectiveLines:   info.TotalEffectiveLines,
			IgnoredLines:     info.TotalIgnoredLines,
			CoveredLines:     info.TotalCoveredLines,
			ModulePath:       modulePath,
			FilePath:         info.Path,
			Coverage:         calculateCoverage(info.TotalCoveredLines, info.TotalEffectiveLines),
			CoverageMode:     string(coverageMode),
		})
		if err != nil {
			return fmt.Errorf("store data: %w", err)
		}
	}

	return nil
}

// dump outputs all coverage results
func dump(all []*report.AllInformation, logger logrus.FieldLogger) {
	logger.Debug("Summary of coverage:")

	for _, info := range all {
		logger.Debugf("%s %d %d %d %d %.1f%%",
			info.Path,
			info.TotalEffectiveLines,
			info.TotalCoveredLines,
			info.TotalIgnoredLines,
			info.TotalLines,
			calculateCoverage(info.TotalCoveredLines, info.TotalEffectiveLines),
		)
	}
}

type fileContentsCache map[string][]string

// findFileContents finds the contents of specific file.
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
	bs, err := ioutil.ReadFile(goModFilename)
	if err != nil {
		return "", err
	}

	result := modfile.ModulePath(bs)
	if result == "" {
		return "", fmt.Errorf("%w: %s", ErrModuleNotFound, goModFilename)
	}

	return result, nil
}
