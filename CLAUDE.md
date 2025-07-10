# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**vel** is a Go HTTP framework focused on simplicity and automatic code generation. It enables developers to write type-safe HTTP handlers that automatically generate OpenAPI specifications and client code in Go and TypeScript.

## Development Commands

- **Build**: `go build -v ./...`
- **Test**: `go test -v ./...` (requires `goimports` and `prettier` for full test suite)
- **Format**: `go fmt ./...`
- **Tidy**: `go mod tidy`

## Core Architecture

### Framework Core
- **`router.go`**: Main HTTP router with generic handler registration using Go 1.24 generics
- **`context.go`**: Request context wrapper with automatic JSON marshaling/unmarshaling
- Built on standard `net/http` with full compatibility - no custom abstractions

### Code Generation System
- **`gen/`**: Template-based client generation for Go and TypeScript
  - Uses Go's `text/template` for code generation
  - Performs type analysis using reflection
  - Supports post-processing with `goimports` and `prettier`
- **`openapi/`**: OpenAPI 3.0 specification generation from handler definitions

### Handler Pattern
Handlers use generics for type safety:
```go
func MyHandler(ctx  context.Context, requestBody I) (responseBody O, callError *vel.Error) {
    // Automatic request unmarshaling to O response type
    // Return response or error
}
```

## Key Design Principles

1. **Code-first approach**: API specs generated from implementation, not documentation
2. **Type safety**: Leverages Go generics for compile-time guarantees
3. **Simplicity**: Opinionated design - no path parameters, explicit types required
4. **Standard library compatibility**: Full `net/http` compatibility maintained

## Code Structure

- Root files contain the core framework (router, context, error handling)
- `gen/` contains all code generation logic and templates
- `openapi/` handles OpenAPI 3.0 specification generation
- Framework uses minimal external dependencies (gorilla/schema, gopkg.in/yaml.v3)

## Common Patterns

- Handlers automatically unmarshal request bodies for POST/PUT/PATCH
- Error responses use structured `*Error` type with status codes
- OpenAPI generation includes header specs and validation rules
- Client generation supports both Go and TypeScript targets

## Testing Requirements

Full test suite requires:
- `goimports` for Go code formatting
- `prettier` for TypeScript code formatting
- Tests verify both runtime behavior and generated code quality
