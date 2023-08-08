package main

import (
	"github.com/coreos/go-systemd/daemon"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	wg *sync.WaitGroup
)

func loop() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	defer wg.Done()
	for {
		sig := <-sigs

		switch sig {
		case syscall.SIGINT, syscall.SIGTERM:
			// stop!
			return
		}
	}
}

func main() {
	wg = &sync.WaitGroup{}
	wg.Add(1)
	go loop()

	sent, e := daemon.SdNotify(false, "READY=1")
	if e != nil {
		panic(e)
	}
	if !sent {
		panic("SystemD notify NOT sent\n")
	}

	log.Println("wait till SIGINT|SIGTERM")
	wg.Wait()
	log.Printf("wait done")
}
