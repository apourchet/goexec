package goexec

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Cmd is a thin wrapper around exec.Cmd, making it thread safe
// and giving extra syntactic sugar and functionality.
// Wait is allowed to be called by multiple threads.
type Cmd struct {
	sync.Mutex

	start   sync.Once
	wait    sync.Once
	waitSem chan struct{}

	cmd      *exec.Cmd
	startErr error
	waitErr  error
}

// New creates a new wrapped exec.Cmd struct that allows for multithreaded
// operations.
func New(cmd *exec.Cmd) *Cmd {
	return &Cmd{
		cmd:     cmd,
		waitSem: make(chan struct{}, 1),
	}
}

// Command returns a new Cmd struct to execute the command with the
// given arguments.
func Command(name string, args ...string) *Cmd {
	return New(exec.Command(name, args...))
}

// Start calls Start on the underlying os.exec.Cmd object. Multiple
// consecutive calls to Start will return the first error and only call
// Start once on the underlying Cmd object.
func (cmd *Cmd) Start() error {
	if cmd.Exited() {
		return fmt.Errorf("command already exitted")
	}
	cmd.Lock()
	defer cmd.Unlock()

	cmd.start.Do(func() {
		err := cmd.cmd.Start()
		cmd.startErr = err
	})
	return cmd.startErr
}

// Run starts the command and then calls wait on it. Returns either the
// Start error or the Wait error if the first was nil.
func (cmd *Cmd) Run() error {
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}

// Wait blocks until the underlying command finishes running. Wait will
// block forever if the command is never started.
func (cmd *Cmd) Wait() error {
	go func() {
		cmd.wait.Do(func() {
			err := cmd.cmd.Wait()
			cmd.Lock()
			cmd.waitErr = err
			cmd.Unlock()
		})
		cmd.waitSem <- struct{}{}
	}()
	<-cmd.waitSem
	defer func() {
		cmd.waitSem <- struct{}{}
	}()
	return cmd.waitErr
}

// WithOutput sets the stdout and stderr writers to the writer passed
// in.
func (cmd *Cmd) WithOutput(w io.Writer) *Cmd {
	cmd.Lock()
	defer cmd.Unlock()
	cmd.cmd.Stdout = w
	cmd.cmd.Stderr = w
	return cmd
}

// WithInput sets the internal stdin of the exec.Cmd.
func (cmd *Cmd) WithInput(r io.Reader) *Cmd {
	cmd.Lock()
	defer cmd.Unlock()
	cmd.cmd.Stdin = r
	return cmd
}

// ExitCode returns it the exit code of the command and a
// possible error if we failed to run it at all.
// returns -1 if the process was never started, along with the error
// describing what the issue was.
func (cmd *Cmd) ExitCode() (int, error) {
	cmd.Lock()
	defer cmd.Unlock()
	if cmd.cmd.ProcessState == nil {
		return -1, fmt.Errorf("command was not run yet")
	}
	return cmd.cmd.ProcessState.ExitCode(), nil
}

// Exited returns whether or not the underlying process has exited.
func (cmd *Cmd) Exited() bool {
	cmd.Lock()
	defer cmd.Unlock()
	if cmd.cmd.ProcessState == nil {
		return false
	}
	return cmd.cmd.ProcessState.Exited()
}
