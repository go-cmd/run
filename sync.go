package run

import (
	"sync"

	"github.com/go-cmd/cmd"
)

// RunSync is a Runner that runs commands synchronously in the order given.
// No timeouts or retries are used. It's the simplest possible Runner which
// expects all commands to be quick and reliable.
type RunSync struct {
	stopOnError bool
	// --
	*sync.Mutex
	running bool
	cmd     *cmd.Cmd // current running
	cur     int      // in cmds if proc != nil
	status  []cmd.Status

	stopChan chan struct{}
}

// NewRunSync creates a new RunSync. If stopOnError is true, Run stops immediately
// and returns ErrNonzeroExit when a command exits non-zero.
func NewRunSync(stopOnError bool) *RunSync {
	return &RunSync{
		stopOnError: stopOnError,
		// --
		Mutex: &sync.Mutex{},
	}
}

// Run runs the list of Cmd and waits for them to complete, or one of them
// to exit non-zero if the runner was created with stopOnError = true.
func (r *RunSync) Run(cmds []cmd.Cmd) error {
	r.Lock()
	if r.running {
		r.Unlock()
		return ErrRunning
	}

	// Init status for each cmd
	r.status = make([]cmd.Status, len(cmds))
	for i, c := range cmds {
		r.status[i].Cmd = c.Name
		r.status[i].Exit = -1
	}

	r.stopChan = make(chan struct{})
	r.running = true
	r.Unlock()

	defer func() {
		r.Lock()
		r.running = false
		r.Unlock()
	}()

	for i, c := range cmds {
		if r.stopped() {
			return ErrStopped
		}

		cmd := cmd.NewCmd(c.Name, c.Args...)
		r.Lock()
		r.cmd = cmd
		r.cur = i
		r.Unlock()

		finalStatus := <-cmd.Start()

		r.Lock()
		r.status[i] = finalStatus
		r.cmd = nil
		r.cur = -1
		r.Unlock()

		if r.stopOnError && r.status[i].Exit != 0 {
			return ErrNonzeroExit
		}
	}

	return nil
}

// Stop stops Run if Run is still running. The return error is from stopping
// the currently active Process, if any. Stop is idempotent.
func (r *RunSync) Stop() error {
	r.Lock()
	defer r.Unlock()

	// If Run isn't running, there's nothing to do
	if !r.running {
		return nil
	}

	// If not already stopped, close the stopChan to stop. This ensures that Run
	// will not run any more commands.
	if !r.stopped() {
		close(r.stopChan)
	}

	// If there's an active cmd, stop it
	var err error
	if r.cmd != nil {
		err = r.cmd.Stop()
	}

	// Return cmd.Stop error, if any
	return err
}

func (r *RunSync) Status() ([]cmd.Status, int) {
	r.Lock()
	defer r.Unlock()
	if r.cmd != nil {
		// Get current status of currently running command
		r.status[r.cur] = r.cmd.Status()
	}
	return r.status, r.cur
}

func (r *RunSync) stopped() bool {
	select {
	case <-r.stopChan:
		return true
	default:
		return false
	}
}
