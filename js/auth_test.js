/**
 * Frontend Authentication Tests for auth.js
 * These tests can be run in a browser console or with a JavaScript testing framework
 */

// Mock fetch function for testing
let mockResponses = {};
const originalFetch = window.fetch;

function mockFetch(url, options) {
    const key = `${options?.method || 'GET'} ${url}`;
    if (mockResponses[key]) {
        const response = mockResponses[key];
        return Promise.resolve({
            ok: response.ok !== false,
            status: response.status || 200,
            json: () => Promise.resolve(response.data)
        });
    }
    return originalFetch(url, options);
}

// Test suite for AuthManager
class AuthManagerTests {
    constructor() {
        this.tests = [];
        this.setupMocks();
    }

    setupMocks() {
        // Mock successful session creation
        mockResponses['POST /api/auth/session'] = {
            ok: true,
            data: {
                success: true,
                sessionId: 'test-session-123',
                message: 'Session created successfully'
            }
        };

        // Mock successful login
        mockResponses['POST /api/auth/login'] = {
            ok: true,
            data: {
                success: true,
                token: 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test.token',
                sessionId: 'test-session-123',
                message: 'Login successful'
            }
        };

        // Mock successful token validation
        mockResponses['GET /api/auth/validate'] = {
            ok: true,
            data: {
                success: true,
                sessionId: 'test-session-123',
                message: 'Token valid'
            }
        };

        // Mock successful logout
        mockResponses['POST /api/auth/logout'] = {
            ok: true,
            data: {
                success: true,
                message: 'Logged out successfully'
            }
        };

        // Replace fetch
        window.fetch = mockFetch;
    }

    tearDown() {
        // Restore original fetch
        window.fetch = originalFetch;
        mockResponses = {};
    }

    addTest(name, testFn) {
        this.tests.push({ name, testFn });
    }

    async runTests() {
        console.log('Starting AuthManager Tests...');
        let passed = 0;
        let failed = 0;

        for (const test of this.tests) {
            try {
                console.log(`Running: ${test.name}`);
                await test.testFn();
                console.log(`✅ PASS: ${test.name}`);
                passed++;
            } catch (error) {
                console.error(`❌ FAIL: ${test.name} - ${error.message}`);
                failed++;
            }
        }

        console.log(`\nTest Results: ${passed} passed, ${failed} failed`);
        this.tearDown();
        return { passed, failed };
    }

    // Helper assertion functions
    assertEqual(actual, expected, message = '') {
        if (actual !== expected) {
            throw new Error(`Expected ${expected}, got ${actual}. ${message}`);
        }
    }

    assertTrue(condition, message = '') {
        if (!condition) {
            throw new Error(`Expected true, got false. ${message}`);
        }
    }

    assertFalse(condition, message = '') {
        if (condition) {
            throw new Error(`Expected false, got true. ${message}`);
        }
    }

    assertNotNull(value, message = '') {
        if (value === null || value === undefined) {
            throw new Error(`Expected non-null value, got ${value}. ${message}`);
        }
    }
}

// Create test instance
const authTests = new AuthManagerTests();

// Test: AuthManager constructor
authTests.addTest('AuthManager constructor initializes correctly', async () => {
    const authManager = new AuthManager();
    
    authTests.assertNotNull(authManager.baseUrl, 'baseUrl should be set');
    authTests.assertEqual(authManager.token, null, 'token should be null initially');
    authTests.assertEqual(authManager.sessionId, null, 'sessionId should be null initially');
    authTests.assertEqual(authManager.baseUrl, window.location.origin, 'baseUrl should match window.location.origin');
});

// Test: Session creation
authTests.addTest('createSession creates a new session', async () => {
    const authManager = new AuthManager();
    
    const sessionId = await authManager.createSession();
    
    authTests.assertNotNull(sessionId, 'sessionId should not be null');
    authTests.assertEqual(sessionId, 'test-session-123', 'sessionId should match mock response');
});

// Test: Login with session ID
authTests.addTest('login with sessionId gets JWT token', async () => {
    const authManager = new AuthManager();
    
    const success = await authManager.login('test-session-123');
    
    authTests.assertTrue(success, 'login should return true');
    authTests.assertNotNull(authManager.token, 'token should be set after login');
    authTests.assertNotNull(authManager.sessionId, 'sessionId should be set after login');
    authTests.assertEqual(authManager.sessionId, 'test-session-123', 'sessionId should match login parameter');
});

// Test: Token validation
authTests.addTest('validateToken validates existing token', async () => {
    const authManager = new AuthManager();
    authManager.token = 'test-token';
    
    const valid = await authManager.validateToken();
    
    authTests.assertTrue(valid, 'validateToken should return true for valid token');
    authTests.assertEqual(authManager.sessionId, 'test-session-123', 'sessionId should be set from validation');
});

