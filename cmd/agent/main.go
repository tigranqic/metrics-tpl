package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tigranqic/metrics-tpl/internal/agent"
)

func main() {
	serverURL := "http://localhost:8080"
	a := agent.NewAgent(serverURL, 2*time.Second, 10*time.Second)

	stop := make(chan struct{})
	go a.Run(stop)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	close(stop)
}
