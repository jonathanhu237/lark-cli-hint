package main

import (
	"os"

	"lark-cue/internal/app"
)

func main() {
	os.Exit(app.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
