# Go Testing Reference

Comprehensive guide to testing in Go with TDD practices.

## TDD Workflow Details

### The Cycle

1. **RED**: Write a test for behavior that doesn't exist yet. Run it - it must fail.
2. **GREEN**: Write the simplest code that makes the test pass. No more.
3. **REFACTOR**: Clean up both test and implementation. Tests must stay green.

### TDD Discipline

- Never write production code without a failing test
- Write only enough test to fail (compilation failures count)
- Write only enough production code to pass the current test
- Refactor aggressively but only when green

### Starting a New Feature with TDD

```go
// Step 1: Write the test (file: calculator_test.go)
package calculator

import "testing"

func TestMultiply(t *testing.T) {
    got := Multiply(3, 4)
    want := 12
    if got != want {
        t.Errorf("Multiply(3, 4) = %d; want %d", got, want)
    }
}

// Step 2: Create minimal implementation (file: calculator.go)
package calculator

func Multiply(a, b int) int {
    return a * b
}

// Step 3: Add more test cases, repeat
```

## Table-Driven Tests

### Basic Structure

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    int
        wantErr bool
    }{
        {"valid positive", "42", 42, false},
        {"valid negative", "-17", -17, false},
        {"valid zero", "0", 0, false},
        {"empty string", "", 0, true},
        {"invalid chars", "abc", 0, true},
        {"overflow", "99999999999999999999", 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
            }
        })
    }
}
```

### With Setup/Teardown per Case

```go
func TestDatabase(t *testing.T) {
    tests := []struct {
        name    string
        setup   func(db *MockDB)
        input   string
        want    *User
        wantErr error
    }{
        {
            name: "user exists",
            setup: func(db *MockDB) {
                db.users["123"] = &User{ID: "123", Name: "Alice"}
            },
            input: "123",
            want:  &User{ID: "123", Name: "Alice"},
        },
        {
            name:    "user not found",
            setup:   func(db *MockDB) {},
            input:   "456",
            wantErr: ErrNotFound,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db := NewMockDB()
            tt.setup(db)
            svc := NewUserService(db)

            got, err := svc.GetUser(context.Background(), tt.input)

            if !errors.Is(err, tt.wantErr) {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %+v, want %+v", got, tt.want)
            }
        })
    }
}
```

## Subtests and Parallel Execution

### Subtests for Organization

```go
func TestUserService(t *testing.T) {
    t.Run("Create", func(t *testing.T) {
        t.Run("valid user", func(t *testing.T) {
            // test creating valid user
        })
        t.Run("duplicate email", func(t *testing.T) {
            // test duplicate email error
        })
    })

    t.Run("Delete", func(t *testing.T) {
        t.Run("existing user", func(t *testing.T) {
            // test deleting existing user
        })
        t.Run("non-existent user", func(t *testing.T) {
            // test deleting non-existent user
        })
    })
}
```

### Parallel Tests

```go
func TestParallel(t *testing.T) {
    tests := []struct {
        name  string
        input int
        want  int
    }{
        {"case1", 1, 2},
        {"case2", 2, 4},
        {"case3", 3, 6},
    }

    for _, tt := range tests {
        tt := tt // capture range variable for parallel execution
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // run this subtest in parallel
            got := Double(tt.input)
            if got != tt.want {
                t.Errorf("Double(%d) = %d; want %d", tt.input, got, tt.want)
            }
        })
    }
}
```

## Mocking Patterns

### Interface-Based Mocks

```go
// Define interface for what you need
type EmailSender interface {
    Send(to, subject, body string) error
}

// Production implementation
type SMTPSender struct {
    host string
}

func (s *SMTPSender) Send(to, subject, body string) error {
    // actual SMTP implementation
}

// Mock for testing
type mockEmailSender struct {
    calls []emailCall
    err   error
}

type emailCall struct {
    to, subject, body string
}

func (m *mockEmailSender) Send(to, subject, body string) error {
    m.calls = append(m.calls, emailCall{to, subject, body})
    return m.err
}

