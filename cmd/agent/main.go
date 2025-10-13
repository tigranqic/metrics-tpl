package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tigranqic/metrics-tpl/internal/agent"
)

func main() {
	serverAddr := flag.String("a", "localhost:8080", "HTTP server address")
	reportIntervalStr := flag.String("r", "10", "Report interval in seconds")
	pollIntervalStr := flag.String("p", "2", "Poll interval in seconds")

	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown arguments: %v\n", flag.Args())
		os.Exit(1)
	}

	reportIntervalSec, err := strconv.Atoi(*reportIntervalStr)
	if err != nil || reportIntervalSec <= 0 {
		fmt.Fprintf(os.Stderr, "Invalid report interval: %s\n", *reportIntervalStr)
		os.Exit(1)
	}

	pollIntervalSec, err := strconv.Atoi(*pollIntervalStr)
	if err != nil || pollIntervalSec <= 0 {
		fmt.Fprintf(os.Stderr, "Invalid poll interval: %s\n", *pollIntervalStr)
		os.Exit(1)
	}

	reportInterval := time.Duration(reportIntervalSec) * time.Second
	pollInterval := time.Duration(pollIntervalSec) * time.Second

	serverAddrUrl := "http://" + *serverAddr
	a := agent.NewAgent(serverAddrUrl, pollInterval, reportInterval)

	stop := make(chan struct{})
	go a.Run(stop)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	close(stop)
}
