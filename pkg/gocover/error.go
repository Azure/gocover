package gocover

const (
	GeneralErrorExitCode        = 1  // bash general error exit code
	UnitTestFailedErrorExitCode = 11 // unit test failed exit code
	LowCoverageErrorExitCode    = 12 // pass rate is lower than the coverage baseline exit code
)

// GoCoverError carries the detail error information for gocover error
type GoCoverError struct {
	ExitCode   int
	Err        error
	ErrMessage string
}

func WrapErrorWithCode(err error, exitCode int, errMessage string) *GoCoverError {
	return &GoCoverError{
		ExitCode:   exitCode,
		Err:        err,
		ErrMessage: errMessage,
	}
}

func WrapError(err error, errMessage string) *GoCoverError {
	return WrapErrorWithCode(err, GeneralErrorExitCode, errMessage)
}

func (e *GoCoverError) Error() string {
	return e.Err.Error()
}
