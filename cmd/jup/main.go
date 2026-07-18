package main

import (
	"context"
	"os"

	"github.com/codeboyzhou/javaup/internal/buildinfo"
	"github.com/codeboyzhou/javaup/internal/cli"
)

func main() {
	app := cli.New(cli.Options{
		Name:        "jup",
		Description: "A command-line tool for managing Java versions.",
		Version:     buildinfo.Version,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	})

	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
