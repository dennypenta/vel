---
title: Error handling
description: error handling
---

## Error Handling

vel provides structured error handling with configurable status code mapping and error processing.
The errors is a subject to change or extend to support more standards such as [RFC9457](https://datatracker.ietf.org/doc/html/rfc9457).

### Error Type Structure

```go
type Error struct {
    // Code is supposed to specify the exact error code use case
    // so a client knows exactly how to handle it
    Code    string            `json:"code"`
    // Message is the opposite to Code, it describes a piece of code
    // when neither a server nor a client know how to handle this case,
    // in order to catch it alter in the logs and come with a solution later
    Message string            `json:"message,omitempty"`
    // Meta adds a key-value pair with any necessary information to handle the error
    Meta    map[string]string `json:"meta,omitempty"`
    // Err is for internal only purpose,
    // e.g. logging, it's never exposed to the client
    Err     error             `json:"-"`
}
```

In the current model it's recommended to use either Code or Message + Err.

### Creating Errors

```go
func ValidationHandler(ctx context.Context, req UserRequest) (UserResponse, *vel.Error) {
    if req.Email == "" {
        return UserResponse{}, &vel.Error{
            Code:    "MISSING_EMAIL",
        }
    }

    if !isValidEmail(req.Email) {
        return UserResponse{}, &vel.Error{
            Code:    "INVALID_EMAIL",
            Meta: map[string]string{
                "field": "email",
                "value": req.Email,
            },
        }
    }

    return UserResponse{}, nil
}
```

### Built-in Error Codes

vel includes built-in error codes for common framework errors:

- `FAILED_DECODING_QUERY`: Query parameter decoding failure
- `FAILED_DECODING_REQUEST_BODY`: Request body JSON decoding failure
- `FAILED_ENCODING_RESPONSE_BODY`: Response body JSON encoding failure

These errors are automatically generated when the framework encounters marshaling/unmarshaling issues.
