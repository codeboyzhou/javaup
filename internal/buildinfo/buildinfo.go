// Package buildinfo exposes values that can be replaced at build time.
package buildinfo

// Version is the application version. Release builds can replace it with:
//
//	go build -ldflags "-X github.com/codeboyzhou/javaup/internal/buildinfo.Version=v1.0.0" ./cmd/jup
var Version = "dev"
