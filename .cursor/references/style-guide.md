# Go Style Guide

## Table of Contents
- [Core Principles](#core-principles)
- [Domain-Driven Design](#domain-driven-design)
- [Naming](#naming)
- [Variables & Declarations](#variables--declarations)
- [Error Handling](#error-handling)
- [Context Handling](#context-handling)
- [Interfaces & Types](#interfaces--types)
- [Concurrency](#concurrency)
- [Memory & Performance](#memory--performance)
- [Logging](#logging)

## Core Principles

1. **Clarity** > cleverness: Code should be easy to understand
2. **Simplicity** > sophistication: Use the simplest tool that works
3. **Concision**: High signal-to-noise ratio
4. **Maintainability**: Easy for future programmers to modify
5. **Consistency**: Follow established patterns

## Domain-Driven Design

### Layer Architecture
- **API** → **Service** → **Repository** → **Domain**
- **Domain** (center) ← depends on nothing external
- Use dependency injection to wire layers

### Domain Root (domain.go)
Define domain entities, value objects, aggregates, service interfaces, request/response types, and domain constants. No external dependencies.

```go
package trade

type Service interface {
    Quote(ctx context.Context, params *QuoteParams) (*Quote, error)
}

type QuoteParams struct {
    PublicKey   string
    InputMint   string
    OutputMint  string
    Amount      uint64
}
```

### Domain Errors (errors.go)
```go
package trade

import "errors"

var ErrNotFound = errors.New("not found")

type SimulateTransactionError struct {
    Message string
}

func (e *SimulateTransactionError) Error() string {
    return e.Message
}
```

### Service Layer (service/)
Implement business logic and use cases. Depend on repository interfaces, orchestrate operations, handle business rules and validations, return domain errors.

```go
package service

type Service struct {
    repo trade.Repository
}

func New(repo trade.Repository) *Service {
    return &Service{repo: repo}
}
```

### File Organization Rules
1. **One concept per file**: Split large services into feature-specific files
2. **Interfaces at boundaries**: Define interfaces where layers meet
3. **Mocks in separate package**: Generated mocks go in `mocks/` directory
4. **Tests alongside implementation**: `service.go` → `service_test.go`

## Naming

**Casing:**
- Use `MixedCaps` or `mixedCaps` (camel case), never `snake_case`
- Exported: `MaxLength` not `MAX_LENGTH`
- Unexported: `maxLength` not `max_length`

**Packages:**
- Lowercase, no underscores, short, not plural
- Avoid: "common", "util", "shared", "lib"

**Functions - Avoid Repetition:**
```go
// Bad: Repeats package name
package yamlconfig
func ParseYAMLConfig(input string) (*Config, error)

// Good: Context makes it clear
package yamlconfig
func Parse(input string) (*Config, error)
```

**Function Naming Conventions:**
- Functions that **return** something: use noun-like names (`JobName()`)
- Functions that **do** something: use verb-like names (`WriteDetail()`)
- Avoid `Get` prefix: `JobName()` not `GetJobName()`

**Variables:**
- Unexported globals: prefix with `_` (e.g., `_defaultPort`)
- Exception: unexported errors use `err` prefix without underscore
- Local variables can be short when context is clear

## Variables & Declarations

**Top-level:**
```go
var s = F()  // Good
var s string = F()  // Bad if F() returns string
```

**Local:**
```go
s := "foo"  // Good
var s = "foo"  // Bad
```

**Zero values:**
```go
var filtered []int  // Good
filtered := []int{}  // Bad
```

## Constants

**Avoid Magic Numbers and Strings:**
```go
const (
    StatusCodeOK         = 200
    StatusCodeBadRequest = 400
    TimeoutHTTP          = 30 * time.Second
    TimeoutDatabase      = 10 * time.Second
)
```

## Error Handling

**Error Types:**
```go
// Static errors
var ErrNotFound = errors.New("not found")

// Dynamic errors
type NotFoundError struct { File string }

// Simple errors
errors.New("connection failed")
fmt.Errorf("invalid config: %v", err)
```

**Error Wrapping:**
```go
// Good: Concise
return fmt.Errorf("new store: %w", err)

// Bad: Verbose "failed to"
return fmt.Errorf("failed to create new store: %w", err)
```

**Error Naming:**
- Exported: `Err` prefix (`ErrBrokenLink`)
- Unexported: `err` prefix (`errNotFound`)
- Custom types: `Error` suffix (`NotFoundError`)

**Error Handling Rules:**
- Use `%w` to wrap errors when callers need underlying error
- Use `%v` to obfuscate underlying error
- Handle errors once: wrap and return OR log and degrade
- Type assertions: always use "comma ok" idiom
- **Use simple error variable names**: Always use `err` instead of descriptive names like `taskErr`, `enqueueErr`

```go
// Good
task, err := NewFetchTokenTask(chain, address, frames)
if err != nil {
    return fmt.Errorf("create task: %w", err)
}

_, err = client.EnqueueContext(ctx, task, asynq.Unique(TTL))
if err != nil {
    return fmt.Errorf("enqueue task: %w", err)
}

// Bad
task, taskErr := NewFetchTokenTask(chain, address, frames)
```

## Context Handling

**Context-Aware Methods:**
- Always prefer methods that accept context when available
- Use context-aware versions of methods (e.g., `EnqueueContext` instead of `Enqueue`)
- Pass context through the entire call chain

```go
// Good: Use context-aware methods
_, err = client.EnqueueContext(ctx, task, asynq.Unique(TTL))

// Bad: Non-context method when context version exists
_, err = client.Enqueue(task, asynq.Unique(TTL))
```

## Interfaces & Types

- Pass interfaces as values, not pointers
- Verify compliance: `var _ http.Handler = (*Handler)(nil)` (place after type definition)
- Value receivers: callable on pointers and values
- Pointer receivers: only on pointers or addressable values
- `sync.Mutex`: use zero value, never embed mutexes

### Interface Design for External Dependencies

```go
//go:generate mockgen -source=birdeye.go -destination=mocks/client.go -package=mocks
type Client interface {
    GetToken(ctx context.Context, address string) (*Token, error)
}

type Cli struct {
    host  string
    httpC *http.Client
}

var _ Client = (*Cli)(nil)
```

## Panics & Exits

- Only panic for truly irrecoverable situations
- `os.Exit` or `log.Fatal*` **only in `main()`**
- All other functions should return errors
- Use `run()` pattern for single exit point

## Concurrency

- Never fire-and-forget goroutines
- Every goroutine must have either:
  - A predictable stop time, OR
  - A way to signal it to stop (e.g., context, channel)
- Use `sync.WaitGroup` for multiple goroutines
- Use `chan struct{}` for single goroutine completion signaling
- Channel size should be 1 or 0 (unbuffered)

## Memory & Performance

**Copy at Boundaries:**
```go
func (d *Driver) SetTrips(trips []Trip) {
    d.trips = make([]Trip, len(trips))
    copy(d.trips, trips)
}
```

**Container Capacity:**
```go
make(map[T1]T2, hint)
make([]T, length, capacity)
```

**Performance Tips:**
- Prefer `strconv` over `fmt` for primitive conversions
- Avoid repeated string-to-byte conversions in loops

## Logging

**Structured Logging with slog:**
```go
// Good: Context-aware logging
logger.InfoContext(ctx, "processing task",
    slog.String("task_id", taskID),
    slog.String("user_id", userID))

// Bad: Non-context logging
logger.Info("processing task", slog.String("task_id", taskID))
```

## Code Structure

- Use `defer` for cleanup (locks, files, etc.)
- Reduce nesting: handle errors early, return/continue early
- Avoid unnecessary else
- Order functions by call order
- Group functions by receiver
- Struct definitions first, then `newXYZ()`, then methods, then utility functions

**Cognitive Complexity Management:**
- Break down complex functions into smaller, focused helper functions
- Extract repetitive logic into reusable methods
- Keep main functions focused on orchestration, not implementation details

## Maps & Slices

**Maps:**
```go
make(map[T1]T2)  // Good: Empty maps
map[T1]T2{}      // Bad
```

**Slices:**
- `nil` is a valid slice of length 0
- Return `nil` instead of `[]T{}`
- Check emptiness: `len(s) == 0`, not `s == nil`
- Zero value slice is usable without `make()`

## Time Handling

- Always use `time.Time` for instants of time
- Always use `time.Duration` for periods of time
- Include unit in field name when using `int`/`float64`: `IntervalMillis`

## Enums

```go
const (
    Add Operation = iota + 1  // Start at 1
    Subtract
    Multiply
)
```

## Built-in Names & Init

- Never shadow built-in names (error, string, int, etc.)
- Do not use `init()`
