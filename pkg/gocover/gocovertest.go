package gocover

import (
	"context"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

func NewGoCoverTest(o *GoCoverTestOption) *goCoverTest {
	return &goCoverTest{
		moduleDir: filepath.Join(o.RepositoryPath, o.ModuleDir),
		outputDir: o.Output,
		stdout:    o.StdOut,
		stderr:    o.StdErr,
		logger:    o.Logger,
		mode:      o.CoverageMode,
		option:    o,
	}
}

type goCoverTest struct {
	moduleDir string
	mode      CoverageMode
	outputDir string
	option    *GoCoverTestOption
	stdout    io.Writer
	stderr    io.Writer
	logger    logrus.FieldLogger
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
		return err
	}
	coverFile := filepath.Join(t.outputDir, "coverage.out")
	cmd := exec.Command(goCmd(), "test", "./...", "-coverprofile", coverFile, "-coverpkg=./...", "-v")
	cmd.Dir = t.moduleDir
	cmd.Stdin = nil
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	t.logger.Debugf("output: %s\n", coverFile)

	gocover, err := t.getGoCover([]string{coverFile})
	if err != nil {
		return err
	}

	return gocover.Run(ctx)
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
			Output:           t.option.Output,
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
			Output:           t.option.Output,
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
