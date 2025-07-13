---
title: Global Options
description: Configure vel's global behavior with GlobalOpts - error processing, status codes, and OPTIONS handling.
---

## Overview

vel provides a global configuration system through the `GlobalOpts` variable,
allowing you to customize framework behavior across all handlers.

### GlobalOpts Structure

The `GlobalOpts` variable is of type `Opts` and provides the following configuration options:

```go
type Opts struct {
    ProcessErr       func(r *http.Request, e *Error)
    MapCodeToStatus  func(code string) int
    SkipOptionMethod bool
}

var GlobalOpts = Opts{
    ProcessErr: nil,
    MapCodeToStatus: func(code string) int {
        if code == "" {
            return http.StatusInternalServerError
        }
        return http.StatusBadRequest
    },
    SkipOptionMethod: false,
}
```

### Configuration Options

1. **ProcessErr** - Custom error processing function called before errors are returned to clients
2. **MapCodeToStatus** - Function that maps error codes to HTTP status codes
3. **SkipOptionMethod** - Boolean flag to control automatic OPTIONS method handling

### Default Behavior

- **ProcessErr**: `nil` (no custom error processing)
- **MapCodeToStatus**: Returns 500 for empty error codes, 400 for all others
- **SkipOptionMethod**: `false` (automatic OPTIONS handling enabled)

## Custom Error Processing

Configure global error processing with the `ProcessErr` function to implement logging, metrics, and custom error handling.

### Basic Error Processing

```go
vel.GlobalOpts.ProcessErr = func(r *http.Request, e *vel.Error) {
    // Log errors with request context
    log.Printf("API Error: %s - %s [%s %s]",
        e.Code, e.Message, r.Method, r.URL.Path)
}
```

## HTTP Status Code Mapping

Configure how error codes map to HTTP status codes using the `MapCodeToStatus` function.

### Default Status Code Mapping

```go
// Default implementation
vel.GlobalOpts.MapCodeToStatus = func(code string) int {
    if code == "" {
        return http.StatusInternalServerError // 500
    }
    return http.StatusBadRequest // 400
}
```

## OPTIONS Method Handling

Control automatic OPTIONS method registration with the `SkipOptionMethod` option.

### Default OPTIONS Handling

By default, vel automatically registers OPTIONS handlers for all registered routes:

```go
// Default behavior - OPTIONS handlers are automatically created
vel.GlobalOpts.SkipOptionMethod = false

// Register a POST handler
vel.RegisterPost(router, "users", CreateUserHandler)

// vel automatically registers:
// OPTIONS /users -> returns 200 OK
```

### Disabling Automatic OPTIONS

```go
// Disable automatic OPTIONS handling
vel.GlobalOpts.SkipOptionMethod = true
```
