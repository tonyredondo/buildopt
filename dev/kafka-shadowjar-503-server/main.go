// Command kafka-shadowjar-503-server exposes a loopback HTTP endpoint that
// rejects every request. It exists only for the Kafka POC fallback fixture.
package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: kafka-shadowjar-503-server READY_FILE")
		os.Exit(64)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	readyFile := os.Args[1]
	if err := os.WriteFile(readyFile, []byte("http://"+listener.Addr().String()+"\n"), 0o600); err != nil {
		panic(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusServiceUnavailable)
	})}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-done
		_ = server.Close()
	}()
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
