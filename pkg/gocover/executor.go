package gocover

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	outCoverageProfile = "coverage.out"
)

type GoCoverTestExecutor interface {
	Run(ctx context.Context) error
}

func NewGoCoverTestExecutor(o *GoCoverTestOption) (GoCoverTestExecutor, error) {
	repositoryAbsPath, err := filepath.Abs(o.RepositoryPath)
	if err != nil {
		return nil, fmt.Errorf("get absolute path of repo: %w", err)
	}

	if o.OutputDir == "" {
		dir, err := createGoCoverTempDirectory()
		if err != nil {
			return nil, fmt.Errorf("create gocover temp directory: %w", err)
		}
		o.OutputDir = dir
	}

	switch o.ExecutorMode {
	case GoExecutor:
		return &goBuiltInTestExecutor{
			repositoryPath: repositoryAbsPath,
			flags:          o.GoFlags,
			moduleDir:      o.ModuleDir,
			outputDir:      o.OutputDir,
			executable:     goCmd(),
			stdout:         o.StdOut,
			stderr:         o.StdErr,
			logger:         o.Logger.WithField("source", "GoCoverTest"),
			mode:           o.CoverageMode,
			option:         o,
		}, nil
	case GinkgoExecutor:

		return &ginkgoTestExecutor{
			repositoryPath: repositoryAbsPath,
			flags:          o.GinkgoFlags,
			moduleDir:      o.ModuleDir,
			outputDir:      o.OutputDir,
			stdout:         o.StdOut,
			stderr:         o.StdErr,
			executable:     ginkgoCmd(),
			logger:         o.Logger.WithField("source", "GoCoverTest"),
			mode:           o.CoverageMode,
			option:         o,
		}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownExecutorMode, o.ExecutorMode)
	}
}

var _ GoCoverTestExecutor = (*goBuiltInTestExecutor)(nil)
var _ GoCoverTestExecutor = (*ginkgoTestExecutor)(nil)

type goBuiltInTestExecutor struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
	flags          []string
	executable     string
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

