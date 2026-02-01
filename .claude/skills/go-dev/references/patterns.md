# Go Design Patterns Reference

Common patterns for building maintainable Go applications.

## Dependency Injection

### Constructor Injection (Preferred)

```go
type UserService struct {
    repo   UserRepository
    logger Logger
    cache  Cache
}

func NewUserService(repo UserRepository, logger Logger, cache Cache) *UserService {
    return &UserService{
        repo:   repo,
        logger: logger,
        cache:  cache,
    }
}
```

### With Functional Options

```go
type UserService struct {
    repo    UserRepository
    logger  Logger
    cache   Cache
    timeout time.Duration
}

type ServiceOption func(*UserService)

func WithCache(c Cache) ServiceOption {
    return func(s *UserService) {
        s.cache = c
    }
}

func WithTimeout(d time.Duration) ServiceOption {
    return func(s *UserService) {
        s.timeout = d
    }
}

func NewUserService(repo UserRepository, logger Logger, opts ...ServiceOption) *UserService {
    s := &UserService{
        repo:    repo,
        logger:  logger,
        timeout: 30 * time.Second, // default
    }
    for _, opt := range opts {
        opt(s)
    }
    return s
}

// Usage
svc := NewUserService(repo, logger,
    WithCache(redisCache),
    WithTimeout(60*time.Second),
)
```

## Repository Pattern

### Interface Definition

```go
// Define in the package that uses it
type UserRepository interface {
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    List(ctx context.Context, filter UserFilter) ([]*User, error)
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}
```

### Implementation

```go
type postgresUserRepo struct {
    db *sql.DB
}

func NewPostgresUserRepo(db *sql.DB) *postgresUserRepo {
    return &postgresUserRepo{db: db}
}

func (r *postgresUserRepo) GetByID(ctx context.Context, id string) (*User, error) {
    query := `SELECT id, email, name, created_at FROM users WHERE id = $1`
    row := r.db.QueryRowContext(ctx, query, id)

    var u User
    err := row.Scan(&u.ID, &u.Email, &u.Name, &u.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("scanning user: %w", err)
    }
    return &u, nil
}

func (r *postgresUserRepo) Create(ctx context.Context, user *User) error {
    query := `INSERT INTO users (id, email, name, created_at) VALUES ($1, $2, $3, $4)`
    _, err := r.db.ExecContext(ctx, query, user.ID, user.Email, user.Name, user.CreatedAt)
    if err != nil {
        return fmt.Errorf("inserting user: %w", err)
    }
    return nil
}
```

## Service Layer Pattern

```go
type UserService struct {
    repo   UserRepository
    hasher PasswordHasher
    events EventPublisher
}

func NewUserService(repo UserRepository, hasher PasswordHasher, events EventPublisher) *UserService {
    return &UserService{repo: repo, hasher: hasher, events: events}
}

func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*User, error) {
    // Validation
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    // Check for existing user
    existing, err := s.repo.GetByEmail(ctx, req.Email)
    if err != nil && !errors.Is(err, ErrNotFound) {
        return nil, fmt.Errorf("checking existing user: %w", err)
    }
    if existing != nil {
        return nil, ErrEmailAlreadyExists
    }

    // Hash password
    hash, err := s.hasher.Hash(req.Password)
    if err != nil {
        return nil, fmt.Errorf("hashing password: %w", err)
    }

    // Create user
    user := &User{
        ID:           uuid.NewString(),
        Email:        req.Email,
        Name:         req.Name,
        PasswordHash: hash,
        CreatedAt:    time.Now(),
    }

    if err := s.repo.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("creating user: %w", err)
    }

    // Publish event
    s.events.Publish(ctx, UserRegisteredEvent{UserID: user.ID})

    return user, nil
}
```

## Handler Pattern (HTTP)

```go
type UserHandler struct {
    svc UserService
}

func NewUserHandler(svc UserService) *UserHandler {
    return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
    var req RegisterRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    user, err := h.svc.Register(r.Context(), req)
    if err != nil {
        switch {
        case errors.Is(err, ErrEmailAlreadyExists):
            h.writeError(w, http.StatusConflict, "email already registered")
        case errors.Is(err, ErrValidation):
            h.writeError(w, http.StatusBadRequest, err.Error())
        default:
            h.writeError(w, http.StatusInternalServerError, "internal error")
        }
        return
    }

    h.writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}

func (h *UserHandler) writeError(w http.ResponseWriter, status int, msg string) {
    h.writeJSON(w, status, map[string]string{"error": msg})
}
```