// Test: Token validation with invalid token
authTests.addTest('validateToken fails with invalid token', async () => {
    // Mock failed validation
    mockResponses['GET /api/auth/validate'] = {
        ok: false,
        status: 401,
        data: {
            success: false,
            message: 'Invalid token'
        }
    };

    const authManager = new AuthManager();
    authManager.token = 'invalid-token';
    
    const valid = await authManager.validateToken();
    
    authTests.assertFalse(valid, 'validateToken should return false for invalid token');
    authTests.assertEqual(authManager.token, null, 'token should be cleared after failed validation');
    authTests.assertEqual(authManager.sessionId, null, 'sessionId should be cleared after failed validation');

    // Restore mock
    mockResponses['GET /api/auth/validate'] = {
        ok: true,
        data: {
            success: true,
            sessionId: 'test-session-123',
            message: 'Token valid'
        }
    };
});

// Test: Authentication status
authTests.addTest('isAuthenticated returns correct status', async () => {
    const authManager = new AuthManager();
    
    // Initially not authenticated
    authTests.assertFalse(authManager.isAuthenticated(), 'should not be authenticated initially');
    
    // Set token and session
    authManager.token = 'test-token';
    authManager.sessionId = 'test-session';
    
    authTests.assertTrue(authManager.isAuthenticated(), 'should be authenticated with token and session');
    
    // Clear token
    authManager.token = null;
    
    authTests.assertFalse(authManager.isAuthenticated(), 'should not be authenticated without token');
});

// Test: WebSocket URL generation
authTests.addTest('getWebSocketUrl generates correct URLs', async () => {
    const authManager = new AuthManager();
    const expectedProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const expectedHost = window.location.host;
    
    // No token or session
    const url1 = authManager.getWebSocketUrl();
    const expected1 = `${expectedProtocol}//${expectedHost}/ws`;
    authTests.assertEqual(url1, expected1, 'URL should be basic WebSocket URL without parameters');
    
    // With session only
    authManager.sessionId = 'test-session';
    const url2 = authManager.getWebSocketUrl();
    const expected2 = `${expectedProtocol}//${expectedHost}/ws?sessionId=test-session`;
    authTests.assertEqual(url2, expected2, 'URL should include sessionId parameter');
    
    // With token (should take precedence)
    authManager.token = 'test-token';
    const url3 = authManager.getWebSocketUrl();
    const expected3 = `${expectedProtocol}//${expectedHost}/ws?token=test-token`;
    authTests.assertEqual(url3, expected3, 'URL should include token parameter when available');
});

// Test: Logout functionality
authTests.addTest('logout clears authentication state', async () => {
    const authManager = new AuthManager();
    
    // Set up authenticated state
    authManager.token = 'test-token';
    authManager.sessionId = 'test-session';
    
    await authManager.logout();
    
    authTests.assertEqual(authManager.token, null, 'token should be cleared after logout');
    authTests.assertEqual(authManager.sessionId, null, 'sessionId should be cleared after logout');
    authTests.assertFalse(authManager.isAuthenticated(), 'should not be authenticated after logout');
});

// Test: Full initialization flow
authTests.addTest('initialize handles full authentication flow', async () => {
    // Mock failed validation (no existing token)
    mockResponses['GET /api/auth/validate'] = {
        ok: false,
        status: 401,
        data: {
            success: false,
            message: 'No token found'
        }
    };

    const authManager = new AuthManager();
    
    const success = await authManager.initialize();
    
    authTests.assertTrue(success, 'initialize should succeed');
    authTests.assertNotNull(authManager.token, 'token should be set after initialization');
    authTests.assertNotNull(authManager.sessionId, 'sessionId should be set after initialization');
    authTests.assertTrue(authManager.isAuthenticated(), 'should be authenticated after initialization');

    // Restore mock
    mockResponses['GET /api/auth/validate'] = {
        ok: true,
        data: {
            success: true,
            sessionId: 'test-session-123',
            message: 'Token valid'
        }
    };
});

// Test: Error handling
authTests.addTest('handles network errors gracefully', async () => {
    // Mock network error
    const originalFetch = window.fetch;
    window.fetch = () => Promise.reject(new Error('Network error'));
    
    const authManager = new AuthManager();
    
    const sessionId = await authManager.createSession();
    authTests.assertEqual(sessionId, null, 'createSession should return null on network error');
    
    const loginSuccess = await authManager.login('test-session');
    authTests.assertFalse(loginSuccess, 'login should return false on network error');
    
    const validationSuccess = await authManager.validateToken();
    authTests.assertFalse(validationSuccess, 'validateToken should return false on network error');
    
    // Restore fetch
    window.fetch = originalFetch;
});

// Export for use in browser console or test runners
if (typeof module !== 'undefined' && module.exports) {
    module.exports = { AuthManagerTests, authTests };
}

// Auto-run tests if in browser environment
if (typeof window !== 'undefined' && window.AuthManager) {
    console.log('AuthManager detected, running tests...');
    authTests.runTests().then(results => {
        console.log('Auth tests completed:', results);
    });
} else {
    console.log('To run tests, load auth.js first, then run: authTests.runTests()');
}
