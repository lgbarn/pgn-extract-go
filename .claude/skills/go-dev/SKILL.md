---
name: go-dev
description: Comprehensive Go development skill for writing idiomatic, well-tested Go code. Use when creating new Go projects, writing or refactoring Go code, implementing Go tests, or reviewing Go code for best practices. Enforces TDD workflow, table-driven tests, and idiomatic patterns including proper error handling, interface design, and package organization.
---

# Go Development

Write idiomatic, well-tested Go code using Test-Driven Development.

## Workflow Decision Tree

1. **Creating new project?** → Initialize module, set up project structure, create first test
2. **Adding new feature?** → Write failing test first → Implement → Refactor
3. **Refactoring existing code?** → Ensure test coverage exists → Refactor → Verify tests pass
4. **Fixing a bug?** → Write test that reproduces bug → Fix → Verify test passes

## TDD Workflow (Red → Green → Refactor)

Always follow this cycle:

```
1. RED:    Write a failing test that defines desired behavior
2. GREEN:  Write minimal code to make the test pass
3. REFACTOR: Clean up while keeping tests green
```

### TDD in Practice

```go
// 1. RED - Write the test first
func TestAdd(t *testing.T) {
    got := Add(2, 3)
    want := 5
    if got != want {
        t.Errorf("Add(2, 3) = %d; want %d", got, want)
    }
}

// 2. GREEN - Minimal implementation
func Add(a, b int) int {
    return a + b
}

// 3. REFACTOR - Improve if needed (this case is already clean)
```

## Project Structure

Standard layout for non-trivial projects:

```
myproject/
├── cmd/
│   └── myapp/
│       └── main.go           # Application entry point
├── internal/
│   ├── domain/               # Core business types
│   ├── service/              # Business logic
│   ├── repository/           # Data access
│   └── handler/              # HTTP/gRPC handlers
├── pkg/                      # Public library code (optional)
├── go.mod
├── go.sum
└── README.md
```

For simple projects or libraries:
```
mylib/
├── mylib.go
├── mylib_test.go
├── go.mod
└── README.md
```

## Initialization Commands

```bash
# New module
go mod init github.com/user/project

# Add dependencies
go get github.com/some/dependency

# Tidy dependencies
go mod tidy

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests verbose
go test -v ./...

# Run specific test
go test -run TestFunctionName ./...
```

## Core Testing Patterns

### Table-Driven Tests (Default Pattern)

Use for any function with multiple input/output cases:

```go
func TestCalculateDiscount(t *testing.T) {
    tests := []struct {
        name     string
        price    float64
        quantity int
        want     float64
    }{
        {"no discount under 10", 100.0, 5, 500.0},
        {"10% discount at 10+", 100.0, 10, 900.0},
        {"20% discount at 50+", 100.0, 50, 4000.0},
        {"zero quantity", 100.0, 0, 0.0},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateDiscount(tt.price, tt.quantity)
            if got != tt.want {
                t.Errorf("CalculateDiscount(%v, %v) = %v; want %v",
                    tt.price, tt.quantity, got, tt.want)
            }
        })
    }
}
```

### Interface-Based Mocking

Define small interfaces at point of use, not at implementation:

```go
// In service package - define what you need
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}

type UserService struct {
    repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
    return &UserService{repo: repo}
}

// In test file - create mock implementing the interface
type mockUserRepo struct {
    users map[string]*User
    err   error
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    if m.err != nil {
        return nil, m.err
    }
    return m.users[id], nil
}

func (m *mockUserRepo) Save(ctx context.Context, user *User) error {
    if m.err != nil {
        return m.err
    }
    m.users[user.ID] = user
    return nil
}

func TestUserService_GetUser(t *testing.T) {
    mock := &mockUserRepo{
        users: map[string]*User{"123": {ID: "123", Name: "Alice"}},
    }
    svc := NewUserService(mock)

    user, err := svc.GetUser(context.Background(), "123")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if user.Name != "Alice" {
        t.Errorf("got name %q; want Alice", user.Name)
    }
}
```

## Error Handling

### Standard Pattern

```go
// Return errors, don't panic
func DoSomething() error {
    if err := step1(); err != nil {
        return fmt.Errorf("step1 failed: %w", err)
    }
    return nil
}

// Wrap with context using %w
func ProcessFile(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return fmt.Errorf("reading %s: %w", path, err)
    }
    // ...
    return nil
}

// Check wrapped errors
if errors.Is(err, os.ErrNotExist) {
    // handle missing file
}

var pathErr *os.PathError
if errors.As(err, &pathErr) {
    // use pathErr.Path, pathErr.Op, etc.
}
```

### Sentinel Errors

```go
// Define at package level
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
)

// Use in functions
func GetUser(id string) (*User, error) {
    user, ok := users[id]
    if !ok {
        return nil, ErrNotFound
    }
    return user, nil
}

// Check in callers
if errors.Is(err, ErrNotFound) {
    // handle not found case
}
```

## Functional Options Pattern

For constructors with optional configuration:

```go
type Server struct {
    addr    string
    timeout time.Duration
    logger  Logger
}

type Option func(*Server)

func WithTimeout(d time.Duration) Option {
    return func(s *Server) {
        s.timeout = d
    }
}

func WithLogger(l Logger) Option {
    return func(s *Server) {
        s.logger = l
    }
}

func NewServer(addr string, opts ...Option) *Server {
    s := &Server{
        addr:    addr,
        timeout: 30 * time.Second, // default
        logger:  defaultLogger,    // default
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage
srv := NewServer(":8080",
    WithTimeout(60*time.Second),
    WithLogger(myLogger),
)
```

## Context Usage

Always pass context as first parameter:

```go
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Pass context to downstream calls
    order, err := s.repo.GetOrder(ctx, orderID)
    if err != nil {
        return err
    }

    // Use context with HTTP requests
    req, _ := http.NewRequestWithContext(ctx, "POST", url, body)

    return nil
}
```

## Quick Reference: Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Package | lowercase, short, no underscores | `http`, `json`, `userservice` |
| Exported | PascalCase | `UserService`, `GetByID` |
| Unexported | camelCase | `userCache`, `parseInput` |
| Interface | er suffix when single method | `Reader`, `Writer`, `Stringer` |
| Acronyms | all caps | `HTTPServer`, `XMLParser`, `userID` |
| Test files | `_test.go` suffix | `user_test.go` |
| Test funcs | `Test` prefix | `TestUserService_GetByID` |

## References

Detailed documentation available:

- **Testing patterns**: See [references/testing.md](references/testing.md) for comprehensive testing including benchmarks, fuzz testing, test fixtures, and advanced mocking
- **Design patterns**: See [references/patterns.md](references/patterns.md) for repository pattern, dependency injection, concurrency patterns, and more
- **Style guide**: See [references/style-guide.md](references/style-guide.md) for complete naming conventions, documentation standards, and code organization
