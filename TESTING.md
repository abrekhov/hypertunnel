# Testing Guide for HyperTunnel

This document describes the testing strategy, infrastructure, and guidelines for HyperTunnel development.

## Overview

HyperTunnel follows **Test-Driven Development (TDD)** principles to ensure code quality, reliability, and maintainability. All new features and bug fixes should be developed using TDD methodology.

## Test Coverage

Current test coverage by package:

| Package | Coverage | Status |
|---------|----------|--------|
| `pkg/datachannel` | ~60% | ✓ Good unit tests |
| `pkg/hashutils` | 83.3% | ✓ Excellent coverage |
| **Overall** | ~45% | 🎯 Target: 70%+ |

## Testing Infrastructure

### CI/CD Workflow

Automated testing runs on every push and pull request via GitHub Actions:

- **Test Workflow** (`.github/workflows/test.yaml`)
  - Runs on Linux, macOS, and Windows
  - Executes unit tests with race detection
  - Generates coverage reports
  - Uploads coverage to Codecov
  - Runs linting with golangci-lint
  - Checks code formatting

- **Release Workflow** (`.github/workflows/release.yaml`)
  - Triggered on version tags (`v*.*.*`)
  - Builds cross-platform binaries
  - Creates GitHub releases

### Linting Configuration

Code quality is enforced via golangci-lint (`.golangci.yaml`):

- Static analysis with `staticcheck`
- Security checks with `gosec`
- Code complexity analysis
- Enforces formatting with `gofmt` and `goimports`

## Running Tests

### Unit Tests

Run all unit tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -v -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt -o coverage.html
```

Run tests with race detection:
```bash
go test -race ./...
```

Run specific package tests:
```bash
go test -v ./pkg/datachannel/...
go test -v ./pkg/hashutils/...
```

### Integration Tests

Integration tests are tagged and run separately:

```bash
go test -v -tags=integration ./...
```

These tests verify end-to-end functionality like encryption/decryption round-trips and the local WebRTC datachannel transfer e2e test in `pkg/datachannel/webrtc_e2e_integration_test.go`.

### Benchmarks

Run performance benchmarks:
```bash
go test -bench=. ./pkg/...
go test -bench=BenchmarkEncode -benchmem ./pkg/datachannel/...
```

### Coverage Thresholds

The CI pipeline checks for minimum coverage:
- **Target:** 70% overall coverage
- **Warning:** Below 70% triggers a warning (non-blocking)
- **Best Practice:** New features should include tests achieving 80%+ coverage

## Test-Driven Development (TDD)

### TDD Workflow

1. **Write a Failing Test**
   - Define the expected behavior
   - Write a test that fails because the feature doesn't exist yet
   - Run the test to verify it fails

2. **Write Minimal Code**
   - Implement just enough code to make the test pass
   - Don't over-engineer or add extra features
   - Keep it simple

3. **Refactor**
   - Clean up the code while keeping tests passing
   - Improve structure, naming, and efficiency
   - Re-run tests after each refactor

4. **Repeat**
   - Continue with the next feature or test case

### TDD Example: Bug Fixes

When fixing bugs, always follow TDD:

**Example: Confirmation Bypass Bug (Fixed in Phase 1)**

1. **Write Test First:**
```go
func TestAskForConfirmation(t *testing.T) {
    t.Run("returns false for 'n' input", func(t *testing.T) {
        input := strings.NewReader("n\n")
        result := askForConfirmation("Test?", input)
        assert.False(t, result) // This fails due to bug
    })
}
```

2. **Fix the Bug:**
```go
func askForConfirmation(s string, in io.Reader) bool {
    // Remove: return true  ← Bug was here
    tries := 3
    // ... rest of implementation
}
```

3. **Verify Test Passes:**
```bash
go test -v ./pkg/datachannel/ -run TestAskForConfirmation
```

## Test Organization

### Directory Structure

```
hypertunnel/
├── pkg/
│   ├── datachannel/
│   │   ├── datachannel.go
│   │   ├── datachannel_test.go      # Unit tests for encode/decode
│   │   ├── handlers.go
│   │   └── handlers_test.go         # Unit tests for file handlers
│   └── hashutils/
│       ├── hashutils.go
│       └── hashutils_test.go        # Unit tests for key derivation
├── integration_test.go              # Integration tests (tagged)
└── coverage.txt                     # Coverage report
```

### Test File Naming

- Unit tests: `*_test.go` in the same package
- Integration tests: `integration_test.go` with build tag
- Benchmarks: Use `Benchmark*` function prefix
- Examples: Use `Example*` function prefix

## Writing Good Tests

### Test Structure

Use table-driven tests for multiple scenarios:

```go
func TestFromKeyToAESKey(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "empty string",
            input:    "",
            expected: "e3b0c442...",
        },
        {
            name:     "simple password",
            input:    "test",
            expected: "9f86d081...",
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result := FromKeyToAESKey(tc.input)
            assert.Equal(t, tc.expected, hex.EncodeToString(result))
        })
    }
}
```

### Test Naming

- **Format:** `Test<FunctionName>_<Scenario>`
- **Examples:**
  - `TestEncode` - Main test for Encode function
  - `TestDecode_InvalidInput` - Specific edge case
  - `TestAskForConfirmation_EmptyInput` - Behavior with empty input

### Assertions

Use `testify/assert` for readable assertions:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Use assert for non-critical checks
assert.Equal(t, expected, actual, "values should match")
assert.NotEmpty(t, result)
assert.True(t, condition)

// Use require for critical checks (stops test on failure)
require.NoError(t, err)
require.NotNil(t, object)
```

