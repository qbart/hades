package main

import (
	"os"

	"github.com/SoftKiwiGames/hades/hades"
)

func main() {
	h := hades.New(os.Stdout, os.Stderr)
	h.Run()
}
