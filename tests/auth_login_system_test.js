/**
 * Comprehensive tests for the authentication and login system
 * Tests temporary user behavior, session management, and token handling
 */

// Test framework setup
const testResults = [];
let testCount = 0;
let passedTests = 0;

function assert(condition, message) {
    testCount++;
    if (condition) {
        passedTests++;
        console.log(`✅ Test ${testCount}: ${message}`);
        testResults.push({ test: testCount, message, passed: true });
    } else {
        console.error(`❌ Test ${testCount}: ${message}`);
        testResults.push({ test: testCount, message, passed: false });
        throw new Error(`Test failed: ${message}`);
    }
}

function assertEquals(actual, expected, message) {
    assert(actual === expected, `${message} (expected: ${expected}, actual: ${actual})`);
}

function assertNotEquals(actual, unexpected, message) {
    assert(actual !== unexpected, `${message} (should not be: ${unexpected}, actual: ${actual})`);
}

function assertExists(value, message) {
    assert(value !== null && value !== undefined, message);
}

function assertNotExists(value, message) {
    assert(value === null || value === undefined, message);
}

// Test suite for AuthManager
class AuthLoginSystemTests {
    
    async setUp() {
        // Reset localStorage before each test
        if (typeof localStorage !== 'undefined') {
            localStorage.clear();
        }
        
        // Mock the global environment
        if (typeof window === 'undefined') {
            global.window = {
                location: { origin: 'http://localhost:8080' },
                localStorage: {
                    data: {},
                    getItem: function(key) { return this.data[key] || null; },
                    setItem: function(key, value) { this.data[key] = value; },
                    removeItem: function(key) { delete this.data[key]; },
                    clear: function() { this.data = {}; }
                }
            };
            global.localStorage = window.localStorage;
        }
    }

    async tearDown() {
        // Clean up after each test
        if (typeof localStorage !== 'undefined') {
            localStorage.clear();
        }
    }

    // Test AuthManager constructor and initialization
    async testAuthManagerConstructor() {
        await this.setUp();
        
        const authManager = new AuthManager();
        
        assertExists(authManager, "AuthManager instance should be created");
        assertEquals(authManager.baseUrl, 'http://localhost:8080', "Base URL should be set correctly");
        assertNotExists(authManager.token, "Token should be null initially");
        assertNotExists(authManager.sessionId, "Session ID should be null initially");
        
        await this.tearDown();
    }

    // Test temporary token clearing on refresh
    async testTemporaryTokenClearingOnRefresh() {
        await this.setUp();
        
        // Create a mock JWT token for 'dyson'
        const mockDysonToken = this.createMockJWTToken('dyson', 'user_12345');
        
        // Store token and session in localStorage
        localStorage.setItem('tinyos_token', mockDysonToken);
        localStorage.setItem('tinyos_session_id', 'user_12345');
        
        // Create AuthManager (this should trigger clearTemporaryTokenOnRefresh)
        const authManager = new AuthManager();
        
        // Token and session should be cleared for dyson
        assertNotExists(authManager.token, "Dyson token should be cleared on refresh");
        assertNotExists(authManager.sessionId, "Dyson session should be cleared on refresh");
        assertNotExists(localStorage.getItem('tinyos_token'), "Token should be removed from localStorage");
        assertNotExists(localStorage.getItem('tinyos_session_id'), "Session ID should be removed from localStorage");
        
        await this.tearDown();
    }

    // Test that regular user tokens are NOT cleared on refresh
    async testRegularUserTokenNotClearedOnRefresh() {
        await this.setUp();
        
        // Create a mock JWT token for regular user
        const mockUserToken = this.createMockJWTToken('alice', 'user_67890');
        
        // Store token and session in localStorage
        localStorage.setItem('tinyos_token', mockUserToken);
        localStorage.setItem('tinyos_session_id', 'user_67890');
        
        // Create AuthManager
        const authManager = new AuthManager();
        
        // Token and session should NOT be cleared for regular users
        assertExists(authManager.token, "Regular user token should NOT be cleared on refresh");
        assertExists(authManager.sessionId, "Regular user session should NOT be cleared on refresh");
        assertEquals(authManager.token, mockUserToken, "Token should match stored token");
        assertEquals(authManager.sessionId, 'user_67890', "Session ID should match stored session");
        
        await this.tearDown();
    }

