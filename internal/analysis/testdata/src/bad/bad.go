// Package bad contains code that fails the analyzer.
package bad

import (
	"log"
	"os"
)

// BadFunction uses panic which is always wrong.
func BadFunction() {
	panic("something went wrong") // want "panic\\(\\) detected"
}

// BadLogFatal calls log.Fatal in this package (not main, so it should NOT be caught).
func BadLogFatal() {
	log.Fatal("this is bad")
}

// BadOsExit calls os.Exit in this package (not main, so it should NOT be caught).
func BadOsExit() {
	os.Exit(1)
}

// AnotherBadFunction also has panic.
func AnotherBadFunction(x int) string {
	if x < 0 {
		panic("negative value") // want "panic\\(\\) detected"
	}
	return "OK"
}
