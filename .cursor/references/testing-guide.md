# Go Testing Guide

## Table of Contents
- [Test File Structure](#test-file-structure)
- [Table-Driven Tests](#table-driven-tests)
- [Testing with Mocks](#testing-with-mocks-gouberorgmock)
- [Error Handling in Tests](#error-handling-in-tests)
- [Assertions and Comparisons](#assertions-and-comparisons)
- [Test File Organization](#test-file-organization)
- [Mock Generation](#mock-generation-with-gouberorgmock)

## Test File Structure

### Package Declaration
```go
package service //nolint:testpackage // for test internal function
```

### TestMain for Setup and Cleanup
Always include TestMain for goroutine leak detection:
```go
func TestMain(m *testing.M) {
    leak := flag.Bool("leak", true, "enable goleak checks")
    flag.Parse()

    code := m.Run()

    if *leak {
        err := goleak.Find()
        if err != nil {
            log.Fatalf("goleak detected leaks: %v", err)
        }
    }

    os.Exit(code)
}
```

## Table-Driven Tests

### Test Structure
```go
func TestFunctionName(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name    string
        // input fields
        want    expectedType
        wantErr bool
    }{
        // test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            // test logic
        })
    }
}
```

### Test Case Naming
- **`"success"`** - For normal successful case (the happy path)
- **`"success - description"`** - For successful cases other than the normal case
- **`"error - description"`** - For error cases

Examples:
```go
"success"
"success - with cache hit"
"success - with empty wallet"
"error - wallet not found"
"error - insufficient balance"
```

## Testing with Mocks (go.uber.org/mock)

### Mock Setup Pattern
```go
tests := []struct {
    name      string
    setupRepo func(ctrl *gomock.Controller) *mocks.MockRepository
    params    *channel.Params
    want      *channel.Result
    wantErr   bool
}{
    {
        name: "success",
        setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
            repo := mocks.NewMockRepository(ctrl)

            // Good: Use Any() only for context, check other parameters
            repo.EXPECT().
                MethodName(gomock.Any(), "expected-id").
                Return(expectedResult, nil)

            return repo
        },
        params: &channel.Params{...},
        want:   &channel.Result{...},
    },
}
```

### Mock Controller
Create gomock.Controller inside each subtest:
```go
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()

        ctrl := gomock.NewController(t)

        service := New(tt.setupRepo(ctrl), nil, nil)
        got, err := service.Method(t.Context(), tt.params)
        // assertions
    })
}
```

### gomock Matchers

**General Rule:**
- Use `gomock.Any()` ONLY for values that cannot be verified (e.g., `context.Context`)
- For simple equality, pass values directly
- Use custom matchers for complex validation logic

**Any/Equal Matchers:**
```go
// Good: Only use Any() for context
repo.EXPECT().
    GetUser(gomock.Any(), "user-123").
    Return(&User{ID: "user-123"}, nil)

// Bad: Using Any() for verifiable parameter
repo.EXPECT().
    GetUser(gomock.Any(), gomock.Any()).  // Should check user ID!
    Return(&User{}, nil)
```

**Type Matchers:**
```go
repo.EXPECT().
    CreateUser(gomock.Any(), gomock.AssignableToTypeOf(&User{})).
    Return(nil)
```

**Nil Matchers:**
```go
repo.EXPECT().
    Process(gomock.Any(), gomock.Nil()).
    Return(nil)

repo.EXPECT().
    Process(gomock.Any(), gomock.Not(gomock.Nil())).
    Return(nil)
```

**Custom Matchers:**
```go
uuidMatcher := gomock.Cond(func(x interface{}) bool {
    id, ok := x.(uuid.UUID)
    return ok && id != uuid.Nil
})

repo.EXPECT().
    GetChannel(gomock.Any(), uuidMatcher, uuidMatcher).
    Return(&channel.Channel{}, nil)
```

**When to Use gomock.Any():**
- `context.Context` - Usually not verifiable in tests
- Timestamps/time values that are generated dynamically
- System-generated values (UUIDs if generated in function)

### Call Modifiers

**Return Values:**
```go
m.EXPECT().GetCount().Return(5)
m.EXPECT().GetUser(gomock.Any()).Return(&User{}, nil)
m.EXPECT().GetUser(gomock.Any()).Return(nil, ErrNotFound)
```

**Call Frequency:**
```go
m.EXPECT().Method()                  // Exactly once (default)
m.EXPECT().Method().AnyTimes()       // Any number of times
m.EXPECT().Method().Times(3)         // Specific number
m.EXPECT().Method().MinTimes(1)      // At least
m.EXPECT().Method().MaxTimes(5)      // At most
```

**Do/DoAndReturn for Side Effects:**
```go
m.EXPECT().
    Process(gomock.Any(), gomock.Any()).
    Do(func(ctx context.Context, data *Data) {
        data.Processed = true
    })

m.EXPECT().
    Calculate(10).
    DoAndReturn(func(x int) (int, error) {
        return x * 2, nil
    })
```

**Call Order:**
```go
gomock.InOrder(
    m.EXPECT().First().Return(nil),
    m.EXPECT().Second().Return(nil),
    m.EXPECT().Third().Return(nil),
)
```

## Error Handling in Tests

### Checking for Errors
```go
got, err := service.Method(t.Context(), tt.params)
if err != nil {
    if !tt.wantErr {
        t.Errorf("Method() failed: %v", err)
    }
}
```

### Verifying Specific Errors
Add an `err error` field to the test struct:
```go
tests := []struct {
    name      string
    setupRepo func(ctrl *gomock.Controller) *mocks.MockRepository
    params    *channel.Params
    want      *channel.Result
    wantErr   bool
    err       error  // Expected error for verification
}{
    {
        name: "error - not found",
        setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
            repo := mocks.NewMockRepository(ctrl)
            repo.EXPECT().
                GetChannel(gomock.Any(), id).
                Return(nil, repository.ErrNotFound)
            return repo
        },
        params:  &channel.Params{ID: id},
        wantErr: true,
        err:     channel.ErrNotFound,
    },
}

// In test execution:
if err != nil {
    if !tt.wantErr {
        t.Errorf("GetChannel() wantErr = %v, got = %v", tt.wantErr, err)
    }

    if !errors.Is(err, tt.err) {
        t.Errorf("GetChannel() error = %v, want %v", err, tt.err)
    }
}
```

### Static Error Variables in Tests
Define static error variables at package level:
```go
package service //nolint:testpackage

var errInternal = errors.New("internal error")

func TestService_Method(t *testing.T) {
    // Use errInternal in test cases
}
```

## Assertions and Comparisons

### Use go-cmp for comparisons
```go
if !cmp.Equal(got, tt.want) {
    t.Errorf("Method() = %v, want %v, diff %v", got, tt.want, cmp.Diff(got, tt.want))
}
```

## Test Constants
Define reusable test constants at the top of the test function:
```go
func TestService_Method(t *testing.T) {
    t.Parallel()

    const (
        solAddress  = "So11111111111111111111111111111111111111112"
        bonkAddress = "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"
    )

    baseTime := time.Unix(1700000000, 0)

    tests := []struct {
        // ...
    }
}
```

## Helper Functions
Create helper functions for repetitive test setup:
```go
func mustNewDecimalFromStr(value string) decimal.Decimal {
    d, err := decimal.NewFromString(value)
    if err != nil {
        panic(err)
    }
    return d
}
```

## Test File Organization

### File Naming Conventions
```
internal/
  domain/
    service.go           # Source file
    service_test.go      # Main test file (contains TestMain)
    repository.go        # Source file
    repository_test.go   # Repository-specific tests
```

### When to Split Test Files
- **Single file exceeds 500+ lines**
- **Testing different concerns**
- **Different test patterns** (unit vs integration vs benchmarks)

### TestMain Placement
- **Place TestMain in the main test file** - Usually `service_test.go`
- **Other test files omit TestMain** - Avoid duplicate declarations

## Context Usage

Always use `t.Context()` instead of `context.Background()`:
```go
// Good: Use t.Context()
got, err := s.calculateWalletPNL(t.Context(), tt.params)

// Bad: Using context.Background()
got, err := s.calculateWalletPNL(context.Background(), tt.params)
```

## Mock Generation with go.uber.org/mock

### Generating Mocks with go:generate
```go
//go:generate mockgen -source=repository.go -destination=../mocks/repository.go -package=mocks
type Repository interface {
    CreateChannel(ctx context.Context, channel *channel.Channel) error
    GetChannel(ctx context.Context, id, userID uuid.UUID) (*channel.Channel, error)
}
```

### mockgen Command Flags
- `-source`: Source file containing interfaces to mock
- `-destination`: Output file for generated mocks
- `-package`: Package name for generated mocks (typically `mocks`)
- `-typed`: Generate type-safe Return/Do/DoAndReturn (recommended)

### Generating Mocks
```bash
go generate ./...
# Or for specific package:
go generate ./internal/domain/repository
```

## Best Practices Summary

1. Always use table-driven tests for functions with multiple scenarios
2. Always enable parallel execution with `t.Parallel()`
3. Always check for goroutine leaks with goleak in TestMain
4. Follow naming convention: `"success"`, `"success - description"`, `"error - description"`
5. Always use cmp.Equal for deep equality checks with diff output
6. Always test both success and error paths
7. Use mocks via setup functions in test struct for flexibility
8. Define constants for reusable test data
9. Check mock parameters whenever possible - Only use `gomock.Any()` for unverifiable values
10. Create `gomock.Controller` per subtest for test isolation
11. Set mock expectations before calling system under test
12. Use static error variables instead of dynamic error creation
13. Verify specific errors using `err` field and `errors.Is()`
