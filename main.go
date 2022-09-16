package main

import (
	"errors"
	"os"

	"github.com/Azure/gocover/pkg/cmd"
	"github.com/Azure/gocover/pkg/gocover"
)

var (
	// Override following variables by -ldflags:
	//  `go build -ldflags "-X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}"`
	//
	// Gocover uses goreleaser/goreleaser-action@v3 to help release, it automcatically inject following variables:
	// which can be found at https://goreleaser.com/cookbooks/using-main.version
	version = "v1.0.0"
	commit  = "none"
	date    = "2022-09-10T00:00:00Z" // https://pkg.go.dev/time#pkg-constants RFC3339
)

func main() {
	command := cmd.NewGoCoverCommand(version, commit, date)
	if err := command.Execute(); err != nil {
		exitCode := gocover.GeneralErrorExitCode
		var e *gocover.GoCoverError
		if errors.As(err, &e) {
			exitCode = e.ExitCode
		}
		os.Exit(exitCode)
	}
}