## Test Categories

### 1. Unit Tests

Test individual functions in isolation:

```go
func TestFromKeyToAESKey(t *testing.T) {
    key := FromKeyToAESKey("test-password")
    assert.Len(t, key, 32, "AES-256 requires 32-byte key")
}
```

### 2. Integration Tests

Test multiple components working together:

```go
//go:build integration

func TestEncryptDecryptRoundTrip(t *testing.T) {
    // Create file, encrypt, decrypt, verify content matches
}
```

### 3. Edge Case Tests

Test boundary conditions and error handling:

```go
func TestEncode_EmptySignal(t *testing.T) {
    signal := Signal{}
    encoded := Encode(signal)
    assert.NotEmpty(t, encoded)
}
```

### 4. Benchmarks

Measure performance:

```go
func BenchmarkEncode(b *testing.B) {
    signal := createTestSignal()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = Encode(signal)
    }
}
```

## Bug Fixes with TDD

All bug fixes must follow TDD:

### Phase 1 Bug Fixes (Completed)

#### 1. Confirmation Bypass Bug
- **Location:** `pkg/datachannel/handlers.go:44`
- **Issue:** Function returned `true` immediately, bypassing user confirmation
- **Test:** `TestAskForConfirmation` with various inputs
- **Fix:** Removed early `return true` statement
- **Status:** ✅ Fixed, all tests pass

#### 2. File Overwrite Check Bug
- **Location:** `pkg/datachannel/handlers.go:19`
- **Issue:** Used `os.IsExist(err)` incorrectly (should check `err == nil`)
- **Test:** `TestFileOverwriteCheck` demonstrates correct behavior
- **Fix:** Changed to `if err == nil` to detect existing files
- **Status:** ✅ Fixed, all tests pass

## Continuous Improvement

### Coverage Goals

Incrementally improve coverage:

1. **Current:** 45% overall
2. **Phase 1 Goal:** 60% (✓ Achieved)
3. **Phase 2 Goal:** 70%
4. **Phase 3 Goal:** 80%+

### Adding New Tests

When adding tests:

1. Start with the most critical paths
2. Add edge cases and error conditions
3. Include performance benchmarks for hot paths
4. Document complex test scenarios

### Test Maintenance

- Update tests when APIs change
- Remove obsolete tests
- Refactor tests to reduce duplication
- Keep tests simple and readable

## Common Test Patterns

### Testing with Temporary Files

```go
func TestFileOperation(t *testing.T) {
    tempDir := t.TempDir() // Auto-cleanup
    testFile := filepath.Join(tempDir, "test.txt")
    err := os.WriteFile(testFile, []byte("content"), 0644)
    require.NoError(t, err)
    // ... test logic
}
```

### Testing with Mock Input

```go
func TestUserInput(t *testing.T) {
    input := strings.NewReader("y\n")
    result := askForConfirmation("Question?", input)
    assert.True(t, result)
}
```

### Testing Error Conditions

```go
func TestErrorHandling(t *testing.T) {
    _, err := os.Stat("non-existent-file")
    assert.Error(t, err)
    assert.True(t, os.IsNotExist(err))
}
```

## Debugging Tests

### Verbose Output

```bash
go test -v ./pkg/datachannel/... -run TestAskForConfirmation
```

### Run Single Test

```bash
go test -v ./pkg/hashutils/... -run TestFromKeyToAESKey/empty_string
```

### Debug with Delve

```bash
dlv test ./pkg/datachannel -- -test.run TestEncode
```

### Print Debug Info

```go
func TestDebug(t *testing.T) {
    result := someFunction()
    t.Logf("Result: %+v", result) // Visible with -v flag
}
```

## Performance Testing

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./pkg/...

# Run specific benchmark
go test -bench=BenchmarkEncode ./pkg/datachannel/...

# With memory stats
go test -bench=. -benchmem ./pkg/...

# Compare before/after
go test -bench=. ./pkg/... > old.txt
# ... make changes ...
go test -bench=. ./pkg/... > new.txt
benchstat old.txt new.txt
```

## Integration with CI/CD

### Automated Checks

Every push triggers:
1. ✅ Unit tests on Linux, macOS, Windows
2. ✅ Integration tests on Ubuntu
3. ✅ Code linting and formatting
4. ✅ Coverage report generation
5. ✅ Coverage threshold check (70% warning)

### Pull Request Workflow

1. Create feature branch
2. Write tests (TDD)
3. Implement feature
4. Ensure all tests pass locally
5. Push to GitHub
6. CI runs automatically
7. Review coverage report
8. Merge when all checks pass

## Best Practices Summary

1. **Always write tests first** (TDD)
2. **Aim for 80%+ coverage** on new code
3. **Test edge cases** and error conditions
4. **Use table-driven tests** for multiple scenarios
5. **Keep tests simple** and readable
6. **Run tests before committing**
7. **Review coverage reports**
8. **Update tests when changing code**
9. **Document complex test scenarios**
10. **Use meaningful test names**

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Testify Package](https://github.com/stretchr/testify)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [TDD Best Practices](https://martinfowler.com/bliki/TestDrivenDevelopment.html)

## Questions?

For questions about testing:
- Read this guide
- Review existing tests as examples
- Check CI workflow configuration
- Open an issue if something is unclear

---

**Last Updated:** 2026-01-09
**Coverage Target:** 70%+
**TDD Approach:** ✅ Mandatory for all new features
