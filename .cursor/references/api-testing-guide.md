# API Testing Guide

## Table of Contents
- [Test File Structure](#test-file-structure)
- [Handler Test Pattern](#handler-test-pattern)
- [HTTP Request Setup](#http-request-setup)
- [Mock Service Setup](#mock-service-setup)
- [Response Comparison](#response-comparison)
- [Custom Matchers](#custom-matchers)
- [Helper Functions](#helper-functions)
- [Test Case Patterns](#test-case-patterns)

## Test File Structure

### Package and Imports
```go
package api //nolint:testpackage // for test internal function

import (
    "bytes"
    "context"
    "errors"
    "flag"
    "io"
    "log"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/go-chi/chi/v5"
    "github.com/google/go-cmp/cmp"
    "github.com/google/go-cmp/cmp/cmpopts"
    "github.com/google/uuid"
    "go.uber.org/goleak"
    "go.uber.org/mock/gomock"

    "vwap/internal/api/middleware"
    "vwap/internal/domain"
    "vwap/internal/domain/mocks"
    "vwap/internal/httpwrap"
)
```

### Static Error Variables
Define static test errors at package level:
```go
var (
    errDatabase     = errors.New("database error")
    errEventPublish = errors.New("event publish error")
)
```

## Handler Test Pattern

```go
func Test_handlerName(t *testing.T) {
    t.Parallel()

    userID := uuid.New()
    resourceID := uuid.New()

    tests := []struct {
        name        string
        setupReq    func() *http.Request
        setupSvc    func(ctrl *gomock.Controller) *mocks.MockService
        wantResp    *httpwrap.Response
        wantErrResp *httpwrap.ErrorResponse
    }{
        {
            name: "success",
            setupReq: func() *http.Request {
                // Create test request
            },
            setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
                // Setup mock service
            },
            wantResp: &httpwrap.Response{
                StatusCode: http.StatusOK,
                Body:       &ExpectedResponse{},
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            ctrl := gomock.NewController(t)
            req := tt.setupReq()

            gotResp, gotErrResp := handlerName(tt.setupSvc(ctrl))(req)

            // Assertions
        })
    }
}
```

### Test Fields
- `name`: Test case name (follow naming conventions)
- `setupReq`: Function that creates and configures `*http.Request`
- `setupSvc`: Function that creates mock service with expectations
- `wantResp`: Expected successful response (`*httpwrap.Response`)
- `wantErrResp`: Expected error response (`*httpwrap.ErrorResponse`)

**Important**: Only one of `wantResp` or `wantErrResp` should be non-nil per test case.

## HTTP Request Setup

### Simple Requests (No URL Parameters)
```go
setupReq: func() *http.Request {
    req := httptest.NewRequest(http.MethodGet, "/resources", nil)
    req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
    return req
}
```

### Requests with URL Parameters
Use helper function:
```go
setupReq: func() *http.Request {
    req := createRequestWithURLParam(
        http.MethodGet,
        fmt.Sprintf("/resources/%s", resourceID.String()),
        resourceID.String(),
    )
    req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
    return req
}
```

### Requests with JSON Body
```go
setupReq: func() *http.Request {
    req := httptest.NewRequest(http.MethodPost, "/resources", nil)

    req.Body = io.NopCloser(bytes.NewReader([]byte(`{
        "name": "test",
        "description": "test description",
        "tags": ["tag1", "tag2"]
    }`)))

    req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
    return req
}
```

### Requests with Multiple URL Parameters
```go
setupReq: func() *http.Request {
    req := httptest.NewRequest(
        http.MethodGet,
        fmt.Sprintf("/resources/%s/items/%s", resourceID.String(), itemID.String()),
        nil,
    )

    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("resource_id", resourceID.String())
    rctx.URLParams.Add("item_id", itemID.String())
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

    return req
}
```

## Mock Service Setup

### Simple Mock Setup
```go
setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
    svc := mocks.NewMockService(ctrl)

    svc.EXPECT().
        GetResource(gomock.Any(), resourceID).
        Return(&domain.Resource{ID: resourceID, Name: "test"}, nil)

    return svc
}
```

### Mock with Specific Parameter Validation
```go
setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
    svc := mocks.NewMockService(ctrl)

    svc.EXPECT().
        CreateResource(gomock.Any(), &domain.CreateParams{
            UserID: userID,
            Name:   "test",
        }).
        Return(&domain.Resource{ID: resourceID}, nil)

    return svc
}
```

### Mock with Multiple Calls
```go
setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
    svc := mocks.NewMockService(ctrl)

    svc.EXPECT().
        ListResourcesByUserID(gomock.Any(), userID).
        Return([]*domain.Resource{}, nil)

    svc.EXPECT().
        CreateResource(gomock.Any(), gomock.Any()).
        Return(nil)

    return svc
}
```

### Mock Returning Errors
```go
setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
    svc := mocks.NewMockService(ctrl)

    svc.EXPECT().
        GetResource(gomock.Any(), resourceID).
        Return(nil, domain.ErrNotFound)

    return svc
}
```

## Response Comparison

### Exact Comparison
```go
if !cmp.Equal(gotResp, tt.wantResp) {
    t.Errorf("handler() resp = %v, want %v, diff %v", gotResp, tt.wantResp, cmp.Diff(gotResp, tt.wantResp))
}

if !cmp.Equal(gotErrResp, tt.wantErrResp, cmpopts.EquateErrors()) {
    t.Errorf("handler() err resp = %v, want %v, diff %v", gotErrResp, tt.wantErrResp, cmp.Diff(gotErrResp, tt.wantErrResp, cmpopts.EquateErrors()))
}
```

### Comparison with Custom Options
For responses with dynamic fields (UUIDs, timestamps):
```go
opts := cmp.Options{
    cmp.Comparer(func(a, b *CreateResourceResponse) bool {
        return a != nil && b != nil && a.ID != uuid.Nil && b.ID != uuid.Nil
    }),
}

if !cmp.Equal(gotResp, tt.wantResp, opts) {
    t.Errorf("handler() resp = %v, want %v, diff %v", gotResp, tt.wantResp, cmp.Diff(gotResp, tt.wantResp, opts))
}
```

### Error Response Comparison
Always use `cmpopts.EquateErrors()` when comparing error responses:
```go
if !cmp.Equal(gotErrResp, tt.wantErrResp, cmpopts.EquateErrors()) {
    t.Errorf("handler() err resp = %v, want %v, diff %v", gotErrResp, tt.wantErrResp, cmp.Diff(gotErrResp, tt.wantErrResp, cmpopts.EquateErrors()))
}
```

## Custom Matchers

### When to Use Custom Matchers
Use custom matchers for parameter validation when:
1. Parameters contain dynamic values (timestamps, generated UUIDs)
2. Parameters are complex structs with many fields
3. You need to validate specific fields while ignoring others

### Custom Matcher Implementation
```go
type createChannelParamsMatcher struct {
    params *channel.CreateChannelParams
}

func (m createChannelParamsMatcher) Matches(x interface{}) bool {
    p, ok := x.(*channel.CreateChannelParams)
    if !ok {
        return false
    }

    return p.ID != uuid.Nil &&
        p.UserID == m.params.UserID &&
        p.Name == m.params.Name &&
        !p.CreatedAt.IsZero()
}

func (m createChannelParamsMatcher) String() string {
    return fmt.Sprintf("%+v", m.params)
}
```

### Using Custom Matcher
```go
svc.EXPECT().
    CreateChannel(gomock.Any(), createChannelParamsMatcher{params: &channel.CreateChannelParams{
        UserID:      userID,
        Name:        "test",
        Description: "description",
    }}).
    Return(nil)
```

## Helper Functions

### URL Parameter Helper
Define this helper at the end of test file:
```go
func createRequestWithURLParam(method, url, paramValue string) *http.Request {
    req := httptest.NewRequest(method, url, nil)
    rctx := chi.NewRouteContext()
    rctx.URLParams.Add("id", paramValue)
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

    return req
}
```

### Pointer Helper Functions
```go
func stringPtr(s string) *string {
    return &s
}

func decimalPtr(d decimal.Decimal) *decimal.Decimal {
    return &d
}

func intPtr(i int) *int {
    return &i
}
```

## Test Case Patterns

### Success Case
```go
{
    name: "success",
    setupReq: func() *http.Request {
        req := httptest.NewRequest(http.MethodGet, "/resources", nil)
        req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        svc := mocks.NewMockService(ctrl)

        svc.EXPECT().
            GetResources(gomock.Any(), userID).
            Return([]*domain.Resource{{ID: resourceID}}, nil)

        return svc
    },
    wantResp: &httpwrap.Response{
        StatusCode: http.StatusOK,
        Body: &ListResourcesResponse{
            Resources: []*ResourceResponse{{ID: resourceID}},
        },
    },
}
```

### Not Found Error Case
```go
{
    name: "error - resource not found",
    setupReq: func() *http.Request {
        req := createRequestWithURLParam(
            http.MethodGet,
            fmt.Sprintf("/resources/%s", resourceID.String()),
            resourceID.String(),
        )
        req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        svc := mocks.NewMockService(ctrl)

        svc.EXPECT().
            GetResource(gomock.Any(), resourceID).
            Return(nil, domain.ErrNotFound)

        return svc
    },
    wantErrResp: &httpwrap.ErrorResponse{
        StatusCode: http.StatusNotFound,
        ErrorMsg:   domain.ErrNotFound.Error(),
        Err:        domain.ErrNotFound,
    },
}
```

### Service Error Case
```go
{
    name: "error - service fails",
    setupReq: func() *http.Request {
        req := httptest.NewRequest(http.MethodPost, "/resources", nil)
        req.Body = io.NopCloser(bytes.NewReader([]byte(`{"name": "test"}`)))
        req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        svc := mocks.NewMockService(ctrl)

        svc.EXPECT().
            CreateResource(gomock.Any(), gomock.Any()).
            Return(nil, errDatabase)

        return svc
    },
    wantErrResp: &httpwrap.ErrorResponse{
        StatusCode: http.StatusInternalServerError,
        ErrorMsg:   "database error",
        Err:        errDatabase,
    },
}
```

### Invalid Parameter Case
```go
{
    name: "error - invalid id parameter",
    setupReq: func() *http.Request {
        req := createRequestWithURLParam(
            http.MethodGet,
            "/resources/invalid-uuid",
            "invalid-uuid",
        )
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        // No expectations - handler should fail before calling service
        return mocks.NewMockService(ctrl)
    },
    wantErrResp: &httpwrap.ErrorResponse{
        StatusCode: http.StatusBadRequest,
        ErrorMsg:   "invalid param: id",
    },
}
```

### Idempotent Delete Case
```go
{
    name: "success - resource not found (idempotent)",
    setupReq: func() *http.Request {
        req := createRequestWithURLParam(
            http.MethodDelete,
            fmt.Sprintf("/resources/%s", resourceID.String()),
            resourceID.String(),
        )
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        svc := mocks.NewMockService(ctrl)

        // DELETE is idempotent - return success even if not found
        svc.EXPECT().
            DeleteResource(gomock.Any(), resourceID).
            Return(domain.ErrNotFound)

        return svc
    },
    wantResp: &httpwrap.Response{
        StatusCode: http.StatusOK,
    },
}
```

### Empty List Case
```go
{
    name: "success - empty list",
    setupReq: func() *http.Request {
        req := httptest.NewRequest(http.MethodGet, "/resources", nil)
        req = req.WithContext(middleware.SetUserID(req.Context(), userID.String()))
        return req
    },
    setupSvc: func(ctrl *gomock.Controller) *mocks.MockService {
        svc := mocks.NewMockService(ctrl)

        svc.EXPECT().
            ListResources(gomock.Any(), userID).
            Return([]*domain.Resource{}, nil)

        return svc
    },
    wantResp: &httpwrap.Response{
        StatusCode: http.StatusOK,
        Body: &ListResourcesResponse{
            Resources: []*ResourceResponse{},
        },
    },
}
```

## Best Practices

1. **Use setupReq and setupSvc functions**: Keep test structure consistent
2. **Preallocate test data**: Define UUIDs and constants at function level
3. **Use helper functions**: Avoid duplicating URL param setup code
4. **Use custom matchers**: For complex parameter validation
5. **Test all error paths**: Cover domain errors, validation errors, service errors
6. **Use static error variables**: Define at package level for reuse
7. **Compare with proper options**: Use cmp.Options for dynamic fields
8. **Always use cmpopts.EquateErrors()**: When comparing error responses
9. **Test edge cases**: Empty lists, multiple items, nil values
10. **Keep tests focused**: One scenario per test case
