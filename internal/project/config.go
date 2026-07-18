// Package project coordinates project detection and local configuration.
package project

import (
	"time"

	"github.com/codeboyzhou/javaup/internal/buildtool"
	"github.com/codeboyzhou/javaup/internal/javainfo"
)

const currentSchemaVersion = 1

// Config is the persisted description of an initialized project.
type Config struct {
	SchemaVersion int                   `json:"schemaVersion"`
	ProjectRoot   string                `json:"projectRoot"`
	BuildTool     buildtool.Info        `json:"buildTool"`
	Java          javainfo.Installation `json:"java"`
	InitializedAt time.Time             `json:"initializedAt"`
}
