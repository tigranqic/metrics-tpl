// Package main shows valid main package usage.
package main

import (
	"log"
	"os"
)

// main is the entry point.
func main() {
	// These are OK inside main function
	log.Fatal("error in main")
	os.Exit(1)
}
