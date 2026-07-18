package main

import (
	"context"
	"os"

	"github.com/codeboyzhou/javaup/internal/buildinfo"
	"github.com/codeboyzhou/javaup/internal/cli"
)

func main() {
	info := buildinfo.Current()

	app := cli.New(cli.Options{
		Name:        "jup",
		ProductName: "javaup",
		Description: "A command-line tool for managing Java versions.",
		Version:     info.Version,
		Platform:    info.Platform,
		Commit:      info.Commit,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	})

	os.Exit(app.Run(context.Background(), os.Args[1:]))
}
