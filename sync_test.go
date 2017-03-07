package run_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/go-cmd/run"
	"github.com/go-test/deep"
)

func TestRunSyncOK(t *testing.T) {
	cmds := []cmd.Cmd{
		{
			Name: "echo",
			Args: []string{"hello"},
		},
		{
			Name: "echo",
			Args: []string{"world"},
		},
	}

	r := run.NewRunSync(true)
	err := r.Run(cmds)
	if err != nil {
		t.Fatal(err)
	}
	gotStatus, cur := r.Status()
	if cur != -1 {
		t.Errorf("got cur status %d, expected -1")
	}
	if len(gotStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}
	if gotStatus[0].PID == gotStatus[1].PID {
		t.Error("status[0] and status[1] PIDs are equal, expected different")
	}
	expectStatus := []cmd.Status{
		{
			Cmd:      "echo",
			PID:      gotStatus[0].PID, // nondeterministic
			Complete: true,
			Exit:     0,
			Error:    nil,
			Runtime:  gotStatus[0].Runtime, // nondeterministic
			Stdout:   []string{"hello"},
			Stderr:   []string{},
		},
		{
			Cmd:      "echo",
			PID:      gotStatus[1].PID, // nondeterministic
			Complete: true,
			Exit:     0,
			Error:    nil,
			Runtime:  gotStatus[1].Runtime, // nondeterministic
			Stdout:   []string{"world"},
			Stderr:   []string{},
		},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
}

func TestRunSyncStop(t *testing.T) {
	cmds := []cmd.Cmd{
		{
			Name: "./test/count-and-sleep",
			Args: []string{"5", "5"},
		},
		{
			Name: "echo",
			Args: []string{"hello"},
		},
	}

	r := run.NewRunSync(true)

	// Stop shouldn't do anything before Run is called
	if err := r.Stop(); err != nil {
		t.Error(err)
	}

	var gotStatus []cmd.Status
	var gotErr error
	doneChan := make(chan struct{})
	go func() {
		gotErr = r.Run(cmds)
		gotStatus, _ = r.Status()
		close(doneChan)
	}()

	time.Sleep(1 * time.Second)

	// Test Status while running
	curStatus, cur := r.Status()
	if cur != 0 {
		t.Error("got cur status %d, expected 0", cur)
	}
	if len(curStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}
	expectStatus := []cmd.Status{
		{
			Cmd:      "./test/count-and-sleep",
			PID:      curStatus[0].PID, // nondeterministic
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  curStatus[0].Runtime, // nondeterministic
			Stdout:   []string{"1"},
			Stderr:   []string{},
		},
		{ // zero value for cmd.Status
			Cmd:      "echo",
			PID:      0,
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  0,
			Stdout:   nil,
			Stderr:   nil,
		},
	}
	if diffs := deep.Equal(curStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}

	// Stop the runner before the 1st job has completed
	if err := r.Stop(); err != nil {
		t.Error(err)
	}

	// Run should return instantly after Stop
	select {
	case <-doneChan:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	// 2 jobs in = 2 status out
	if len(gotStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}

	expectStatus[0] = cmd.Status{
		Cmd:      "./test/count-and-sleep",
		PID:      gotStatus[0].PID, // nondeterministic
		Complete: false,
		Exit:     -1,
		Error:    errors.New("signal: terminated"),
		Runtime:  gotStatus[0].Runtime, // nondeterministic
		Stdout:   []string{"1"},
		Stderr:   []string{},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}

	// Stop is idempotent
	if err := r.Stop(); err != nil {
		t.Error(err)
	}
}

func TestRunSyncStopOnError(t *testing.T) {
	cmds := []cmd.Cmd{
		{ // causes Run to return ErrNonzeroExit
			Name: "false",
			Args: []string{},
		},
		{ // doesn't run
			Name: "echo",
			Args: []string{"hello"},
		},
	}

	r := run.NewRunSync(true)
	err := r.Run(cmds)
	if err != run.ErrNonzeroExit {
		t.Error("got nil err, expected ErrNonzeroExit")
	}
	gotStatus, cur := r.Status()
	if cur != -1 {
		t.Errorf("got cur status %d, expected -1", cur)
	}
	if len(gotStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}
	expectStatus := []cmd.Status{
		{
			Cmd:      "false",
			PID:      gotStatus[0].PID, // nondeterministic
			Complete: true,
			Exit:     1,
			Error:    nil,
			Runtime:  gotStatus[0].Runtime, // nondeterministic
			Stdout:   []string{},
			Stderr:   []string{},
		},
		{ // zero value for cmd.Status
			Cmd:      "echo",
			PID:      0,
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  0,
			Stdout:   nil,
			Stderr:   nil,
		},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}

	// Same commands but stopOnError = false so failure is ignored
	r = run.NewRunSync(false)

	err = r.Run(cmds)
	if err != nil {
		t.Error(err)
	}
	gotStatus, _ = r.Status()
	if len(gotStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}
	expectStatus = []cmd.Status{
		{
			Cmd:      "false",
			PID:      gotStatus[0].PID, // nondeterministic
			Complete: true,
			Exit:     1,
			Error:    nil,
			Runtime:  gotStatus[0].Runtime, // nondeterministic
			Stdout:   []string{},
			Stderr:   []string{},
		},
		{
			Cmd:      "echo",
			PID:      gotStatus[1].PID, // nondeterministic
			Complete: true,
			Exit:     0,
			Error:    nil,
			Runtime:  gotStatus[1].Runtime, // nondeterministic
			Stdout:   []string{"hello"},
			Stderr:   []string{},
		},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
}

func TestRunSyncStopped(t *testing.T) {
	// sigterm-exit-0 exits 0 on SIGTERM which is what Stop sends, so although
	// the exit code is zero, the main Run loop should stop when it checks the
	// called to stopped(). In other words: Stop always stops even if the command
	// returns exit=0 after SIGTERM and stopOnError=false.
	cmds := []cmd.Cmd{
		{
			Name: "./test/sigterm-exit-0",
			Args: []string{},
		},
		{
			Name: "echo",
			Args: []string{"hello"},
		},
	}

	r := run.NewRunSync(false)

	var gotStatus []cmd.Status
	var gotErr error
	doneChan := make(chan struct{})
	go func() {
		gotErr = r.Run(cmds)
		gotStatus, _ = r.Status()
		close(doneChan)
	}()

	time.Sleep(1 * time.Second)

	// Check that Run returns ErrRunning on 2nd+ call
	err := r.Run(cmds)
	if err != run.ErrRunning {
		t.Error("got nil error, expected ErrRunning")
	}

	// Stop the first cmd which will return 0
	if err := r.Stop(); err != nil {
		t.Error(err)
	}

	// Run should return instantly after Stop
	select {
	case <-doneChan:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for Run to return")
	}

	if len(gotStatus) != 2 {
		t.Fatal("expected 2 Status, got %d", len(gotStatus))
	}
	expectStatus := []cmd.Status{
		{
			Cmd:      "./test/sigterm-exit-0",
			PID:      gotStatus[0].PID, // nondeterministic
			Complete: false,
			Exit:     0,
			Error:    nil,
			Runtime:  gotStatus[0].Runtime, // nondeterministic
			Stdout:   []string{},
			Stderr:   []string{"Terminated: 15"},
		},
		{ // zero value for cmd.Status
			Cmd:      "echo",
			PID:      0,
			Complete: false,
			Exit:     -1,
			Error:    nil,
			Runtime:  0,
			Stdout:   nil,
			Stderr:   nil,
		},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
}
