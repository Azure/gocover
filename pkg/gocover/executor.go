package gocover

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	GinkgoEnabledEnvKey = "GINKGO_MODE"
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
		tmpDir, err := ioutil.TempDir("", "gocover")
		if err != nil {
			return nil, err
		}
		o.OutputDir = tmpDir
	}

	if err := os.MkdirAll(o.OutputDir, fs.ModePerm); err != nil {
		return nil, fmt.Errorf("create output directory %s: %w", o.OutputDir, err)
	}

	if os.Getenv(GinkgoEnabledEnvKey) != "" {
		return &ginkgoTestExecutor{
			repositoryPath: repositoryAbsPath,
			moduleDir:      o.ModuleDir,
			outputDir:      o.OutputDir,
			stdout:         o.StdOut,
			stderr:         o.StdErr,
			executable:     ginkgoCmd(),
			logger:         o.Logger.WithField("source", "GoCoverTest"),
			mode:           o.CoverageMode,
			option:         o,
		}, nil
	} else {
		return &goBuiltInTestExecutor{
			repositoryPath: repositoryAbsPath,
			moduleDir:      o.ModuleDir,
			outputDir:      o.OutputDir,
			executable:     goCmd(),
			stdout:         o.StdOut,
			stderr:         o.StdErr,
			logger:         o.Logger.WithField("source", "GoCoverTest"),
			mode:           o.CoverageMode,
			option:         o,
		}, nil
	}
}

var _ GoCoverTestExecutor = (*goBuiltInTestExecutor)(nil)
var _ GoCoverTestExecutor = (*ginkgoTestExecutor)(nil)

type goBuiltInTestExecutor struct {
	repositoryPath string
	moduleDir      string
	mode           CoverageMode
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

	coverFile := filepath.Join(t.outputDir, "coverage.out")
	cmd := exec.Command(t.executable, "test", "./...", "-coverprofile", coverFile, "-coverpkg=./...", "-v")
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
	executable     string
	outputDir      string
	option         *GoCoverTestOption
	stdout         io.Writer
	stderr         io.Writer
	logger         logrus.FieldLogger
}

func (e *ginkgoTestExecutor) Run(ctx context.Context) error {
	coverFiles, err := e.runTests(ctx)
	if err != nil {
		return err
	}

	one := filepath.Join(e.outputDir, "cover.out")
	f, err := os.Create(one)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, "mode: atomic\n")

	for _, c := range coverFiles {
		pf, err := os.Open(c)
		if err != nil {
			panic(err)
		}
		s := bufio.NewScanner(pf)
		s.Scan()
		for s.Scan() {
			fmt.Fprintf(f, "%s\n", s.Text())
		}
		pf.Close()
	}
	f.Close()

	gocover, err := buildGoCover(e.mode, e.option, []string{one}, e.logger)
	if err != nil {
		return err
	}

	e.logger.Info("run unit test succeeded")
	e.logger.Infof("cover profile: %s", coverFiles)
	if err := gocover.Run(ctx); err != nil {
		err := fmt.Errorf("run gocover: %w", err)
		e.logger.WithError(err).Error()
		return err
	}

	for _, f := range coverFiles {
		_, name := filepath.Split(f)
		e.logger.Debugf("move %s to %s", f, filepath.Join(e.outputDir, name))
		if err := os.Rename(f, filepath.Join(e.outputDir, name)); err != nil {
			e.logger.Error(err)
		}
	}

	return nil
}

func (executor *ginkgoTestExecutor) runTests(ctx context.Context) ([]string, error) {
	logger := executor.logger.WithFields(
		logrus.Fields{
			"moduledir": executor.moduleDir,
			"executor":  "ginkgo",
		},
	)

	logger.Debugf("executing cmd: %s build -r -cover -coverpkg ./... ./", executor.executable)
	buildCmd := exec.Command(executor.executable, "build", "-r", "-cover", "-coverpkg", "./...", "./")
	buildCmd.Dir = filepath.Join(executor.repositoryPath, executor.moduleDir)
	buildCmd.Stdin = nil
	buildCmd.Stdout = executor.stdout
	buildCmd.Stderr = executor.stderr
	if err := buildCmd.Run(); err != nil {
		err = fmt.Errorf(`executing cmd '%s build -r -cover -coverpkg ./... ./' failed: %w`, executor.executable, err)
		logger.WithError(err).Error()
		return nil, err
	}
	logger.Debug("tests built sucessfully")

	logger.Debugf("executing cmd: %s -p -r -trace -cover -coverpkg ./... ./", executor.executable)
	runCmd := exec.Command(executor.executable, "-p", "-r", "-trace", "-cover", "-coverpkg", "./...", "./")
	runCmd.Dir = filepath.Join(executor.repositoryPath, executor.moduleDir)
	runCmd.Stdin = nil
	runCmd.Stdout = executor.stdout
	runCmd.Stderr = executor.stderr
	if err := runCmd.Run(); err != nil {
		err = fmt.Errorf(`executing cmd '%s -r -trace -cover -coverpkg ./... ./' failed: %w`, executor.executable, err)
		logger.WithError(err).Error()
		return nil, err
	}

	files, err := glob(filepath.Join(executor.repositoryPath, executor.moduleDir), func(s string) bool {
		return strings.HasSuffix(s, ".coverprofile.1") || strings.HasSuffix(s, ".coverprofile")
	})
	if err != nil {
		return nil, err
	}

	logger.Debugf("total: %d", len(files))
	for _, f := range files {
		logger.Debugf("%s", f)
	}

	return files, nil
}

func glob(root string, fn func(string) bool) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if fn(s) {
			files = append(files, s)
		}
		return nil
	})
	return files, err
}

// func glob(dir string, ext string) ([]string, error) {
// 	files := []string{}
// 	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
// 		fmt.Println(path, f.Name())
// 		if strings.HasSuffix(path, ext) {
// 			files = append(files, path)
// 		}
// 		return nil
// 	})

// 	return files, err
// }

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
