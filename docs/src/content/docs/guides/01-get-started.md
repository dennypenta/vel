---
title: Getting Started
description: Get started with vel
---

## Introduction

**vel** is a Go HTTP framework designed for developers who want type safety, automatic documentation, and minimal boilerplate.
All of it without reflect magic.
Built on Go's standard `net/http` library with full compatibility, vel leverages Go generics to provide compile-time safety while automatically generating OpenAPI specifications and client code.

### Key Features

- **Automatic OpenAPI 3.0 generation** from your handler implementations, it defines what you implement, not what you promise
- **Client code generation** for Go and TypeScript
- **Full net/http compatibility** - no custom abstractions
- **Comfortable experience** - just define the arguments, no dancing with jsons, bodies, etc.

### When to Use vel

Well, it's up to you til your boss decided you are gonna use vel

### Framework Philosophy

There are a few ideas the framework keeps in mind, it's highly opinionated. Therefore, the development speed is coming with some constrains

- **REST** sucks. vel doesn't use parameters in path and encourages using only POST HTTP method, mostly in order to avoid long discussions in the team which method to PICK, but also some technical aspects e.g. not to let browsers knock your doors.
  It supports GET as well and it's up to you use the others.
- **Dev experience first** is great. Let's delegate transport layer to the boring library.
- **Code gen** is great. Let's not to write the HTTP clients every now and again, leave this job to the boring library.
- **Document first** sucks. So let's provide the doc to what we have implemented.

## What does not support and will never do

It means every related issue will be closed without any comments. If you like talking about why contact me outside of github.

- _REST_
- _ORM_

## Installation

### Prerequisites

- Go 1.24+ (the project might work with lower versions, but wasn't tested)

### Install via go get

```bash
go get github.com/dennypenta/vel
```

## Quick Start

Let's create your first vel API with a simple "Hello" handler.

### 1. Your First vel Handler

Create a new Go file with the following code:

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
	vel.RegisterPost(router, "hello", HelloHandler)

	http.ListenAndServe(":8080", router.Mux())
}
```

### 2. Understanding the Handler Pattern

vel handlers use Go generics for type safety:

```go
func MyHandler(ctx context.Context, requestBody I) (responseBody O, callError *vel.Error) {
    // Automatic request unmarshaling to I request type
    // Return O response type or error
}
```

- Request bodies are automatically unmarshaled to your input type
- Response bodies are automatically marshaled from your output type
- Errors use the structured `*vel.Error` type with status codes

### 3. Running Your First API

Run your application:

```bash
go run main.go
```

Your API will be available at `http://localhost:8080`

### 4. Testing with curl

Test your handler with curl:

```bash
curl -X POST http://localhost:8080/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "World"}'
```

Expected response:

```json
{ "message": "Hello, World" }
```