## Concurrency Patterns

### Worker Pool

```go
func ProcessItems(ctx context.Context, items []Item, workers int) []Result {
    jobs := make(chan Item, len(items))
    results := make(chan Result, len(items))

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for item := range jobs {
                select {
                case <-ctx.Done():
                    return
                case results <- process(item):
                }
            }
        }()
    }

    // Send jobs
    for _, item := range items {
        jobs <- item
    }
    close(jobs)

    // Wait and collect
    go func() {
        wg.Wait()
        close(results)
    }()

    var output []Result
    for r := range results {
        output = append(output, r)
    }
    return output
}
```

### Fan-Out, Fan-In

```go
func FanOut(ctx context.Context, input <-chan int, workers int) []<-chan int {
    outputs := make([]<-chan int, workers)
    for i := 0; i < workers; i++ {
        outputs[i] = worker(ctx, input)
    }
    return outputs
}

func FanIn(ctx context.Context, channels ...<-chan int) <-chan int {
    out := make(chan int)
    var wg sync.WaitGroup

    for _, ch := range channels {
        wg.Add(1)
        go func(c <-chan int) {
            defer wg.Done()
            for v := range c {
                select {
                case <-ctx.Done():
                    return
                case out <- v:
                }
            }
        }(ch)
    }

    go func() {
        wg.Wait()
        close(out)
    }()

    return out
}
```

### Pipeline

```go
func Pipeline(ctx context.Context, input []int) <-chan int {
    // Stage 1: Generate
    gen := func(nums []int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for _, n := range nums {
                select {
                case <-ctx.Done():
                    return
                case out <- n:
                }
            }
        }()
        return out
    }

    // Stage 2: Square
    square := func(in <-chan int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for n := range in {
                select {
                case <-ctx.Done():
                    return
                case out <- n * n:
                }
            }
        }()
        return out
    }

    // Stage 3: Filter even
    filterEven := func(in <-chan int) <-chan int {
        out := make(chan int)
        go func() {
            defer close(out)
            for n := range in {
                if n%2 == 0 {
                    select {
                    case <-ctx.Done():
                        return
                    case out <- n:
                    }
                }
            }
        }()
        return out
    }

    return filterEven(square(gen(input)))
}
```

### Semaphore (Limit Concurrency)

```go
type Semaphore struct {
    ch chan struct{}
}

func NewSemaphore(max int) *Semaphore {
    return &Semaphore{ch: make(chan struct{}, max)}
}

func (s *Semaphore) Acquire(ctx context.Context) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case s.ch <- struct{}{}:
        return nil
    }
}

func (s *Semaphore) Release() {
    <-s.ch
}

// Usage
func ProcessWithLimit(ctx context.Context, items []Item) {
    sem := NewSemaphore(10) // max 10 concurrent

    var wg sync.WaitGroup
    for _, item := range items {
        wg.Add(1)
        go func(it Item) {
            defer wg.Done()
            if err := sem.Acquire(ctx); err != nil {
                return
            }
            defer sem.Release()
            process(it)
        }(item)
    }
    wg.Wait()
}
```

### Once (Lazy Initialization)

```go
type Config struct {
    once   sync.Once
    config *AppConfig
    err    error
}

func (c *Config) Get() (*AppConfig, error) {
    c.once.Do(func() {
        c.config, c.err = loadConfig()
    })
    return c.config, c.err
}
```

## Error Handling Patterns

### Custom Error Types

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

type NotFoundError struct {
    Resource string
    ID       string
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s with id %s not found", e.Resource, e.ID)
}

// Usage
func GetUser(id string) (*User, error) {
    user, ok := users[id]
    if !ok {
        return nil, &NotFoundError{Resource: "user", ID: id}
    }
    return user, nil
}

// Checking
var notFound *NotFoundError
if errors.As(err, &notFound) {
    log.Printf("not found: %s %s", notFound.Resource, notFound.ID)
}
```

### Error Aggregation

```go
type MultiError struct {
    Errors []error
}

