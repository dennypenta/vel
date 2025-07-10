# vel

A type-safe Go HTTP framework with automatic OpenAPI generation and client code generation.

## Features

- **Type-safe handlers** using Go generics for compile-time safety
- **Automatic OpenAPI 3.0 generation** from your handler implementations
- **Client code generation** for Go and TypeScript
- **Full net/http compatibility** - no custom abstractions
- **Code-first approach** - API specs generated from implementation, not documentation
- **Minimal dependencies** - built on Go standard library

## Quick Start

```go
package main

import (
	"context"
	"net/http"

	"github.com/dennypenta/vel"
)

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string `json:"message"`
}

func HelloHandler(ctx context.Context, req HelloRequest) (HelloResponse, *vel.Error) {
	return HelloResponse{
		Message: "Hello, " + req.Name,
	}, nil
}

func main() {
	router := vel.NewRouter()
	vel.RegisterPost(router, "/hello", HelloHandler)

	http.ListenAndServe(":8080", router.Mux())
}
```

## Installation

```bash
go get github.com/dennypenta/vel
```

## Development

- **Build**: `go build -v ./...`
- **Test**: `go test -v ./...`
- **Format**: `go fmt ./...`

## Design Philosophy

- **Opinionated**: No path parameters, explicit types required
- **Simple**: Leverages Go's type system for safety
- **Compatible**: Full net/http compatibility maintained
- **Generated**: OpenAPI specs and clients generated from code
