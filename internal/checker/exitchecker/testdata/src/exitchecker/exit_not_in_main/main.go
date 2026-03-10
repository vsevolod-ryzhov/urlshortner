package main

import "os"

func someFunc() {
	os.Exit(1) // OK - not in main
}

func main() {
	// no exit here
}
