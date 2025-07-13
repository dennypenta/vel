---
title: Code Generation
description: client generation
---

## Overview

vel provides an option to generate a client libraries and OpenAPI documentation from your Go handlers.
This code-first approach ensures your API documentation and client code are always in sync with your implementation.

### Supported Targets

- **Go**
- **TypeScript**
- **OpenAPI 3.0**: API specifications

## Client Generation

You have to write simple script in your project to generate a client

```go
package main

import (
    "os"
    "github.com/dennypenta/vel/gen"
)

func main() {
    router := myapp.NewRouter()

    err := gen.GenerateClientToFile(router, gen.ClientGeneratorConfig{
        // generated client class/struct name
        TypeName:    "Client",
        // Package name for Go clients (ignored for TypeScript)
        PackageName: "client",
        // Output directory path
        OutputDir:   "client",
        // "go" or "ts" are supported
        Language:    "go",
        // Post-processing pipe command
        PostProcess: "goimports",
    })
    if err != nil {
        panic(err)
    }
}
```

Alternatively, you can write a client to a buffer

```go
package main

import (
    "os"
    "bytes"
    "github.com/dennypenta/vel/gen"
)

func main() {
    router := myapp.NewRouter()

    var buf bytes.Buffer{}

    err := gen.GenerateClient(router, &buf, gen.ClientGeneratorConfig{
        // generated client class/struct name
        TypeName:    "Client",
        // Package name for Go clients (ignored for TypeScript)
        PackageName: "client",
        // "go" or "ts" are supported
        Language:    "go",
        // Post-processing pipe command
        PostProcess: "goimports",
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(buf.String())
}
```

### Type Mapping

vel automatically maps Go types to target languages:

**Go to TypeScript Mapping:**

- `string` ’ `string`
- `int`, `int32`, `int64`, `float64` ’ `number`
- `bool` ’ `boolean`
- `[]Type` ’ `Type[]`
- `map[K]V` ’ `Record<K, V>`
- `time.Time` ’ `string` (ISO format)

### Post-processing

Post processing are shell commands that take the generate output and pipe it out.
Imagine it as shell `code | postProcessingCommand`, but technically it is `postProcessing < code`