    // Test WebSocket URL generation with token
    async testWebSocketUrlWithToken() {
        await this.setUp();
        
        const authManager = new AuthManager();
        const mockToken = this.createMockJWTToken('alice', 'user_67890');
        authManager.token = mockToken;
        authManager.sessionId = 'user_67890';
        
        const wsUrl = authManager.getWebSocketUrl();
        
        assert(wsUrl.includes('token='), "WebSocket URL should include token parameter");
        assert(wsUrl.includes(encodeURIComponent(mockToken)), "WebSocket URL should include encoded token");
        assert(wsUrl.startsWith('ws://'), "WebSocket URL should use ws protocol");
        
        await this.tearDown();
    }

    // Test WebSocket URL generation with session ID fallback
    async testWebSocketUrlWithSessionIdFallback() {
        await this.setUp();
        
        const authManager = new AuthManager();
        authManager.token = null; // No token
        authManager.sessionId = 'guest_12345';
        
        const wsUrl = authManager.getWebSocketUrl();
        
        assert(wsUrl.includes('sessionId='), "WebSocket URL should include sessionId parameter when no token");
        assert(wsUrl.includes('guest_12345'), "WebSocket URL should include session ID");
        
        await this.tearDown();
    }

    // Test that clearTemporaryTokenOnRefresh works correctly
    async testClearTemporaryTokenOnRefreshFunction() {
        await this.setUp();
        
        const authManager = new AuthManager();
        
        // Test with dyson token
        const dysonToken = this.createMockJWTToken('dyson', 'temp_12345');
        authManager.token = dysonToken;
        authManager.sessionId = 'temp_12345';
        
        const cleared = authManager.clearTemporaryTokenOnRefresh();
        
        assert(cleared, "clearTemporaryTokenOnRefresh should return true for dyson");
        assertNotExists(authManager.token, "Token should be cleared");
        assertNotExists(authManager.sessionId, "Session ID should be cleared");
        
        // Test with regular user token
        const userToken = this.createMockJWTToken('alice', 'user_67890');
        authManager.token = userToken;
        authManager.sessionId = 'user_67890';
        
        const notCleared = authManager.clearTemporaryTokenOnRefresh();
        
        assert(!notCleared, "clearTemporaryTokenOnRefresh should return false for regular users");
        assertExists(authManager.token, "Regular user token should not be cleared");
        assertExists(authManager.sessionId, "Regular user session should not be cleared");
        
        await this.tearDown();
    }

    // Test token storage and retrieval
    async testTokenStorageAndRetrieval() {
        await this.setUp();
        
        const authManager = new AuthManager();
        const testToken = 'test_token_12345';
        
        // Test storing token
        authManager.setStoredToken(testToken);
        assertEquals(authManager.token, testToken, "Token should be set in memory");
        assertEquals(localStorage.getItem('tinyos_token'), testToken, "Token should be stored in localStorage");
        
        // Test retrieving token
        const retrievedToken = authManager.getStoredToken();
        assertEquals(retrievedToken, testToken, "Retrieved token should match stored token");
        
        // Test clearing token
        authManager.setStoredToken(null);
        assertNotExists(authManager.token, "Token should be cleared from memory");
        assertNotExists(localStorage.getItem('tinyos_token'), "Token should be removed from localStorage");
        
        await this.tearDown();
    }

    // Test session ID storage and retrieval
    async testSessionIdStorageAndRetrieval() {
        await this.setUp();
        
        const authManager = new AuthManager();
        const testSessionId = 'test_session_12345';
        
        // Test storing session ID
        authManager.setStoredSessionId(testSessionId);
        assertEquals(authManager.sessionId, testSessionId, "Session ID should be set in memory");
        assertEquals(localStorage.getItem('tinyos_session_id'), testSessionId, "Session ID should be stored in localStorage");
        
        // Test retrieving session ID
        const retrievedSessionId = authManager.getStoredSessionId();
        assertEquals(retrievedSessionId, testSessionId, "Retrieved session ID should match stored session ID");
        
        // Test clearing session ID
        authManager.setStoredSessionId(null);
        assertNotExists(authManager.sessionId, "Session ID should be cleared from memory");
        assertNotExists(localStorage.getItem('tinyos_session_id'), "Session ID should be removed from localStorage");
        
        await this.tearDown();
    }