func (t *goBuiltInTestExecutor) Run(ctx context.Context) error {
	logger := t.logger.WithFields(
		logrus.Fields{
			"moduledir": t.moduleDir,
			"executor":  "go",
		},
	)

	coverFile := filepath.Join(t.outputDir, outCoverageProfile)
	testString := fmt.Sprintf("go test ./... -coverprofile %s -coverpkg=./... -v", coverFile)

	cmd := exec.Command(t.executable, "test", "./...", "-coverprofile", coverFile, "-coverpkg=./...", "-v")
	cmd.Dir = filepath.Join(t.repositoryPath, t.moduleDir)
	cmd.Stdin = nil
	cmd.Stdout = t.stdout
	cmd.Stderr = t.stderr

	logger.Infof("run unit tests: '%s'", testString)
	if err := cmd.Run(); err != nil {
		t.logger.WithError(err).Errorf(`run unit test '%s'`, testString)
		return fmt.Errorf("unit test failed: %w", err)
	}

	gocover, err := buildGoCover(t.mode, t.option, []string{coverFile}, logger)
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

type ginkgoTestExecutor struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
	flags          []string
	executable     string
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

func (e *ginkgoTestExecutor) Run(ctx context.Context) error {
	err := e.runTests(ctx)
	if err != nil {
		return err
	}

	coverFiles, err := findCoverProfiles(filepath.Join(e.repositoryPath, e.moduleDir))
	if err != nil {
		return err
	}

	e.logger.Debugf("total: %d", len(coverFiles))
	for _, f := range coverFiles {
		e.logger.Debugf("%s", f)
	}

	mergedFile, err := mergeCoverProfiles(e.outputDir, coverFiles)
	if err != nil {
		return fmt.Errorf("merge cover profiles: %w", err)
	}

	gocover, err := buildGoCover(e.mode, e.option, []string{mergedFile}, e.logger)
	if err != nil {
		return err
	}

	e.logger.Infof("cover profile: %s", mergedFile)
	if err := gocover.Run(ctx); err != nil {
		err := fmt.Errorf("run gocover: %w", err)
		e.logger.WithError(err).Error()
		return err
	}

	for _, f := range coverFiles {
		e.logger.Debugf("clean file: %s", f)
		_ = os.Remove(f)
	}

	return nil
}

func mergeCoverProfiles(outputdir string, coverProfiles []string) (string, error) {
	result := filepath.Join(outputdir, outCoverageProfile)
	f, err := os.Create(result)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fmt.Fprint(f, "mode: atomic\n")
	for _, c := range coverProfiles {
		pf, err := os.Open(c)
		if err != nil {
			return "", err
		}
		s := bufio.NewScanner(pf)

		s.Scan() // skip first line of cover profile because it's cover profile's metadata
		for s.Scan() {
			fmt.Fprintf(f, "%s\n", s.Text())
		}
		pf.Close()
	}

	return result, nil
}

func (executor *ginkgoTestExecutor) runTests(ctx context.Context) error {
	workingDir := filepath.Join(executor.repositoryPath, executor.moduleDir)
	logger := executor.logger.WithFields(logrus.Fields{
		"moduledir":  executor.moduleDir,
		"workingdir": workingDir,
		"executor":   "ginkgo",
	})

	buildArgs := []string{"build", "-r", "-cover", "-coverpkg=./...", "./"}
	buildString := fmt.Sprintf("%s %s", executor.executable, strings.Join(buildArgs, " "))

	logger.Infof("executing cmd: %s", buildString)
	buildCmd := exec.Command(executor.executable, buildArgs...)
	buildCmd.Dir = workingDir
	buildCmd.Stdin = nil
	buildCmd.Stdout = executor.stdout
	buildCmd.Stderr = executor.stderr
	if err := buildCmd.Run(); err != nil {
		logger.WithError(err).Errorf(`executing cmd %s`, buildString)
		return fmt.Errorf("build tests: %w", err)
	}
	logger.Infof("ginkgo tests built sucessfully")

	ginkgoFlags := []string{}
	for _, flag := range executor.flags {
		if trimmed := strings.TrimSpace(flag); trimmed != "" {
			ginkgoFlags = append(ginkgoFlags, trimmed)
		}
	}
	ginkgoFlags = append(ginkgoFlags, "./")
	runString := fmt.Sprintf("%s %s", executor.executable, strings.Join(ginkgoFlags, " "))

	logger.Infof("executing cmd: %s", runString)
	runCmd := exec.Command(executor.executable, ginkgoFlags...)
	runCmd.Dir = workingDir
	runCmd.Stdin = nil
	runCmd.Stdout = executor.stdout
	runCmd.Stderr = executor.stderr
	if err := runCmd.Run(); err != nil {
		logger.WithError(err).Errorf(`executing cmd %s`, runString)
		return fmt.Errorf("unit test failed: %w", err)
	}
	logger.Info("ginkgo tests run sucessfully")

	return nil
}

func findCoverProfiles(dir string) ([]string, error) {
	files, err := glob(dir, func(s string) bool {
		// ginkgo generates cover profile file ends with ".coverprofile.1" or ".coverprofile"
		return strings.HasSuffix(s, ".coverprofile.1") || strings.HasSuffix(s, ".coverprofile")
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func glob(root string, fn func(string) bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if fn(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
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
	var exeSuffix string
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	path := filepath.Join(os.Getenv("GOPATH"), "bin", "go"+exeSuffix)
	if _, err := os.Stat(path); err == nil {
		return path
	}
	return "ginkgo"
}

func buildGoCover(
	mode CoverageMode,
	option *GoCoverTestOption,
	coverProfiles []string,
	logger logrus.FieldLogger,
) (GoCover, error) {
	switch mode {
	case FullCoverage:
		return NewFullCover(&FullOption{
			CoverProfiles:    coverProfiles,
			RepositoryPath:   option.RepositoryPath,
			ModuleDir:        option.ModuleDir,
			CoverageBaseline: option.CoverageBaseline,
			ReportFormat:     option.ReportFormat,
			ReportName:       option.ReportName,
			OutputDir:        option.OutputDir,
			Excludes:         option.Excludes,
			Style:            option.Style,
			DbOption:         option.DbOption,
			Logger:           logger,
		})
	case DiffCoverage:
		return NewDiffCover(&DiffOption{
			CoverProfiles:    coverProfiles,
			CompareBranch:    option.CompareBranch,
			RepositoryPath:   option.RepositoryPath,
			ModuleDir:        option.ModuleDir,
			ModulePath:       option.ModuleDir,
			CoverageBaseline: option.CoverageBaseline,
			ReportFormat:     option.ReportFormat,
			ReportName:       option.ReportName,
			OutputDir:        option.OutputDir,
			Excludes:         option.Excludes,
			Style:            option.Style,
			DbOption:         option.DbOption,
			Logger:           logger,
		})
	default:
		return nil, ErrUnknownCoverageMode
	}
}
