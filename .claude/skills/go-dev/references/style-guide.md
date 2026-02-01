# Go Style Guide

Idiomatic Go conventions for readable, maintainable code.

## Naming Conventions

### Packages

```go
// Good - lowercase, short, singular
package user
package http
package json

// Bad
package userService    // no camelCase
package user_service   // no underscores
package users          // avoid plural (usually)
package httpHandler    // too long
```

### Variables and Functions

```go
// Exported (public) - PascalCase
func GetUserByID(id string) (*User, error)
type UserService struct {}
var MaxRetries = 3

// Unexported (private) - camelCase
func validateEmail(email string) error
type userCache struct {}
var defaultTimeout = 30 * time.Second
```

### Acronyms

```go
// Good - acronyms are all caps
var userID string
func ServeHTTP(w http.ResponseWriter, r *http.Request)
type XMLParser struct {}
var apiURL string

// Bad
var UserId string    // should be UserID
func ServeHttp()     // should be ServeHTTP
type XmlParser struct {} // should be XMLParser
```

### Interface Names

```go
// Single-method interfaces: method name + "er"
type Reader interface {
    Read(p []byte) (n int, err error)
}

type Stringer interface {
    String() string
}

type Closer interface {
    Close() error
}

// Multi-method interfaces: descriptive noun
type ReadWriter interface {
    Reader
    Writer
}

type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    Save(ctx context.Context, user *User) error
}
```

### Receiver Names

```go
// Good - short, consistent, not "this" or "self"
func (u *User) FullName() string {
    return u.FirstName + " " + u.LastName
}

func (us *UserService) GetByID(id string) (*User, error) {
    // ...
}

// Bad
func (this *User) FullName() string  // don't use "this"
func (self *User) FullName() string  // don't use "self"
func (user *User) FullName() string  // too long
```

## Code Organization

### File Structure

```go
// file: user.go

package user

import (
    "context"
    "errors"
    "fmt"

    "github.com/google/uuid"

    "myapp/internal/domain"
)

// Constants
const (
    MaxNameLength  = 100
    MinAgeRequired = 13
)

// Errors
var (
    ErrNotFound     = errors.New("user not found")
    ErrInvalidEmail = errors.New("invalid email")
)

// Types
type User struct {
    ID        string
    Email     string
    Name      string
    CreatedAt time.Time
}

// Constructors
func New(email, name string) *User {
    return &User{
        ID:        uuid.NewString(),
        Email:     email,
        Name:      name,
        CreatedAt: time.Now(),
    }
}

// Methods
func (u *User) Validate() error {
    if u.Email == "" {
        return ErrInvalidEmail
    }
    return nil
}

// Functions
func ValidateEmail(email string) bool {
    // ...
}
```

### Import Grouping

```go
import (
    // Standard library
    "context"
    "errors"
    "fmt"
    "time"

    // Third-party packages
    "github.com/google/uuid"
    "github.com/lib/pq"

    // Internal packages
    "myapp/internal/config"
    "myapp/internal/domain"
)
```

### Package Layout

```
internal/
├── domain/           # Core business types (no dependencies)
│   ├── user.go
│   ├── order.go
│   └── product.go
├── service/          # Business logic (depends on domain)
│   ├── user.go
│   └── order.go
├── repository/       # Data access (depends on domain)
│   ├── user.go
│   └── postgres/
│       └── user.go
└── handler/          # HTTP/gRPC handlers (depends on service)
    ├── user.go
    └── middleware.go
```

## Comments and Documentation

### Package Comments

```go
// Package user provides user management functionality including
// registration, authentication, and profile management.
package user
```

### Function Comments

```go
// GetByID retrieves a user by their unique identifier.
// It returns ErrNotFound if no user exists with the given ID.
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
    // ...
}

// Register creates a new user account with the given email and password.
// It validates the email format and password strength before creating the account.
// Returns the created user or an error if registration fails.
//
// Example:
//
//     user, err := svc.Register(ctx, "user@example.com", "securepass123")
//     if err != nil {
//         log.Fatal(err)
//     }
func (s *Service) Register(ctx context.Context, email, password string) (*User, error) {
    // ...
}
```

