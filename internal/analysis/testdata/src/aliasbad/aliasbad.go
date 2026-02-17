package main

import myos "os"

func foo() {
	myos.Exit(1) // want "os\\.Exit\\(\\) detected outside main function"
}
