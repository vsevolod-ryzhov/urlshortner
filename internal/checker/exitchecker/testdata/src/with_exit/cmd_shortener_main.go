package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit in cmd/shortener/cmd_shortener_main.go is not allowed"
}
