package main

import (
	"log"
	"os"
)

// init calls log.Fatal outside main function - should be caught.
func init() {
	log.Fatal("init error") // want "log\\.Fatal\\(\\) detected outside main function"
}

// badExit calls os.Exit outside main function - should be caught.
func badExit() {
	os.Exit(1) // want "os\\.Exit\\(\\) detected outside main function"
}

// main is the OK entry point.
func main() {
	_ = 42
}
