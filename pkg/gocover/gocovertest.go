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

const (
	GinkgoEnabledEnvKey = "GINKGO_MODE"
)

func NewGoCoverTest(o *GoCoverTestOption) (GoCoverTestExecutor, error) {
	repositoryAbsPath, err := filepath.Abs(o.RepositoryPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path of repo: %w", err)
	}

	if os.Getenv(GinkgoEnabledEnvKey) != "" {
		return &ginkgoTestExecutor{
			repositoryPath: repositoryAbsPath,
			moduleDir:      o.ModuleDir,
			outputDir:      o.OutputDir,
			stdout:         o.StdOut,
			stderr:         o.StdErr,
			logger:         o.Logger.WithField("source", "GoCoverTest"),
			mode:           o.CoverageMode,
			option:         o,
		}, nil
	} else {
		return &goBuiltInTestExecutor{
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
}

type GoCoverTestExecutor interface {
	Run(ctx context.Context) error
}

var _ GoCoverTestExecutor = (*goBuiltInTestExecutor)(nil)
var _ GoCoverTestExecutor = (*ginkgoTestExecutor)(nil)

type goBuiltInTestExecutor struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

type ginkgoTestExecutor struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

func (t *goBuiltInTestExecutor) Run(ctx context.Context) error {
	if t.outputDir == "" {
		tmpDir, err := ioutil.TempDir("", "gocover")
		if err != nil {
			return err
		}
		t.outputDir = tmpDir
	}

	f, err := filepath.Abs(t.outputDir)
	if err != nil {
		return err
	}
	t.outputDir = f

	if err := os.MkdirAll(t.outputDir, fs.ModePerm); err != nil {
		return fmt.Errorf("create output directory %s: %w", t.outputDir, err)
	}

	logger := t.logger.WithFields(
		logrus.Fields{
			"moduledir": t.moduleDir,
			"mode":      "go",
		},
	)

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

func (executor *ginkgoTestExecutor) Run(ctx context.Context) error {
	coverFiles, err := executor.runTests(ctx)
	if err != nil {
		return err
	}

	gocover, err := executor.getGoCover(coverFiles)
	if err != nil {
		return err
	}

	executor.logger.Info("run unit test succeeded")
	executor.logger.Infof("cover profile: %s", coverFiles)
	if err := gocover.Run(ctx); err != nil {
		err := fmt.Errorf("run gocover: %w", err)
		executor.logger.WithError(err).Error()
		return err
	}

	for _, f := range coverFiles {
		_, name := filepath.Split(f)
		executor.logger.Debugf("move %s to %s", f, filepath.Join(executor.outputDir, name))
		if err := os.Rename(f, filepath.Join(executor.outputDir, name)); err != nil {
			executor.logger.Error(err)
		}
	}

	return nil
}

func (executor *ginkgoTestExecutor) runTests(ctx context.Context) ([]string, error) {
	if executor.outputDir == "" {
		tmpDir, err := ioutil.TempDir("", "gocover")
		if err != nil {
			return nil, err
		}
		executor.outputDir = tmpDir
	}

	if err := os.MkdirAll(executor.outputDir, fs.ModePerm); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", executor.outputDir, err)
	}

	logger := executor.logger.WithFields(
		logrus.Fields{
			"moduledir": executor.moduleDir,
			"mode":      "ginkgo",
		},
	)

	logger.Debugf("executing cmd: %s build -r -cover -coverpkg ./... ./", ginkgoCmd())
	buildCmd := exec.Command(ginkgoCmd(), "build", "-r", "-cover", "-coverpkg", "./...", "./")
	buildCmd.Dir = filepath.Join(executor.repositoryPath, executor.moduleDir)
	buildCmd.Stdin = nil
	buildCmd.Stdout = executor.stdout
	buildCmd.Stderr = executor.stderr
	if err := buildCmd.Run(); err != nil {
		err = fmt.Errorf(`executing cmd '%s build -r -cover -coverpkg ./... ./' failed: %w`, ginkgoCmd(), err)
		logger.WithError(err).Error()
		return nil, err
	}
	logger.Debug("tests built sucessfully")

	logger.Debugf("executing cmd: %s run -r -trace -cover -coverpkg ./... ./", ginkgoCmd())
	runCmd := exec.Command(ginkgoCmd(), "-r", "-trace", "-cover", "-coverpkg", "./...", "./")
	runCmd.Dir = filepath.Join(executor.repositoryPath, executor.moduleDir)
	runCmd.Stdin = nil
	runCmd.Stdout = executor.stdout
	runCmd.Stderr = executor.stderr
	if err := runCmd.Run(); err != nil {
		err = fmt.Errorf(`executing cmd '%s -r -trace -cover -coverpkg ./... ./' failed: %w`, ginkgoCmd(), err)
		logger.WithError(err).Error()
		return nil, err
	}

	files, err := glob(filepath.Join(executor.repositoryPath, executor.moduleDir), ".coverprofile")
	if err != nil {
		return nil, err
	}

	logger.Debugf("total: %d", len(files))

	for _, f := range files {
		logger.Debugf("%s", f)
	}

	return files, nil
}

func glob(dir string, ext string) ([]string, error) {
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func (t *ginkgoTestExecutor) getGoCover(coverProfiles []string) (GoCover, error) {
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

func (t *goBuiltInTestExecutor) getGoCover(coverProfiles []string) (GoCover, error) {
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

func ginkgoCmd() string {
	if path, err := exec.LookPath("ginkgo"); err == nil {
		return path
	}
	return "ginkgo"
}
