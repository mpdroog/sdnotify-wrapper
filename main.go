package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var (
	sigs    chan os.Signal
	closing bool
)

const maxPktSize = 1024

func readUnixConn(conn *net.UnixConn, msgs chan []byte) {
	for {
		msg := make([]byte, maxPktSize)
		nread, err := conn.Read(msg)
		if err != nil {
			if !closing {
				fmt.Fprintf(os.Stderr, "Failed to read from unix socket: %v\n", err)
				msgs <- []byte("ERR")
				sigs <- syscall.SIGINT
			}
			close(msgs)
			return
		}

		msgs <- msg[:nread]
	}
}

type unixProxy struct {
	src   string
	local *net.UnixConn
}

func newProxy(src string) (*unixProxy, error) {
	if _, err := os.Stat(src); err == nil {
		if err := os.Remove(src); err != nil {
			return nil, err
		}
	}

	// start listening
	local, err := net.ListenUnixgram("unixgram", &net.UnixAddr{
		Name: src,
		Net:  "unixgram",
	})

	if err != nil {
		return nil, err
	}

	return &unixProxy{
		src:   src,
		local: local,
	}, nil
}

func (p *unixProxy) run(cancel chan struct{}, ready chan struct{}) {
	msgs := make(chan []byte)

	go readUnixConn(p.local, msgs)

	for {
		select {
		case msg := <-msgs:
			if bytes.Equal(msg, []byte("ERR")) {
				// failure
				return
			}
			if bytes.Equal(msg, []byte("READY=1")) {
				ready <- struct{}{}
				return
			}
			fmt.Printf("unhandled msg=%s\n", msg)

		case <-cancel:
			p.local.Close()
			return
		}
	}
}

func (p *unixProxy) Close() error {
	if e := p.local.Close(); e != nil {
		return e
	}
	return os.Remove(p.src)
}

func forkExec(argv []string) (*os.Process, error) {
	return os.StartProcess(argv[0], argv, &os.ProcAttr{
		Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
	})
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s proxy-socket cmd ...\n", os.Args[0])
		os.Exit(1)
	}

	proxySock := os.Args[1]
	proxyPid := proxySock + ".pid"

	sigs = make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGCHLD)

	os.Setenv("NOTIFY_SOCKET", proxySock)

	proxy, err := newProxy(proxySock)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating proxy: %v\n", err)
		os.Exit(1)
	}
	defer proxy.Close() // ignoring socket close and failure to delete here

	// fork/exec
	proc, err := forkExec(os.Args[2:len(os.Args)])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}

	// proxy the unixgram messages
	cancel := make(chan struct{})
	ready := make(chan struct{})
	go proxy.run(cancel, ready)

	for {
		select {
		case <-ready:
			// Client says it can fly without us
			pid := proc.Pid
			err = os.WriteFile(proxyPid, []byte(fmt.Sprintf("%d", pid)), 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "proc saving pid failed: %v\n", err)
			}
			err = proc.Release()
			if err != nil {
				fmt.Fprintf(os.Stderr, "proc release failed: %v\n", err)
			}
			closing = true
			fmt.Printf("%d\n", pid)
			return

		case sig := <-sigs:
			// We got interrupted by OS, cancel all
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				// propogate to child
				proc.Signal(sig)

			case syscall.SIGCHLD:
				ps, err := proc.Wait()
				if err != nil {
					fmt.Fprintf(os.Stderr, "waitpid failed: %v\n", err)
					os.Exit(1)
				}

				close(cancel)

				ec := ps.Sys().(syscall.WaitStatus).ExitStatus()
				os.Exit(ec)
			}
		}

	}
}
