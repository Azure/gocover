package main

import (
	"os"

	"github.com/Azure/gocover/pkg/cmd"
)

func main() {
	command := cmd.NewGoCoverCommand()
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