### Type Comments

```go
// User represents a registered user in the system.
// Users can have multiple roles and belong to one organization.
type User struct {
    // ID is the unique identifier for the user.
    ID string

    // Email is the user's email address, used for authentication.
    Email string

    // Name is the user's display name.
    Name string

    // CreatedAt is when the user account was created.
    CreatedAt time.Time
}
```

## Error Handling Style

### Return Early

```go
// Good - return early on errors
func ProcessFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("opening file: %w", err)
    }
    defer f.Close()

    data, err := io.ReadAll(f)
    if err != nil {
        return fmt.Errorf("reading file: %w", err)
    }

    return process(data)
}

// Bad - nested if-else
func ProcessFile(path string) error {
    f, err := os.Open(path)
    if err == nil {
        defer f.Close()
        data, err := io.ReadAll(f)
        if err == nil {
            return process(data)
        } else {
            return fmt.Errorf("reading file: %w", err)
        }
    } else {
        return fmt.Errorf("opening file: %w", err)
    }
}
```

### Error Wrapping

```go
// Good - add context with %w
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    user, err := s.users.GetByID(ctx, req.UserID)
    if err != nil {
        return nil, fmt.Errorf("getting user %s: %w", req.UserID, err)
    }

    order, err := s.orders.Create(ctx, user, req.Items)
    if err != nil {
        return nil, fmt.Errorf("creating order: %w", err)
    }

    return order, nil
}

// Bad - losing error context
func (s *Service) CreateOrder(ctx context.Context, req CreateOrderRequest) (*Order, error) {
    user, err := s.users.GetByID(ctx, req.UserID)
    if err != nil {
        return nil, err  // no context added
    }
    // ...
}
```

### Handling Errors

```go
// Check specific errors with errors.Is
if errors.Is(err, sql.ErrNoRows) {
    return nil, ErrNotFound
}

// Extract error types with errors.As
var pathErr *os.PathError
if errors.As(err, &pathErr) {
    log.Printf("operation %s failed on path %s", pathErr.Op, pathErr.Path)
}

// Don't use string comparison
// Bad
if err.Error() == "not found" {
    // fragile!
}
```

## Formatting

### Line Length

```go
// Good - break long lines sensibly
func (s *UserService) CreateUserWithOptions(
    ctx context.Context,
    email string,
    name string,
    opts ...CreateOption,
) (*User, error) {
    // ...
}

// Good - break long calls
result, err := s.repository.FindUsersWithComplexCriteria(
    ctx,
    filter,
    pagination,
    sorting,
)
```

### Struct Literals

```go
// Good - one field per line for multiple fields
user := &User{
    ID:        uuid.NewString(),
    Email:     "user@example.com",
    Name:      "Alice",
    CreatedAt: time.Now(),
}

// Good - single line for simple structs
point := Point{X: 10, Y: 20}

// Good - use field names always (except simple types)
return &Response{
    Status:  200,
    Message: "success",
}

// Bad - positional args in multi-field struct
return &User{uuid.NewString(), "email", "name", time.Now()}
```

### Blank Lines

```go
func ProcessOrder(ctx context.Context, order *Order) error {
    // Validate order
    if err := order.Validate(); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }

    // Check inventory
    available, err := checkInventory(ctx, order.Items)
    if err != nil {
        return fmt.Errorf("checking inventory: %w", err)
    }
    if !available {
        return ErrOutOfStock
    }

    // Process payment
    payment, err := processPayment(ctx, order)
    if err != nil {
        return fmt.Errorf("processing payment: %w", err)
    }

    // Complete order
    order.PaymentID = payment.ID
    order.Status = StatusCompleted
    return saveOrder(ctx, order)
}
```

## Control Flow

### Switch vs If-Else

```go
// Good - use switch for multiple conditions
switch status {
case StatusPending:
    return "Pending"
case StatusActive:
    return "Active"
case StatusCancelled:
    return "Cancelled"
default:
    return "Unknown"
}

// Good - type switch
switch v := value.(type) {
case string:
    return len(v)
case int:
    return v
case []byte:
    return len(v)
default:
    return 0
}
```

### Range Loops