    // Test authentication status check
    async testAuthenticationStatus() {
        await this.setUp();
        
        const authManager = new AuthManager();
        
        // Test not authenticated initially
        assert(!authManager.isAuthenticated(), "Should not be authenticated initially");
        
        // Test authenticated with token and session
        authManager.token = 'test_token';
        authManager.sessionId = 'test_session';
        assert(authManager.isAuthenticated(), "Should be authenticated with token and session");
        
        // Test not authenticated with only token
        authManager.sessionId = null;
        assert(!authManager.isAuthenticated(), "Should not be authenticated with only token");
        
        // Test not authenticated with only session
        authManager.token = null;
        authManager.sessionId = 'test_session';
        assert(!authManager.isAuthenticated(), "Should not be authenticated with only session");
        
        await this.tearDown();
    }

    // Test invalid token handling
    async testInvalidTokenHandling() {
        await this.setUp();
        
        const authManager = new AuthManager();
        
        // Test with invalid token format
        authManager.token = 'invalid_token';
        const cleared = authManager.clearTemporaryTokenOnRefresh();
        
        assert(cleared, "Should clear invalid token");
        assertNotExists(authManager.token, "Invalid token should be cleared");
        
        await this.tearDown();
    }

    // Helper method to create mock JWT tokens
    createMockJWTToken(username, sessionId) {
        const header = { alg: 'HS256', typ: 'JWT' };
        const payload = { 
            username: username, 
            sessionId: sessionId,
            exp: Math.floor(Date.now() / 1000) + (60 * 60) // 1 hour from now
        };
        
        // Create base64 encoded parts (simplified mock)
        const headerB64 = btoa(JSON.stringify(header));
        const payloadB64 = btoa(JSON.stringify(payload));
        const signature = 'mock_signature';
        
        return `${headerB64}.${payloadB64}.${signature}`;
    }

    // Run all tests
    async runAllTests() {
        console.log("🚀 Starting Authentication and Login System Tests...\n");
        
        const tests = [
            'testAuthManagerConstructor',
            'testTemporaryTokenClearingOnRefresh',
            'testRegularUserTokenNotClearedOnRefresh',
            'testWebSocketUrlWithToken',
            'testWebSocketUrlWithSessionIdFallback',
            'testClearTemporaryTokenOnRefreshFunction',
            'testTokenStorageAndRetrieval',
            'testSessionIdStorageAndRetrieval',
            'testAuthenticationStatus',
            'testInvalidTokenHandling'
        ];

        for (const testName of tests) {
            try {
                console.log(`\n📋 Running ${testName}...`);
                await this[testName]();
                console.log(`✅ ${testName} passed`);
            } catch (error) {
                console.error(`❌ ${testName} failed:`, error.message);
            }
        }

        console.log(`\n📊 Test Results: ${passedTests}/${testCount} tests passed`);
        
        if (passedTests === testCount) {
            console.log("🎉 All tests passed!");
        } else {
            console.log(`⚠️  ${testCount - passedTests} tests failed`);
        }

        return { total: testCount, passed: passedTests, results: testResults };
    }
}

// Add btoa function for Node.js environment if not available
if (typeof btoa === 'undefined') {
    global.btoa = function(str) {
        return Buffer.from(str, 'binary').toString('base64');
    };
}

// Export for both Node.js and browser environments
if (typeof module !== 'undefined' && module.exports) {
    module.exports = AuthLoginSystemTests;
} else if (typeof window !== 'undefined') {
    window.AuthLoginSystemTests = AuthLoginSystemTests;
}

// Auto-run tests if this file is executed directly
if (typeof require !== 'undefined' && require.main === module) {
    // Load AuthManager for testing (adjust path as needed)
    try {
        const AuthManager = require('../js/auth.js');
        global.AuthManager = AuthManager;
    } catch (error) {
        console.error("Could not load AuthManager for testing:", error.message);
        console.log("Please run tests in browser environment or adjust require path");
    }
    
    const testSuite = new AuthLoginSystemTests();
    testSuite.runAllTests().then(results => {
        process.exit(results.passed === results.total ? 0 : 1);
    });
}
