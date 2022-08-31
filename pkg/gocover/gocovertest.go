package gocover

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

func NewGoCoverTest(o *GoCoverTestOption) (*goCoverTest, error) {
	repositoryAbsPath, err := filepath.Abs(o.RepositoryPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path of repo: %w", err)
	}

	return &goCoverTest{
		repositoryPath: repositoryAbsPath,
		moduleDir:      o.ModuleDir,
		outputDir:      o.OutputDir,
		stdout:         o.StdOut,
		stderr:         o.StdErr,
		logger:         o.Logger.WithField("source", "GoCoverTest"),
		mode:           o.CoverageMode,
		option:         o,
	}, nil
}

type goCoverTest struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

func (t *goCoverTest) RunTests(ctx context.Context) error {
	if t.outputDir == "" {
		tmpDir, err := ioutil.TempDir("", "gocover")
		if err != nil {
			return err
		}
		t.outputDir = tmpDir
	}

	if err := os.MkdirAll(t.outputDir, fs.ModePerm); err != nil {
		return fmt.Errorf("create output directory %s: %w", t.outputDir, err)
	}

	logger := t.logger.WithField("moduledir", t.moduleDir)

	coverFile := filepath.Join(t.outputDir, "coverage.out")
	cmd := exec.Command(goCmd(), "test", "./...", "-coverprofile", coverFile, "-coverpkg=./...", "-v")
	cmd.Dir = filepath.Join(t.repositoryPath, t.moduleDir)
	cmd.Stdin = nil
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr

	logger.Infof("run unit test: 'go test ./... -coverprofile %s -coverpkg=./... -v'", coverFile)
	if err := cmd.Run(); err != nil {
		err = fmt.Errorf(`run unit test 'go test ./... -coverprofile %s -coverpkg=./... -v' failed: %w`, coverFile, err)
		t.logger.WithError(err).Error()
		return err
	}

	gocover, err := t.getGoCover([]string{coverFile})
	if err != nil {
		return err
	}

	logger.Info("run unit test succeeded")
	logger.Infof("cover profile: %s", coverFile)
	if err := gocover.Run(ctx); err != nil {
		err := fmt.Errorf("run gocover: %w", err)
		logger.WithError(err).Error()
		return err
	}
	return nil
}

func (t *goCoverTest) getGoCover(coverProfiles []string) (GoCover, error) {
	switch t.mode {
	case FullCoverage:
		return NewFullCover(&FullOption{
			CoverProfiles:    coverProfiles,
			RepositoryPath:   t.option.RepositoryPath,
			ModuleDir:        t.option.ModuleDir,
			CoverageBaseline: t.option.CoverageBaseline,
			ReportFormat:     t.option.ReportFormat,
			ReportName:       t.option.ReportName,
			OutputDir:        t.outputDir,
			Excludes:         t.option.Excludes,
			Style:            t.option.Style,
			DbOption:         t.option.DbOption,
			Logger:           t.logger,
		})
	case DiffCoverage:
		return NewDiffCover(&DiffOption{
			CoverProfiles:    coverProfiles,
			CompareBranch:    t.option.CompareBranch,
			RepositoryPath:   t.option.RepositoryPath,
			ModuleDir:        t.option.ModuleDir,
			ModulePath:       t.option.ModuleDir,
			CoverageBaseline: t.option.CoverageBaseline,
			ReportFormat:     t.option.ReportFormat,
			ReportName:       t.option.ReportName,
			OutputDir:        t.outputDir,
			Excludes:         t.option.Excludes,
			Style:            t.option.Style,
			DbOption:         t.option.DbOption,
			Logger:           t.logger,
		})
	default:
		return nil, ErrUnknownCoverageMode
	}
}

func goCmd() string {
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	path := filepath.Join(runtime.GOROOT(), "bin", "go"+exeSuffix)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return "go"
}
