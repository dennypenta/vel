package gen

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dennypenta/vel"
)

// ClientGeneratorConfig holds configuration for generating API clients
type ClientGeneratorConfig struct {
	TypeName    string
	PackageName string
	OutputDir   string
	Language    string // "go" or "ts"
	PostProcess string // e.g., "goimports" or "prettier"
}

// GenerateClientToFile generates an API client and writes it to a file
func GenerateClientToFile(router *vel.Router, config ClientGeneratorConfig) error {
	// Determine file extension and template
	var filename string
	switch config.Language {
	case "go":
		filename = "client.go"
	case "ts":
		filename = "client.ts"
	default:
		return fmt.Errorf("language %s is not supported", config.Language)
	}

	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return err
	}

	filePath := filepath.Join(config.OutputDir, filename)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return GenerateClient(router, file, config)
}

// GenerateClient generates an API client and writes it to the provided writer
func GenerateClient(router *vel.Router, w io.Writer, config ClientGeneratorConfig) error {
	// Create generator
	generator, err := New(ClientDesc{
		TypeName:    config.TypeName,
		PackageName: config.PackageName,
	}, router.Meta())
	if err != nil {
		return err
	}

	// Determine template
	var template string
	switch config.Language {
	case "go":
		template = "go:default"
	case "ts":
		template = "ts:default"
	default:
		return fmt.Errorf("language %s is not supported", config.Language)
	}

	// Generate client code
	return generator.Generate(w, template, config.PostProcess)
}

// GenerateOpenAPIToFile generates an OpenAPI specification and writes it to a file
func GenerateOpenAPIToFile(router *vel.Router, outputPath, title, version string) error {
	file, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return GenerateOpenAPI(router, file, title, version)
}

func GenerateOpenAPI(router *vel.Router, w io.Writer, title, version string) error {
	generator, err := New(ClientDesc{
		TypeName:    "Client",
		PackageName: "client",
	}, router.Meta())
	if err != nil {
		return err
	}
	return generator.GenerateOpenAPIYAML(w, title, version)
}
