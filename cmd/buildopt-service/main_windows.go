//go:build windows

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/svc"
)

type serviceHandler struct {
	binary string
	args   []string
}

func (handler serviceHandler) Execute(
	_ []string,
	requests <-chan svc.ChangeRequest,
	status chan<- svc.Status,
) (bool, uint32) {
	const accepts = svc.AcceptStop | svc.AcceptShutdown
	status <- svc.Status{State: svc.StartPending}
	command := exec.Command(handler.binary, handler.args...)
	if err := command.Start(); err != nil {
		return false, 1
	}
	finished := make(chan error, 1)
	go func() { finished <- command.Wait() }()
	status <- svc.Status{State: svc.Running, Accepts: accepts}
	for {
		select {
		case err := <-finished:
			if err != nil {
				return false, 1
			}
			return false, 0
		case request := <-requests:
			switch request.Cmd {
			case svc.Interrogate:
				status <- request.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				_ = command.Process.Kill()
				<-finished
				return false, 0
			}
		}
	}
}

func main() {
	flags := flag.NewFlagSet("buildopt-service", flag.ExitOnError)
	name := flags.String("service-name", "", "registered Windows service name")
	component := flags.String("component", "", "server or edge")
	config := flags.String("config", "", "absolute component configuration")
	_ = flags.Parse(os.Args[1:])
	if *name == "" || (*component != "server" && *component != "edge") ||
		!filepath.IsAbs(*config) || flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: buildopt-service --service-name NAME --component server|edge --config ABSOLUTE_PATH")
		os.Exit(64)
	}
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bin := filepath.Dir(executable)
	var binary string
	var args []string
	if *component == "server" {
		binary = filepath.Join(bin, "buildopt-server.exe")
		args = []string{"serve", "--self-hosted-config", *config}
	} else {
		binary = filepath.Join(bin, "buildopt-edge.exe")
		args = []string{"serve", "--config", *config}
	}
	if err := svc.Run(*name, serviceHandler{binary: binary, args: args}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
