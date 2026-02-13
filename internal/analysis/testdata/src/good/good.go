// Package good contains code that passes the analyzer.
package good

import (
	"fmt"
	"log"
	"os"
)

// SafeFunction uses no dangerous calls.
func SafeFunction() error {
	fmt.Println("This is safe")
	return nil
}

// AnotherSafeFunction also safe.
func AnotherSafeFunction() {
	x := 42
	_ = x
}

// MainPackageExample shows correct usage in main package.
func ValidLogFatal() {
	// This is OK in main package main function
	log.Fatal("application failed")
}

// ValidOsExit shows correct os.Exit usage.
func ValidOsExit(code int) {
	os.Exit(code)
}
