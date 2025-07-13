---
title: OpenAPI Generation
description: spec generation
---

## OpenAPI Generation

Similar to a client generation script you can define OpenAPI generation

```go
func main() {
    router := myapp.NewRouter()

    err := gen.GenerateOpenAPIToFile(router, "./api-spec.yaml", "My API", "1.0.0")
    if err != nil {
        panic(err)
    }
}
```

### Custom Annotations

Not the entire spec can be extracted from the data types, so vel provides capabilities to define in details the headers, errors and many more
Add custom documentation through the `Spec` field:

```go
vel.RegisterPost(router, "users", CreateUserHandler).SetSpec(openapi.Spec{
    Description: "Create a new user account",
    RequestHeaders: openapi.KeyValueSpec{
        Key:          "X-API-Key",
        ValueType:    openapi.String,
        Description:  "API key for authentication",
        ValueExample: "api_key_12345",
        Validation: openapi.Validation{
            Required: true,
        },
    },
    ResponseHeaders: openapi.KeyValueSpec{
        Key:          "X-Rate-Limit",
        ValueType:    openapi.String,
        Description:  "Rate limit information",
        ValueExample: "100",
    },
    Errors: []openapi.ErrorSpec{
        {
            Code:        "VALIDATION_ERROR",
            Description: "Request validation failed",
            Meta: []openapi.KeyValueSpec{
                {
                    Key:         "field",
                    ValueType:   openapi.String,
                    Description: "Field that failed validation",
                },
            },
        },
    },
})
```
