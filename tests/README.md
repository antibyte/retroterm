# Authentication & Login System Tests

This directory contains comprehensive tests for the authentication and login system, specifically testing the temporary user behavior and session management.

## Test Coverage

### Frontend Tests (`auth_login_system_test.js` & `auth_login_system_test.html`)

**Core Authentication Features:**
- ✅ AuthManager constructor and initialization
- ✅ Token storage and retrieval from localStorage
- ✅ Session ID management
- ✅ Authentication status validation
- ✅ WebSocket URL generation with authentication

**Temporary User Behavior:**
- ✅ Automatic token clearing for "dyson" on browser refresh
- ✅ Regular user token persistence after refresh  
- ✅ Session ID fallback handling
- ✅ Invalid token handling
- ✅ Security boundary testing

**WebSocket Authentication:**
- ✅ JWT token authentication
- ✅ Session ID fallback when no token available
- ✅ URL parameter encoding

### Backend Tests (`auth_login_system_test.go`)

**Token Management:**
- ✅ Temporary user identification (`IsTemporaryUser` function)
- ✅ JWT token generation for temporary users
- ✅ JWT token generation for regular users
- ✅ Token validation and claim verification
- ✅ Token expiration timing (15 minutes for temporary, 24h for regular)

**Session Management:**
- ✅ Session restoration blocking for temporary users
- ✅ Cookie settings for temporary vs regular users
- ✅ Security boundary testing
- ✅ Performance benchmarks

## Running the Tests

### Frontend Tests (Browser)

1. **Interactive HTML Test Runner:**
   ```
   http://localhost:8080/tests/auth_login_system_test.html
   ```
   - Click "▶️ Run All Tests" to execute the full test suite
   - Use "🎯 Run Specific Tests" to run individual tests
   - View real-time console output and detailed results

2. **Manual JavaScript Testing:**
   ```javascript
   // In browser console
   const testSuite = new AuthLoginSystemTests();
   testSuite.runAllTests();
   ```

### Backend Tests (Go)

1. **Run all authentication tests:**
   ```bash
   go test -v ./tests/ -run TestAuth
   ```

2. **Run specific test:**
   ```bash
   go test -v ./tests/ -run TestTemporaryUserSessionManagement
   ```

3. **Run with benchmarks:**
   ```bash
   go test -v ./tests/ -bench=BenchmarkToken
   ```

## Key Test Scenarios

### Temporary User ("dyson") Behavior

1. **Browser Refresh Scenario:**
   - User logs in as "dyson" → receives temporary JWT token
   - Browser refresh → token automatically cleared from localStorage
   - New session starts as guest (not logged in)
   - Commands work normally with new guest session

2. **Session Restoration Blocking:**
   - "dyson" session exists in memory
   - Frontend attempts to restore via sessionId
   - Backend blocks restoration for temporary users
   - New guest session created instead

3. **Token Properties:**
   - Temporary tokens have `IsTempToken: true` claim
   - Short expiration (15 minutes vs 24 hours)
   - Secure cookie settings with limited lifetime

### Regular User Behavior

1. **Session Persistence:**
   - User logs in → receives 24-hour JWT token
   - Browser refresh → token persists in localStorage
   - Session restored successfully if token valid
   - Commands continue working with restored session

2. **Token Properties:**
   - Regular tokens have `IsTempToken: false` claim
   - 24-hour expiration
   - Standard cookie settings

## Test Structure

### Frontend Test Framework

```javascript
// Custom assertion framework
function assert(condition, message) { /* ... */ }
function assertEquals(actual, expected, message) { /* ... */ }
function assertExists(value, message) { /* ... */ }

// Test class with setup/teardown
class AuthLoginSystemTests {
    async setUp() { /* Clean localStorage */ }
    async tearDown() { /* Cleanup */ }
    async testSpecificFeature() { /* Test implementation */ }
}
```

### Backend Test Structure

```go
// Standard Go testing framework
func TestFeatureName(t *testing.T) {
    // Test implementation
    if !condition {
        t.Errorf("Expected %v, got %v", expected, actual)
    }
}

// Benchmark tests
func BenchmarkFeature(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // Benchmark implementation
    }
}
```

## Test Data & Mocks

### Mock JWT Tokens
- Header: `{ "alg": "HS256", "typ": "JWT" }`
- Payload: `{ "username": "...", "sessionId": "...", "exp": ... }`
- Signature: `"mock_signature"` (for testing only)

### Test Users
- **dyson**: Temporary user (special behavior)
- **alice**: Regular user (normal behavior)
- **bob**: Regular user (normal behavior)

### Test Sessions
- **temp_session_123**: Temporary session
- **user_session_456**: Regular user session
- **guest_session_789**: Guest session

## Security Considerations

### What is Tested
- ✅ Temporary user identification is case-sensitive
- ✅ Whitespace handling in usernames
- ✅ Token claim validation
- ✅ Session restoration blocking
- ✅ Cookie security settings
- ✅ XSS protection (HttpOnly cookies)
- ✅ CSRF protection (SameSite cookies)

### What is NOT Tested (Production Considerations)
- ❌ Actual JWT signature validation (uses mock signatures)
- ❌ HTTPS enforcement in production
- ❌ Rate limiting
- ❌ IP binding validation
- ❌ Database persistence
- ❌ Concurrent session handling

## Continuous Integration

These tests should be run:
- ✅ Before every deployment
- ✅ On every pull request
- ✅ Daily as part of automated testing
- ✅ After any authentication-related changes

## Troubleshooting

### Common Issues

1. **"localStorage not available" errors:**
   - Tests automatically handle this with fallback
   - Mock localStorage is created for Node.js environment

2. **"AuthManager not found" errors:**
   - Ensure `auth.js` is loaded before test files
   - Check file paths in HTML test runner

3. **Test timing issues:**
   - Tests use mock time for token expiration
   - Real-time tests may need adjustment for slow systems

4. **Browser compatibility:**
   - Tests require modern browser with localStorage support
   - ES6 features are used (async/await, arrow functions)

### Debug Mode

Enable debug output in tests:
```javascript
// In browser console
window.DEBUG_AUTH_TESTS = true;
```

### Performance Monitoring

Benchmark results help monitor:
- Token generation speed
- Token validation performance  
- Memory usage during test runs
- Network request timing (for integration tests)

## Maintenance

### Adding New Tests

1. **Frontend tests:**
   - Add method to `AuthLoginSystemTests` class
   - Update `runAllTests()` method to include new test
   - Add description to HTML test runner

2. **Backend tests:**
   - Create new `TestFeatureName` function
   - Follow Go testing conventions
   - Add benchmark if performance-critical

3. **Update documentation:**
   - Add test description to this README
   - Update test coverage section
   - Document any new test data or scenarios

### When to Update Tests

- ✅ New authentication features added
- ✅ Security improvements implemented
- ✅ Bug fixes in authentication logic
- ✅ Changes to token structure or claims
- ✅ Session management modifications
- ✅ WebSocket authentication changes
