package gocover

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockGoCover implements GoCover interface for testing
type mockGoCover struct {
	runErr error
}

func (m *mockGoCover) Run(ctx context.Context) error {
	return m.runErr
}

// Helper to patch exec.Command for testing
var execCommand = exec.Command

func TestGoBuiltInTestExecutor_Run_Success(t *testing.T) {
	// Patch exec.Command to simulate success
	oldCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "GO_HELPER_EXIT_CODE=0")
		return cmd
	}
	defer func() { execCommand = oldCommand }()

	var buf bytes.Buffer
	var logBuf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logBuf)

	executor := &goBuiltInTestExecutor{
		repositoryPath: "/tmp",
		moduleDir:      "",
		mode:           FullCoverage,
		flags:          []string{"-count=1"},
		executable:     "go",
		outputDir:      "/tmp",
		option:         &GoCoverTestOption{},
		stdout:         &buf,
		stderr:         &buf,
		logger:         logger,
	}

	executor.Run(context.Background())
	a := logBuf.String()
	assert.Contains(t, a, "go test ./... -count=1 -coverprofile /tmp/coverage.out -coverpkg=./... -v")
}

func TestGoBuiltInTestExecutor_Run_CommandFails(t *testing.T) {
	oldCommand := execCommand
	execCommand = func(name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "GO_HELPER_EXIT_CODE=1")
		return cmd
	}
	defer func() { execCommand = oldCommand }()

	var buf bytes.Buffer
	executor := &goBuiltInTestExecutor{
		repositoryPath: "/tmp",
		moduleDir:      "",
		mode:           FullCoverage,
		flags:          []string{},
		executable:     "go",
		outputDir:      "/tmp",
		option:         &GoCoverTestOption{},
		stdout:         &buf,
		stderr:         &buf,
		logger:         logrus.New(),
	}

	err := executor.Run(context.Background())
	assert.Error(t, err)
}

// TestHelperProcess is not a real test. It's used as a helper process for exec.Command patching.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Use GO_HELPER_EXIT_CODE to control exit code for all test scenarios
	code := 0
	if v := os.Getenv("GO_HELPER_EXIT_CODE"); v != "" {
		// ignore error, default to 0
		fmt.Sscanf(v, "%d", &code)
	}
	os.Exit(code)
}
