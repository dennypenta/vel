---
title: Core Concepts
description: Understanding vel's core concepts - handlers, routing
---

## Handler Pattern

vel uses Go generics to create type-safe HTTP handlers with automatic JSON marshaling and unmarshaling.
This approach provides compile-time safety while eliminating boilerplate code.

### Generic Handler Signature

Every vel handler follows this generic signature:

```go
type Handler[I, O any] func(ctx context.Context, i I) (O, *Error)
```

Where:

- **I**: Input type (request body for POST, query parameters for GET)
- **O**: Output type (response body)
- **ctx**: Standard Go context for request lifecycle management. It also holds `*http.Request and http.ResponseWriter` instances.
- **\*Error**: Framework-specific error type for structured error responses

### Handler Function Anatomy

Here's a complete example showing the handler pattern:

```go
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type CreateUserResponse struct {
    ID      int    `json:"id"`
    Name    string `json:"name"`
    Email   string `json:"email"`
    Created string `json:"created"`
}

func CreateUserHandler(ctx context.Context, req CreateUserRequest) (CreateUserResponse, *vel.Error) {
    if req.Name == "" {
        return CreateUserResponse{}, &vel.Error{
            Code:    "MISSING_NAME",
        }
    }

    user := CreateUserResponse{
        ID:      generateID(),
        Name:    req.Name,
        Email:   req.Email,
        Created: time.Now().Format(time.RFC3339),
    }

    return user, nil
}
```

### Input Type Handling

vel handles input types differently based on HTTP method:

- **GET**: Query parameters are decoded to input type using struct tags

```go
type SearchRequest struct {
    Query string `schema:"q"`
    Limit int    `schema:"limit"`
}

func SearchHandler(ctx context.Context, req SearchRequest) (struct{}, *vel.Error) {
    // GET /search?q=golang&limit=10
    // req.Query = "golang", req.Limit = 10
    return struct{}{}, nil
}
```

## Router System

vel's router system is built on Go's standard `net/http` package with additional features for handler registration and metadata collection.

### Creating a Router

```go
func main() {
    router := vel.NewRouter()

    // Register your handlers
    vel.RegisterPost(router, "users", CreateUserHandler)
    vel.RegisterGet(router, "users", GetUsersHandler)

    // Start the server
    http.ListenAndServe(":8080", router.Mux())
}
```

### Path Routing Convention

vel uses a simple operation ID-based routing system:

- **Pattern**: `METHOD /operationID`
- **Examples**:
  - `POST /users`
  - `GET /search`

### Handler Registration with Middleware

You can register handlers with standard `net/http` middlewares:

```go
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !r.Headers.Get("Authorization") == "Good" {
            w.WriteStatus(403)
            w.Write([]byte("Unfortunately, authorization header is not good"))
            return
        }
        next.ServeHTTP(w, r)
    })
}

vel.RegisterPost(router, "protected", protectedHandler, authMiddleware)
```

## Request Context

vel provides context wrapper functions to access the underlying HTTP request and response objects from within your handlers.

### Context Wrapper Functions

```go
// Access the HTTP request
func MyHandler(ctx context.Context, req MyRequest) (MyResponse, *vel.Error) {
    httpReq := vel.RequestFromContext(ctx)
    userAgent := httpReq.Header.Get("User-Agent")

    // Access the HTTP response writer
    w := vel.WriterFromContext(ctx)
    w.Header().Set("X-Custom-Header", "value")

    return MyResponse{}, nil
}
```

## Standard net/http handlers

You have 2 options to register a standard handler:

- Register using native router as is
- Register using provided API

### Register using native router

To achieve it vel exposes mux Rotuer which you can use

```go
	vel.NewRouter().Mux().Handle("POST /auth", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
```

### Register using vel API

vel provides the API to register a standard handler and describe all the necessary meta information and data types for documentation purpose

```go
	vel.RegisterHandlerFunc(router, vel.HandlerMeta{
		Input:       struct{}{},
		Output:      domain.GithubCallbackResponse{},
		Method:      "GET",
		OperationID: "auth",
        Spec: vel.Spec{
			Description: "auth handler",
		},
	}, handlers.GithubAuthHandler)
```