// Test using the mock
func TestNotifyUser(t *testing.T) {
    mock := &mockEmailSender{}
    svc := NewNotificationService(mock)

    err := svc.NotifyUser("user@example.com", "Welcome!")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(mock.calls) != 1 {
        t.Fatalf("expected 1 email, got %d", len(mock.calls))
    }
    if mock.calls[0].to != "user@example.com" {
        t.Errorf("wrong recipient: %s", mock.calls[0].to)
    }
}
```

### Spy Pattern (Record Calls)

```go
type spyLogger struct {
    logs []string
}

func (s *spyLogger) Log(msg string) {
    s.logs = append(s.logs, msg)
}

func (s *spyLogger) Contains(msg string) bool {
    for _, log := range s.logs {
        if strings.Contains(log, msg) {
            return true
        }
    }
    return false
}

func TestLogging(t *testing.T) {
    spy := &spyLogger{}
    svc := NewService(spy)

    svc.DoSomething()

    if !spy.Contains("operation completed") {
        t.Error("expected log message not found")
    }
}
```

### Stub Pattern (Canned Responses)

```go
type stubTimeProvider struct {
    now time.Time
}

func (s *stubTimeProvider) Now() time.Time {
    return s.now
}

func TestExpiration(t *testing.T) {
    fixedTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
    stub := &stubTimeProvider{now: fixedTime}
    cache := NewCache(stub)

    cache.Set("key", "value", 1*time.Hour)

    // Advance time by 2 hours
    stub.now = fixedTime.Add(2 * time.Hour)

    _, ok := cache.Get("key")
    if ok {
        t.Error("expected cache entry to be expired")
    }
}
```

## Test Fixtures and Helpers

### Test Helper Functions

```go
// t.Helper() makes error reports point to the caller
func assertEqual(t *testing.T, got, want interface{}) {
    t.Helper()
    if !reflect.DeepEqual(got, want) {
        t.Errorf("got %v; want %v", got, want)
    }
}

func assertError(t *testing.T, err, want error) {
    t.Helper()
    if !errors.Is(err, want) {
        t.Errorf("got error %v; want %v", err, want)
    }
}

func assertNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

### Test Fixtures with testdata

```go
// Files in testdata/ are ignored by go build but available to tests
func TestParseConfig(t *testing.T) {
    data, err := os.ReadFile("testdata/valid_config.json")
    if err != nil {
        t.Fatalf("reading fixture: %v", err)
    }

    cfg, err := ParseConfig(data)
    if err != nil {
        t.Fatalf("parsing config: %v", err)
    }

    if cfg.Name != "test" {
        t.Errorf("name = %q; want test", cfg.Name)
    }
}
```

### Golden Files Pattern

```go
var update = flag.Bool("update", false, "update golden files")

func TestFormatOutput(t *testing.T) {
    input := loadTestData(t, "input.txt")
    got := FormatOutput(input)

    goldenPath := "testdata/golden.txt"
    if *update {
        os.WriteFile(goldenPath, []byte(got), 0644)
    }

    want, _ := os.ReadFile(goldenPath)
    if got != string(want) {
        t.Errorf("output mismatch:\ngot:\n%s\nwant:\n%s", got, want)
    }
}

// Run with: go test -update to regenerate golden files
```

## Benchmarks

### Basic Benchmark

```go
func BenchmarkFibonacci(b *testing.B) {
    for i := 0; i < b.N; i++ {
        Fibonacci(20)
    }
}

// Run with: go test -bench=.
```

### Benchmark with Setup

```go
func BenchmarkSort(b *testing.B) {
    // Setup outside the loop
    data := make([]int, 10000)
    for i := range data {
        data[i] = rand.Intn(10000)
    }

    b.ResetTimer() // Don't count setup time

    for i := 0; i < b.N; i++ {
        // Copy data to avoid sorting already-sorted slice
        input := make([]int, len(data))
        copy(input, data)
        sort.Ints(input)
    }
}
```

### Sub-benchmarks

