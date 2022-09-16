package gocover

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoCoverError(t *testing.T) {
	assertion := assert.New(t)

	err := WrapError(assert.AnError, "execute failed")
	assertion.EqualErrorf(err, assert.AnError.Error(), "error string")
	assertion.Equalf(GeneralErrorExitCode, err.ExitCode, "general error exit code")
	assertion.Equalf("execute failed", err.ErrMessage, "error message")

	err = WrapErrorWithCode(assert.AnError, LowCoverageErrorExitCode, "coverage is too low")
	assertion.EqualErrorf(err, assert.AnError.Error(), "error string")
	assertion.Equalf(LowCoverageErrorExitCode, err.ExitCode, "general error exit code")
	assertion.Equalf("coverage is too low", err.ErrMessage, "error message")
}
