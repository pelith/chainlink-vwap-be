---
name: go
description: Go development guidelines for the vwap codebase. Use when working with Go files (.go), writing tests (*_test.go), changing go.mod/module path, running golangci-lint, editing Makefile targets, working with sqlc/migrations/seeds (database/*), implementing API handlers (internal/*/api/*.go), or creating new domain packages under internal/. Provides project structure, code patterns, testing conventions, and required workflows.
---

# Go Development Guide

## How to Use This Skill (must follow)

When doing any Go work in this repo, treat `.cursor/references/` as the source of truth and **read the relevant reference before coding**:

- API handler / routing / response shape → `@.cursor/references/api-guide.md`
- API handler tests / gomock / request helpers → `@.cursor/references/api-testing-guide.md`
- Go style / naming / error handling / logging → `@.cursor/references/style-guide.md`
- Testing conventions / goleak / patterns → `@.cursor/references/testing-guide.md`

## Project Structure

```
vwap/
├── cmd/{api,migration}/          # Application entry points
├── database/
│   ├── migrations/               # SQL migrations (YYYYMMDDHHMMSS_name.up/down.sql)
│   ├── queries/                  # SQL queries for sqlc
│   └── seeds/                    # SQL seed migrations (by env)
└── internal/
    ├── api/                      # HTTP server, routing, middleware
    ├── config/                   # Config loading + typed app configs
    ├── db/                       # sqlc-generated code
    ├── httpwrap/                 # HTTP response utilities
    └── {domain}/                 # Domain packages (examples: user/, liquidity/)
```

## Required Workflow After Code Changes

1. `make gci-format` - Format imports
2. `make lint` - Fix all linting issues
3. `make test` - Ensure tests pass
4. If SQL queries changed: `make sqlc`

## Core Patterns

### Error Handling
```go
// Define domain errors
var ErrNotFound = errors.New("not found")

// Wrap errors with context
return fmt.Errorf("get user: %w", err)

// Check with errors.Is
if errors.Is(err, domain.ErrNotFound) { ... }

// Use simple err variable names (not taskErr, repoErr)
```

### Testing Pattern
```go
func TestService_Method(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        setupRepo func(ctrl *gomock.Controller) *mocks.MockRepository
        want      *Result
        wantErr   bool
    }{
        {
            name: "success",
            setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
                repo := mocks.NewMockRepository(ctrl)
                repo.EXPECT().GetByID(gomock.Any(), id).Return(&Entity{}, nil)
                return repo
            },
            want: &Result{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctrl := gomock.NewController(t)
            svc := New(tt.setupRepo(ctrl))

            got, err := svc.Method(t.Context(), params)
            if err != nil {
                if !tt.wantErr {
                    t.Errorf("Method() failed: %v", err)
                }
                return
            }

            if !cmp.Equal(got, tt.want) {
                t.Errorf("Method() = %v, want %v, diff %v", got, tt.want, cmp.Diff(got, tt.want))
            }
        })
    }
}
```

## Key Conventions

- **Naming**: MixedCaps (never snake_case), avoid Get prefix (`Name()` not `GetName()`)
- **Context**: Always pass as first parameter, use context-aware methods
- **Logging**: Use `slog` with `InfoContext`, `ErrorContext` (not `Info`, `Error`)
- **Time**: Always use `time.Now().UTC()`
- **Slices**: Preallocate with capacity when size is known
- **Mocks**: Use `gomock.Any()` only for context, check other parameters

## Detailed References

For comprehensive guidelines, see:
- [Go Style Guide](../../references/style-guide.md) - Complete coding standards
- [Testing Guide](../../references/testing-guide.md) - Testing patterns and gomock usage
- [API Guide](../../references/api-guide.md) - HTTP handler implementation
- [API Testing Guide](../../references/api-testing-guide.md) - API test patterns