func (m *MultiError) Error() string {
    if len(m.Errors) == 1 {
        return m.Errors[0].Error()
    }
    return fmt.Sprintf("%d errors occurred", len(m.Errors))
}

func (m *MultiError) Add(err error) {
    if err != nil {
        m.Errors = append(m.Errors, err)
    }
}

func (m *MultiError) ErrorOrNil() error {
    if len(m.Errors) == 0 {
        return nil
    }
    return m
}

// Usage
func ValidateUser(u *User) error {
    var errs MultiError
    if u.Name == "" {
        errs.Add(&ValidationError{Field: "name", Message: "required"})
    }
    if u.Email == "" {
        errs.Add(&ValidationError{Field: "email", Message: "required"})
    }
    return errs.ErrorOrNil()
}
```

## Builder Pattern

```go
type QueryBuilder struct {
    table   string
    columns []string
    where   []string
    args    []interface{}
    orderBy string
    limit   int
}

func NewQuery(table string) *QueryBuilder {
    return &QueryBuilder{table: table}
}

func (q *QueryBuilder) Select(cols ...string) *QueryBuilder {
    q.columns = cols
    return q
}

func (q *QueryBuilder) Where(condition string, arg interface{}) *QueryBuilder {
    q.where = append(q.where, condition)
    q.args = append(q.args, arg)
    return q
}

func (q *QueryBuilder) OrderBy(col string) *QueryBuilder {
    q.orderBy = col
    return q
}

func (q *QueryBuilder) Limit(n int) *QueryBuilder {
    q.limit = n
    return q
}

func (q *QueryBuilder) Build() (string, []interface{}) {
    cols := "*"
    if len(q.columns) > 0 {
        cols = strings.Join(q.columns, ", ")
    }

    query := fmt.Sprintf("SELECT %s FROM %s", cols, q.table)

    if len(q.where) > 0 {
        query += " WHERE " + strings.Join(q.where, " AND ")
    }
    if q.orderBy != "" {
        query += " ORDER BY " + q.orderBy
    }
    if q.limit > 0 {
        query += fmt.Sprintf(" LIMIT %d", q.limit)
    }

    return query, q.args
}

// Usage
query, args := NewQuery("users").
    Select("id", "name", "email").
    Where("active = $1", true).
    Where("created_at > $2", time.Now().AddDate(0, -1, 0)).
    OrderBy("created_at DESC").
    Limit(10).
    Build()
```

## Middleware Pattern (HTTP)

```go
type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
    for i := len(middlewares) - 1; i >= 0; i-- {
        h = middlewares[i](h)
    }
    return h
}

func LoggingMiddleware(logger Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            next.ServeHTTP(w, r)
            logger.Info("request",
                "method", r.Method,
                "path", r.URL.Path,
                "duration", time.Since(start),
            )
        })
    }
}

func AuthMiddleware(auth Authenticator) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := r.Header.Get("Authorization")
            user, err := auth.Validate(token)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), userKey, user)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Usage
handler := Chain(myHandler,
    LoggingMiddleware(logger),
    AuthMiddleware(auth),
    RecoveryMiddleware(),
)
```

## Result Type Pattern

```go
type Result[T any] struct {
    Value T
    Err   error
}

func Ok[T any](v T) Result[T] {
    return Result[T]{Value: v}
}

func Err[T any](err error) Result[T] {
    return Result[T]{Err: err}
}

func (r Result[T]) IsOk() bool {
    return r.Err == nil
}

func (r Result[T]) Unwrap() T {
    if r.Err != nil {
        panic(r.Err)
    }
    return r.Value
}

func (r Result[T]) UnwrapOr(def T) T {
    if r.Err != nil {
        return def
    }
    return r.Value
}

// Usage
func ParseConfig(path string) Result[*Config] {
    data, err := os.ReadFile(path)
    if err != nil {
        return Err[*Config](err)
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return Err[*Config](err)
    }
    return Ok(&cfg)
}
```

## Graceful Shutdown

```go
func main() {
    srv := &http.Server{Addr: ":8080", Handler: router}

    // Start server
    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("shutting down...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("shutdown error: %v", err)
    }

    log.Println("server stopped")
}
```
