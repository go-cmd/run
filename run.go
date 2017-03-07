// Package run provides strategies for running command with a uniform interface.
// This packages requires github.com/go-cmd/cmd.
package run

import (
	"errors"

	"github.com/go-cmd/cmd"
)

var (
	// ErrRunning is returned by a Runner if its Run method is called while still running.
	ErrRunning = errors.New("already running")

	// ErrStopped is returned by a Runner if its Stop method is called before Run finishes.
	ErrStopped = errors.New("Stop called")

	// ErrNonzeroExit is returned by a Runner if a Cmd returns a non-zero exit code.
	ErrNonzeroExit = errors.New("non-zero exit")
)

// A Runner runs a list of commands. The interface is intentionally trivial
// because the real benefit lies in its implementation. For example, RunSync
// runs commands synchronously in the order given without retries or timeouts.
// Another implementation could implement a more complex strategy, for example,
// running commands in parallel. The implementation details are hidden from and
// irrelevant to the caller, which allows the caller to focus on running commands,
// not how they are ran.
type Runner interface {
	Run([]cmd.Cmd) error
	Stop() error
	Status() []cmd.Status
}
