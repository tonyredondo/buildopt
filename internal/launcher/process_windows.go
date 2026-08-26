//go:build windows

package launcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const createNewProcessGroup = 0x00000200

func notifyOptimizeLearningContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt)
}

func executeChild(childArgs []string, environmentOverrides map[string]string, stdin io.Reader, stdout, stderr io.Writer) childExecution {
	return executeChildWithReserved(childArgs, environmentOverrides, nil, stdin, stdout, stderr)
}

func executeChildWithReserved(childArgs []string, environmentOverrides map[string]string, additionalReserved []string, stdin io.Reader, stdout, stderr io.Writer) childExecution {
	command := exec.Command(childArgs[0], childArgs[1:]...)
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNewProcessGroup}
	command.Env = replaceEnvironmentWithReserved(os.Environ(), environmentOverrides, additionalReserved)
	startedAt := time.Now()
	if err := command.Start(); err != nil {
		return childExecution{startedAt: startedAt, err: err}
	}
	job, err := newChildJob(command.Process.Pid)
	if err != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return childExecution{started: true, startedAt: startedAt, completedAt: time.Now(), err: fmt.Errorf("protect child process tree: %w", err)}
	}
	defer windows.CloseHandle(job)
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, os.Interrupt)
	stop := make(chan struct{})
	stopped := make(chan struct{})
	cancelled := make(chan struct{}, 1)
	go func() {
		defer close(stopped)
		select {
		case <-stop:
		case <-signals:
			if err := windows.TerminateJobObject(job, 3); err != nil {
				_, _ = fmt.Fprintf(stderr, "buildopt: cannot terminate child job: %v\n", err)
				return
			}
			cancelled <- struct{}{}
		}
	}()
	waitErr := command.Wait()
	signal.Stop(signals)
	close(stop)
	<-stopped
	return childExecution{started: true, startedAt: startedAt, completedAt: time.Now(), cancelled: len(cancelled) > 0, err: waitErr}
}

func newChildJob(processID int) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}
	information := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	information.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&information)), uint32(unsafe.Sizeof(information))); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(processID))
	if err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	return job, nil
}

func platformSignalExitCode(*exec.ExitError) (int, bool) { return 0, false }