```go
func BenchmarkConcat(b *testing.B) {
    sizes := []int{10, 100, 1000, 10000}

    for _, size := range sizes {
        b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
            strs := make([]string, size)
            for i := range strs {
                strs[i] = "x"
            }

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                _ = strings.Join(strs, "")
            }
        })
    }
}
```

### Memory Allocation Benchmarks

```go
func BenchmarkAllocs(b *testing.B) {
    b.ReportAllocs() // Report memory allocations

    for i := 0; i < b.N; i++ {
        _ = make([]byte, 1024)
    }
}

// Run with: go test -bench=. -benchmem
```

## Fuzz Testing (Go 1.18+)

### Basic Fuzz Test

```go
func FuzzParseInt(f *testing.F) {
    // Add seed corpus
    f.Add("123")
    f.Add("-456")
    f.Add("0")

    f.Fuzz(func(t *testing.T, input string) {
        result, err := ParseInt(input)
        if err != nil {
            return // Invalid input is fine
        }

        // Property: formatted result should parse back to same value
        formatted := fmt.Sprintf("%d", result)
        reparsed, err := ParseInt(formatted)
        if err != nil {
            t.Errorf("failed to reparse %q: %v", formatted, err)
        }
        if reparsed != result {
            t.Errorf("round-trip failed: %d -> %q -> %d", result, formatted, reparsed)
        }
    })
}

// Run with: go test -fuzz=FuzzParseInt
```

### Fuzz with Multiple Inputs

```go
func FuzzJSON(f *testing.F) {
    f.Add([]byte(`{"name": "test"}`))
    f.Add([]byte(`[]`))
    f.Add([]byte(`null`))

    f.Fuzz(func(t *testing.T, data []byte) {
        var v interface{}
        if err := json.Unmarshal(data, &v); err != nil {
            return // Invalid JSON is fine
        }

        // Property: valid JSON should re-marshal successfully
        _, err := json.Marshal(v)
        if err != nil {
            t.Errorf("failed to remarshal: %v", err)
        }
    })
}
```

## Integration Testing

### Build Tags for Integration Tests

```go
//go:build integration

package mypackage

import "testing"

func TestDatabaseIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    db := connectToRealDatabase()
    defer db.Close()

    // ... integration tests
}

// Run with: go test -tags=integration
```

### TestMain for Setup/Teardown

```go
var testDB *sql.DB

func TestMain(m *testing.M) {
    // Setup
    var err error
    testDB, err = sql.Open("postgres", testDSN)
    if err != nil {
        log.Fatal(err)
    }

    // Run tests
    code := m.Run()

    // Teardown
    testDB.Close()

    os.Exit(code)
}

func TestQuery(t *testing.T) {
    // Use testDB here
    rows, err := testDB.Query("SELECT 1")
    // ...
}
```

## HTTP Testing

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    // Create request
    req := httptest.NewRequest("GET", "/users/123", nil)
    req.Header.Set("Authorization", "Bearer token")

    // Create response recorder
    rec := httptest.NewRecorder()

    // Call handler
    handler := NewUserHandler(mockRepo)
    handler.ServeHTTP(rec, req)

    // Check response
    if rec.Code != http.StatusOK {
        t.Errorf("status = %d; want %d", rec.Code, http.StatusOK)
    }

    var user User
    json.NewDecoder(rec.Body).Decode(&user)
    if user.ID != "123" {
        t.Errorf("user.ID = %q; want 123", user.ID)
    }
}
```

### Test Server

```go
func TestClient(t *testing.T) {
    // Create test server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/api/data" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
    }))
    defer srv.Close()

    // Use test server URL
    client := NewAPIClient(srv.URL)
    resp, err := client.GetData()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if resp.Status != "ok" {
        t.Errorf("status = %q; want ok", resp.Status)
    }
}
```

## Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out

# Require minimum coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//' | \
    xargs -I {} sh -c '[ {} -ge 80 ] || (echo "Coverage below 80%" && exit 1)'
```
