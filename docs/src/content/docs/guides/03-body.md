---
title: Request and Response Handling
description: Understanding how vel handles request processing, response generation
---

## Request Processing

vel automatically handles request processing through its generic handler system, providing seamless JSON unmarshaling and query parameter handling.

### Automatic JSON Unmarshaling

For POST vel automatically unmarshals JSON request bodies to your input type:

```go
type CreateUserRequest struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Age      int    `json:"age"`
    IsActive bool   `json:"isActive"`
}

func CreateUserHandler(ctx context.Context, req CreateUserRequest) (CreateUserResponse, *vel.Error) {
    // req is automatically populated from JSON request body
    if req.Name == "" {
        return CreateUserResponse{}, &vel.Error{
            Code:    "MISSING_NAME",
            Message: "Name is required",
        }
    }

    return CreateUserResponse{
        ID:    generateID(),
        Name:  req.Name,
        Email: req.Email,
    }, nil
}
```

The framework uses Go's standard `encoding/json` package for unmarshaling, so all standard JSON tags and behaviors apply.

### Query Parameter Handling

For GET requests, vel automatically decodes query parameters using struct tags:

```go
type SearchRequest struct {
    Query    string `schema:"q"`
    Limit    int    `schema:"limit"`
    Page     int    `schema:"page"`
    Category string `schema:"category"`
}

type SearchResponse struct {
    Results []ResultType `json:"results"`
    Total   int          `json:"tota"`
}

func SearchHandler(ctx context.Context, req SearchRequest) (SearchResponse, *vel.Error) {
    // GET /search?q=golang&limit=10&page=2&category=tutorial
    // req.Query = "golang"
    // req.Limit = 10
    // req.Page = 2
    // req.Category = "tutorial"

    results := performSearch(req.Query, req.Limit, req.Page, req.Category)
    return SearchResponse{
        Results: results,
        Total:   len(results),
    }, nil
}
```

**Key features:**

- Uses the `schema` struct tag to map query parameters to struct fields, check [gorilla/scheme](https://github.com/gorilla/schema) for more info.
- Supports automatic type conversion (string to int, bool, etc.)
- Missing parameters result in zero values
- Invalid type conversions return `FAILED_DECODING_QUERY` error code (more about error codes later)

### Middlewares

Apply middleware for cross-cutting concerns:

```go
func CORSMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusNoContent)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func LoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        next.ServeHTTP(w, r)

        log.Printf("Request %s %s took %v", r.Method, r.URL.Path, time.Since(start))
    })
}

// Apply middleware to router
router := vel.NewRouter()
router.Use(CORSMiddleware)
router.Use(LoggingMiddleware)
```

## Empty request/response

vel supports empty structs as data types in order to skip request/response marshalling.
Use `struct{}` for handlers that don't need input or output:

```go
// No input or output
func HealthCheckHandler(ctx context.Context, _ struct{}) (struct{}, *vel.Error) {
    return struct{}{}, nil
}

// Input only
func LogEventHandler(ctx context.Context, req LogEventRequest) (struct{}, *vel.Error) {
    return struct{}{}, nil
}

// Output only
func GetStatusHandler(ctx context.Context, _ struct{}) (StatusResponse, *vel.Error) {
    return StatusResponse{
        Status: "healthy",
        Uptime: getUptime(),
    }, nil
}
```