```go
// Iterating with index and value
for i, v := range items {
    fmt.Printf("%d: %v\n", i, v)
}

// Iterating values only
for _, v := range items {
    process(v)
}

// Iterating indices only
for i := range items {
    items[i] = i * 2
}

// Iterating maps
for key, value := range m {
    fmt.Printf("%s: %v\n", key, value)
}
```

## Struct Design

### Use Pointers for Large Structs

```go
// Large struct - use pointer receiver and return pointers
type Report struct {
    Title    string
    Sections []Section
    Data     [][]float64
    Metadata map[string]string
}

func (r *Report) AddSection(s Section) {
    r.Sections = append(r.Sections, s)
}

func GenerateReport(data [][]float64) *Report {
    return &Report{Data: data}
}

// Small struct - value receiver is fine
type Point struct {
    X, Y float64
}

func (p Point) Distance(other Point) float64 {
    dx := p.X - other.X
    dy := p.Y - other.Y
    return math.Sqrt(dx*dx + dy*dy)
}
```

### Zero Values

```go
// Design types to have useful zero values
type Buffer struct {
    data []byte
}

func (b *Buffer) Write(p []byte) {
    b.data = append(b.data, p...)  // works with nil slice
}

// Usage - no constructor needed
var buf Buffer
buf.Write([]byte("hello"))

// Config with sensible defaults
type Config struct {
    Timeout time.Duration
    Retries int
}

func (c Config) TimeoutOrDefault() time.Duration {
    if c.Timeout == 0 {
        return 30 * time.Second
    }
    return c.Timeout
}
```

## Channel and Goroutine Style

### Channel Direction

```go
// Specify direction when possible
func producer(out chan<- int) {
    for i := 0; i < 10; i++ {
        out <- i
    }
    close(out)
}

func consumer(in <-chan int) {
    for v := range in {
        process(v)
    }
}
```

### Goroutine Lifecycle

```go
// Always ensure goroutines can exit
func worker(ctx context.Context, jobs <-chan Job) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-jobs:
            if !ok {
                return  // channel closed
            }
            process(job)
        }
    }
}

// Use WaitGroup for multiple goroutines
func processAll(items []Item) {
    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(it Item) {
            defer wg.Done()
            process(it)
        }(item)
    }
    wg.Wait()
}
```

## Common Antipatterns to Avoid

```go
// Bad - naked return with named results (confusing)
func divide(a, b int) (result int, err error) {
    if b == 0 {
        err = errors.New("division by zero")
        return  // confusing - what's returned?
    }
    result = a / b
    return
}

// Good - explicit return
func divide(a, b int) (int, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

// Bad - init() for complex setup
func init() {
    db, err := sql.Open("postgres", os.Getenv("DB_URL"))
    if err != nil {
        panic(err)  // can't test, can't handle gracefully
    }
    globalDB = db
}

// Good - explicit initialization
func main() {
    db, err := sql.Open("postgres", os.Getenv("DB_URL"))
    if err != nil {
        log.Fatalf("opening database: %v", err)
    }
    defer db.Close()
    
    svc := NewService(db)
    // ...
}

// Bad - returning interface from constructor
func NewReader() io.Reader {
    return &myReader{}  // hides concrete type
}

// Good - return concrete type, accept interfaces
func NewReader() *MyReader {
    return &MyReader{}
}

func Process(r io.Reader) error {
    // accepts any Reader
}
```

## Testing Style

### Test Function Names

```go
// Format: TestTypeName_MethodName_Scenario
func TestUserService_GetByID_ReturnsUser(t *testing.T)
func TestUserService_GetByID_NotFound(t *testing.T)
func TestUserService_Create_ValidatesEmail(t *testing.T)

// For package-level functions
func TestValidateEmail_ValidEmail(t *testing.T)
func TestValidateEmail_EmptyString(t *testing.T)
```

### Test Organization

```go
func TestCalculate(t *testing.T) {
    // Arrange
    input := 42
    expected := 84

    // Act
    result := Calculate(input)

    // Assert
    if result != expected {
        t.Errorf("Calculate(%d) = %d; want %d", input, result, expected)
    }
}
```
