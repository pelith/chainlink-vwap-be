# API Implementation Guide

## Table of Contents
- [Core Principles](#core-principles)
- [File Structure](#file-structure)
- [Handler Function Pattern](#handler-function-pattern)
- [Request/Response DTOs](#requestresponse-dtos)
- [Error Handling](#error-handling)
- [URL and Query Parameters](#url-and-query-parameters)
- [Request Body Handling](#request-body-handling)
- [Response Construction](#response-construction)
- [AddRoutes Function](#addroutes-function)
- [Data Transformation](#data-transformation)

## Core Principles

1. **Thin API Layer**: API handlers should be thin - delegate business logic to service layer
2. **Clear Error Handling**: Use consistent error response patterns
3. **Type Safety**: Use strongly typed requests and responses
4. **Early Returns**: Return errors immediately, avoid deep nesting
5. **Consistent Patterns**: Follow established conventions for maintainability

## File Structure

### API Package Organization
```
internal/
  domain/
    api/
      api.go          // AddRoutes + handler functions + DTOs
      api_test.go     // Handler tests
```

### api.go Structure
```go
package api

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "vwap/internal/httpwrap"
    "vwap/internal/domain"
)

// Constants (if any)
const (
    maxResultsLimit = 100
)

// AddRoutes function (must come first)
func AddRoutes(r chi.Router, svc domain.Service) {
    r.Post("/resource", httpwrap.Handler(createResource(svc)))
    r.Get("/resource/{id}", httpwrap.Handler(getResource(svc)))
}

// Request/Response DTOs
type CreateResourceRequest struct {
    Name string `json:"name"`
}

type ResourceResponse struct {
    ID   uuid.UUID `json:"id"`
    Name string    `json:"name"`
}

// Handler functions (lowercase, factory functions returning httpwrap.HandlerFunc)
func createResource(svc domain.Service) httpwrap.HandlerFunc {
    return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
        // Handler implementation
    }
}
```

## Handler Function Pattern

### Handler Signature
Always use factory functions that return `httpwrap.HandlerFunc`:
```go
// Good: Factory function pattern
func getResource(svc domain.Service) httpwrap.HandlerFunc {
    return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
        // Implementation
    }
}

// Bad: Direct handler
func getResource(svc domain.Service, w http.ResponseWriter, r *http.Request) {
    // Don't do this
}
```

### Handler Implementation Flow

**Standard flow (in order):**
1. Parse URL parameters
2. Parse request body (if needed)
3. Extract context values (userID, etc.)
4. Call service layer
5. Handle errors with early returns
6. Return response

```go
func createResource(svc domain.Service) httpwrap.HandlerFunc {
    return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
        // 1. Parse URL parameters (if any)
        idStr := chi.URLParam(r, "id")
        id, err := uuid.Parse(idStr)
        if err != nil {
            return nil, httpwrap.NewInvalidParamErrorResponse("id")
        }

        // 2. Parse request body
        var req CreateResourceRequest
        if errResp := httpwrap.BindBody(r, &req); errResp != nil {
            return nil, errResp
        }

        // 3. Extract context
        ctx := r.Context()
        userID := middleware.GetUserID(r)

        // 4. Call service
        result, err := svc.CreateResource(ctx, &domain.CreateParams{
            UserID: uuid.MustParse(userID),
            Name:   req.Name,
        })
        if err != nil {
            // 5. Handle errors
            return nil, &httpwrap.ErrorResponse{
                StatusCode: http.StatusInternalServerError,
                ErrorMsg:   err.Error(),
                Err:        err,
            }
        }

        // 6. Return success response
        return &httpwrap.Response{
            StatusCode: http.StatusOK,
            Body: &ResourceResponse{
                ID:   result.ID,
                Name: result.Name,
            },
        }, nil
    }
}
```

## Request/Response DTOs

### Naming Conventions
- **Request types**: Use suffix `Request` (e.g., `CreateChannelRequest`)
- **Response types**: Use suffix `Response` (e.g., `ChannelResponse`)
- **Internal DTOs**: Use descriptive names without suffix (e.g., `Token`, `Trades`)
- **All types**: Exported (start with capital letter)

### Request Structures
```go
// Create requests
type CreateResourceRequest struct {
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
}

// Update requests (use pointers for optional fields)
type UpdateResourceRequest struct {
    Name        *string `json:"name"`
    Description *string `json:"description"`
    MaxLimit    *int    `json:"max_limit"`
}
```

### Response Structures
```go
// Single resource response
type ResourceResponse struct {
    ID          uuid.UUID        `json:"id"`
    Name        string           `json:"name"`
    CreatedAt   time.Time        `json:"created_at"`
    MaxLimit    *decimal.Decimal `json:"max_limit"` // Pointer for optional
}

// List response
type ListResourcesResponse struct {
    Resources []*ResourceResponse `json:"resources"`
    Total     int                 `json:"total"`
}
```

## Error Handling

### Error Response Pattern
Always use early returns for errors:
```go
result, err := svc.Method(ctx, params)
if err != nil {
    return nil, &httpwrap.ErrorResponse{
        StatusCode: http.StatusInternalServerError,
        ErrorMsg:   err.Error(),
        Err:        err,
    }
}
```

### Domain-Specific Errors
Use `errors.Is()` to check for domain-specific errors:
```go
result, err := svc.GetResource(ctx, id)
if err != nil {
    if errors.Is(err, domain.ErrNotFound) {
        return nil, &httpwrap.ErrorResponse{
            StatusCode: http.StatusNotFound,
            ErrorMsg:   err.Error(),
            Err:        err,
        }
    }

    return nil, &httpwrap.ErrorResponse{
        StatusCode: http.StatusInternalServerError,
        ErrorMsg:   err.Error(),
        Err:        err,
    }
}
```

### HTTP Status Code Mapping
```go
// 400 Bad Request - Invalid input, validation errors
if errors.Is(err, domain.ErrInvalidInput) {
    return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusBadRequest, ...}
}

// 404 Not Found - Resource not found
if errors.Is(err, domain.ErrNotFound) {
    return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusNotFound, ...}
}

// 409 Conflict - Resource conflict
if errors.Is(err, domain.ErrConflict) {
    return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusConflict, ...}
}

// 500 Internal Server Error - Unexpected errors
return nil, &httpwrap.ErrorResponse{StatusCode: http.StatusInternalServerError, ...}
```

### Helper Functions for Errors
```go
// Invalid parameter
return nil, httpwrap.NewInvalidParamErrorResponse("id")

// Invalid body (use BindBody which handles this)
if errResp := httpwrap.BindBody(r, &req); errResp != nil {
    return nil, errResp
}
```

## URL and Query Parameters

### Parsing URL Parameters
```go
// Simple parameter
idStr := chi.URLParam(r, "id")

// Parse and validate UUID
id, err := uuid.Parse(idStr)
if err != nil {
    return nil, httpwrap.NewInvalidParamErrorResponse("id")
}

// Parse other types
limitStr := chi.URLParam(r, "limit")
limit, err := strconv.Atoi(limitStr)
if err != nil {
    return nil, httpwrap.NewInvalidParamErrorResponse("limit")
}
```

### Query Parameters
```go
query := r.URL.Query()

// Optional parameters with defaults
limitStr := query.Get("limit")
limit := 50 // default
if limitStr != "" {
    parsedLimit, err := strconv.Atoi(limitStr)
    if err != nil {
        return nil, httpwrap.NewInvalidParamErrorResponse("limit")
    }
    limit = parsedLimit
}

// Boolean parameters
includeDeleted := query.Get("include_deleted") == "true"

// Array parameters
tags := query["tags"] // []string
```

## Request Body Handling

### JSON Decoding
Always use `httpwrap.BindBody` for request body parsing:
```go
// Good: Use httpwrap.BindBody
var req CreateResourceRequest
if errResp := httpwrap.BindBody(r, &req); errResp != nil {
    return nil, errResp
}

// Bad: Manual decoding
var req CreateResourceRequest
err := json.NewDecoder(r.Body).Decode(&req)
```

## Response Construction

### Success Responses
```go
// Response with body
return &httpwrap.Response{
    StatusCode: http.StatusOK,
    Body: &ResourceResponse{ID: resource.ID, Name: resource.Name},
}, nil

// Response without body (for updates/deletes)
return &httpwrap.Response{
    StatusCode: http.StatusOK,
    Body:       nil,
}, nil

// Created response
return &httpwrap.Response{
    StatusCode: http.StatusCreated,
    Body: &CreateResourceResponse{ID: resourceID},
}, nil
```

## AddRoutes Function

### Route Registration Pattern
```go
func AddRoutes(r chi.Router, svc domain.Service) {
    // Resource CRUD
    r.Post("/resources", httpwrap.Handler(createResource(svc)))
    r.Get("/resources", httpwrap.Handler(listResources(svc)))
    r.Get("/resources/{id}", httpwrap.Handler(getResource(svc)))
    r.Put("/resources/{id}", httpwrap.Handler(updateResource(svc)))
    r.Delete("/resources/{id}", httpwrap.Handler(deleteResource(svc)))

    // Sub-resources
    r.Post("/resources/{id}/items", httpwrap.Handler(addItems(svc)))
    r.Get("/resources/{id}/items", httpwrap.Handler(listItems(svc)))

    // Actions
    r.Post("/resources/{id}/activate", httpwrap.Handler(activateResource(svc)))
}
```

### HTTP Method Selection
- **POST**: Create new resources, trigger actions
- **GET**: Retrieve resources (single or list)
- **PUT**: Update entire resource, add to collection
- **DELETE**: Remove resources

## Data Transformation

### Domain to API DTOs
```go
channels := make([]*ChannelResponse, 0, len(domainChannels))
for _, ch := range domainChannels {
    channels = append(channels, &ChannelResponse{
        ID:        ch.ID,
        Name:      ch.Name,
        CreatedAt: ch.CreatedAt,
    })
}

return &httpwrap.Response{
    StatusCode: http.StatusOK,
    Body: &ListChannelsResponse{Channels: channels},
}, nil
```

### Handling Optional Fields
```go
var maxLimit *decimal.Decimal
if ch.MaxLimit != nil {
    maxLimit = ch.MaxLimit
}

response := &ChannelResponse{
    ID:       ch.ID,
    MaxLimit: maxLimit,
}
```

## Idempotency Patterns

### DELETE Operations
DELETE should be idempotent - return success even if resource doesn't exist:
```go
func deleteResource(svc domain.Service) httpwrap.HandlerFunc {
    return func(r *http.Request) (*httpwrap.Response, *httpwrap.ErrorResponse) {
        // Parse ID...

        err := svc.DeleteResource(ctx, id)
        // Ignore NotFound errors for idempotency
        if err != nil && !errors.Is(err, domain.ErrNotFound) {
            return nil, &httpwrap.ErrorResponse{
                StatusCode: http.StatusInternalServerError,
                ErrorMsg:   err.Error(),
                Err:        err,
            }
        }

        return &httpwrap.Response{StatusCode: http.StatusOK, Body: nil}, nil
    }
}
```

## Best Practices

1. **Keep handlers thin**: Business logic belongs in service layer
2. **Early returns**: Return errors immediately, avoid nesting
3. **Consistent error handling**: Use the same pattern across all handlers
4. **Type safety**: Always validate and parse parameters
5. **Preallocate slices**: Use `make([]T, 0, capacity)` when size is known
6. **Use helpers**: Leverage `httpwrap.BindBody`, `httpwrap.NewInvalidParamErrorResponse`
7. **Clear naming**: Use descriptive DTO names that match their purpose
8. **Handle nil values**: Check for nil before dereferencing pointers
9. **UTC timestamps**: Always use `time.Now().UTC()` for consistency
