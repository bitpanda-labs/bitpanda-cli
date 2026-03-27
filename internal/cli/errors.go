package cli

import "fmt"

// ExitError wraps an error with a specific process exit code.
// This allows Execute() to signal which exit code main should use
// without calling os.Exit() directly.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// newExitError creates an ExitError with the given code and underlying error.
func newExitError(code int, err error) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf("%w", err)}
}
