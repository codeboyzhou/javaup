package maven

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxParentDepth = 16

var propertyPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type pomModel struct {
	Parent     *pomParent    `xml:"parent"`
	Properties pomProperties `xml:"properties"`
	Build      pomBuild      `xml:"build"`
}

type pomParent struct {
	RelativePath *string `xml:"relativePath"`
}

type pomProperties map[string]string

func (p *pomProperties) UnmarshalXML(decoder *xml.Decoder, start xml.StartElement) error {
	if *p == nil {
		*p = make(pomProperties)
	}

	for {
		token, err := decoder.Token()
		if err != nil {
			return err
		}

		switch value := token.(type) {
		case xml.StartElement:
			var content string
			if err := decoder.DecodeElement(&content, &value); err != nil {
				return err
			}
			(*p)[value.Name.Local] = strings.TrimSpace(content)
		case xml.EndElement:
			if value.Name == start.Name {
				return nil
			}
		}
	}
}

type pomBuild struct {
	Plugins          []pomPlugin         `xml:"plugins>plugin"`
	PluginManagement pomPluginManagement `xml:"pluginManagement"`
}

type pomPluginManagement struct {
	Plugins []pomPlugin `xml:"plugins>plugin"`
}

type pomPlugin struct {
	GroupID       string               `xml:"groupId"`
	ArtifactID    string               `xml:"artifactId"`
	Configuration compilerPluginConfig `xml:"configuration"`
}

type compilerPluginConfig struct {
	Release string `xml:"release"`
	Source  string `xml:"source"`
	Target  string `xml:"target"`
}

type pomData struct {
	properties map[string]string
	release    string
	source     string
	target     string
}

func detectBuildJavaVersion(pomPath string) (string, error) {
	data, err := loadPOM(pomPath, make(map[string]struct{}), 0)
	if err != nil {
		return "", err
	}

	candidates := []string{
		data.release,
		data.properties["maven.compiler.release"],
		data.target,
		data.properties["maven.compiler.target"],
		data.source,
		data.properties["maven.compiler.source"],
		data.properties["java.version"],
		data.properties["jdk.version"],
	}

	for _, candidate := range candidates {
		candidate = resolveProperties(candidate, data.properties)
		if candidate == "" {
			continue
		}

		version, err := normalizeJavaVersion(candidate)
		if err != nil {
			return "", fmt.Errorf("parse Java build version %q in %s: %w", candidate, pomPath, err)
		}
		return version, nil
	}

	return "", nil
}

func loadPOM(path string, visited map[string]struct{}, depth int) (pomData, error) {
	if depth >= maxParentDepth {
		return pomData{}, fmt.Errorf("maven parent hierarchy exceeds %d levels", maxParentDepth)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return pomData{}, fmt.Errorf("resolve POM path %s: %w", path, err)
	}
	absolutePath = filepath.Clean(absolutePath)
	if _, exists := visited[absolutePath]; exists {
		return pomData{}, fmt.Errorf("maven parent cycle detected at %s", absolutePath)
	}
	visited[absolutePath] = struct{}{}
	defer delete(visited, absolutePath)

	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return pomData{}, fmt.Errorf("read Maven POM %s: %w", absolutePath, err)
	}

	var model pomModel
	if err := xml.Unmarshal(content, &model); err != nil {
		return pomData{}, fmt.Errorf("parse Maven POM %s: %w", absolutePath, err)
	}

	data := pomData{properties: make(map[string]string)}
	if parentPath, ok := localParentPath(absolutePath, model.Parent); ok {
		if _, err := os.Stat(parentPath); err == nil {
			data, err = loadPOM(parentPath, visited, depth+1)
			if err != nil {
				return pomData{}, err
			}
		}
	}

	if data.properties == nil {
		data.properties = make(map[string]string)
	}
	for key, value := range model.Properties {
		data.properties[key] = value
	}

	configuration := compilerConfiguration(model.Build.Plugins)
	if configuration == (compilerPluginConfig{}) {
		configuration = compilerConfiguration(model.Build.PluginManagement.Plugins)
	}
	if configuration.Release != "" {
		data.release = configuration.Release
	}
	if configuration.Source != "" {
		data.source = configuration.Source
	}
	if configuration.Target != "" {
		data.target = configuration.Target
	}

	return data, nil
}

func localParentPath(pomPath string, parent *pomParent) (string, bool) {
	if parent == nil {
		return "", false
	}

	relativePath := "../pom.xml"
	if parent.RelativePath != nil {
		relativePath = strings.TrimSpace(*parent.RelativePath)
		if relativePath == "" {
			return "", false
		}
	}

	return filepath.Clean(filepath.Join(filepath.Dir(pomPath), filepath.FromSlash(relativePath))), true
}

func compilerConfiguration(plugins []pomPlugin) compilerPluginConfig {
	for _, plugin := range plugins {
		if strings.TrimSpace(plugin.ArtifactID) != "maven-compiler-plugin" {
			continue
		}
		groupID := strings.TrimSpace(plugin.GroupID)
		if groupID == "" || groupID == "org.apache.maven.plugins" {
			return plugin.Configuration
		}
	}
	return compilerPluginConfig{}
}

func resolveProperties(value string, properties map[string]string) string {
	value = strings.TrimSpace(value)
	for range 10 {
		changed := false
		value = propertyPattern.ReplaceAllStringFunc(value, func(match string) string {
			parts := propertyPattern.FindStringSubmatch(match)
			replacement, exists := properties[parts[1]]
			if !exists {
				return match
			}
			changed = true
			return strings.TrimSpace(replacement)
		})
		if !changed {
			break
		}
	}
	return value
}

func normalizeJavaVersion(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "1.")
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return "", fmt.Errorf("unsupported version syntax")
	}
	return value[:end], nil
}
