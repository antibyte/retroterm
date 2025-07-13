/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

/**
 * Authentication manager for TinyOS JWT token handling
 */
class AuthManager {    constructor() {
        this.baseUrl = window.location.origin;
        this.token = this.getStoredToken();
        this.sessionId = this.getStoredSessionId();
        
        // Clear temporary user tokens on browser refresh
        this.clearTemporaryTokenOnRefresh();
    }

    /**
     * Get token from localStorage
     */
    getStoredToken() {
        try {
            return localStorage.getItem('tinyos_token');
        } catch (e) {
            console.warn('[AUTH] localStorage not available, using memory storage');
            return null;
        }
    }

    /**
     * Store token in localStorage
     */
    setStoredToken(token) {
        try {
            if (token) {
                localStorage.setItem('tinyos_token', token);
            } else {
                localStorage.removeItem('tinyos_token');
            }
        } catch (e) {
            console.warn('[AUTH] localStorage not available for token storage');
        }
        this.token = token;
    }

    /**
     * Get session ID from localStorage
     */
    getStoredSessionId() {
        try {
            return localStorage.getItem('tinyos_session_id');
        } catch (e) {
            console.warn('[AUTH] localStorage not available, using memory storage');
            return null;
        }
    }

    /**
     * Store session ID in localStorage
     */
    setStoredSessionId(sessionId) {
        try {
            if (sessionId) {
                localStorage.setItem('tinyos_session_id', sessionId);
            } else {
                localStorage.removeItem('tinyos_session_id');
            }
        } catch (e) {
            console.warn('[AUTH] localStorage not available for session ID storage');
        }
        this.sessionId = sessionId;
    }    /**
     * Check if stored token belongs to a temporary user and clear it on browser refresh
     */
    clearTemporaryTokenOnRefresh() {
        if (!this.token) return false;
        
        try {
            // Decode token payload to check username
            const parts = this.token.split('.');
            if (parts.length !== 3) return false;
            
            const payload = JSON.parse(atob(parts[1]));
            if (payload && payload.username === 'dyson') {
                // console.log('[AUTH] Clearing temporary user token and session on browser refresh');
                this.setStoredToken(null);
                this.setStoredSessionId(null);
                return true;
            }
        } catch (error) {
            // Invalid token, clear it anyway
            this.setStoredToken(null);
            this.setStoredSessionId(null);
            return true;
        }
        
        return false;
    }    /**
     * Initialize authentication - check for existing token or create new session
     */
    async initialize() {
        // console.log('[AUTH] Initializing authentication...');
        
        // Try to validate existing token (temporary user tokens already cleared in constructor)
        if (this.token) {
            const valid = await this.validateToken();
            if (valid) {
                // console.log('[AUTH] Existing token is valid');
                return true;
            }
        }
        
        // console.log('[AUTH] No valid token found, creating new session...');
        
        // Create a new guest session
        const sessionId = await this.createSession();
        if (!sessionId) {
            console.error('[AUTH] Failed to create session');
            return false;
        }
        
        // Login with the new session ID to get JWT token
        const loginSuccess = await this.login(sessionId);
        if (loginSuccess) {
            // console.log('[AUTH] Successfully authenticated with new session');
            return true;
        }
        
        console.error('[AUTH] Failed to authenticate with new session');
        return false;
    }

    /**
     * Create a new guest session
     */
    async createSession() {
        try {
            const response = await fetch(`${this.baseUrl}/api/auth/session`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include', // Include cookies
                body: JSON.stringify({})
            });

            const data = await response.json();
            
            if (data.success) {
                // console.log('[AUTH] Session created:', data.sessionId);
                return data.sessionId;
            } else {
                console.error('[AUTH] Session creation failed:', data.message);
                return null;
            }
        } catch (error) {
            console.error('[AUTH] Session creation error:', error);
            return null;
        }
    }

    /**
     * Login with session ID and get JWT token
     */
    async login(sessionId) {
        // console.log('[AUTH] Logging in with session:', sessionId);
        
        try {
            const response = await fetch(`${this.baseUrl}/api/auth/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                credentials: 'include', // Include cookies
                body: JSON.stringify({
                    sessionId: sessionId
                })
            });

            const data = await response.json();
              if (data.success) {
                this.setStoredToken(data.token);
                this.setStoredSessionId(data.sessionId);
                // console.log('[AUTH] Login successful, token stored');
                return true;
            } else {
                console.error('[AUTH] Login failed:', data.message);
                return false;
            }
        } catch (error) {
            console.error('[AUTH] Login error:', error);
            return false;
        }
    }

    /**
     * Validate current token
     */
    async validateToken() {
        try {
            const response = await fetch(`${this.baseUrl}/api/auth/validate`, {
                method: 'GET',
                credentials: 'include', // Include cookies
                headers: {
                    'Authorization': this.token ? `Bearer ${this.token}` : ''
                }
            });

            const data = await response.json();
              if (data.success) {
                this.setStoredSessionId(data.sessionId);
                // console.log('[AUTH] Token validated for session:', this.sessionId);
                return true;
            } else {
                // console.log('[AUTH] Token validation failed:', data.message);
                this.setStoredToken(null);
                this.setStoredSessionId(null);
                return false;
            }        } catch (error) {
            console.error('[AUTH] Token validation error:', error);
            this.setStoredToken(null);
            this.setStoredSessionId(null);
            return false;
        }
    }

    /**
     * Logout and clear token
     */
    async logout() {
        // console.log('[AUTH] Logging out...');
        
        try {
            await fetch(`${this.baseUrl}/api/auth/logout`, {
                method: 'POST',
                credentials: 'include'
            });
        } catch (error) {
            console.error('[AUTH] Logout error:', error);
        }        
        this.setStoredToken(null);
        this.setStoredSessionId(null);
        // console.log('[AUTH] Logged out');
    }

    /**
     * Get current session ID
     */
    getSessionId() {
        return this.sessionId;
    }

    /**
     * Get current token
     */
    getToken() {
        return this.token;
    }

    /**
     * Check if authenticated
     */
    isAuthenticated() {
        return this.token !== null && this.sessionId !== null;
    }

    /**
     * Get WebSocket URL with authentication
     */
    getWebSocketUrl() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const host = window.location.host;
        
        if (this.token) {
            return `${protocol}//${host}/ws?token=${encodeURIComponent(this.token)}`;
        } else if (this.sessionId) {
            return `${protocol}//${host}/ws?sessionId=${encodeURIComponent(this.sessionId)}`;
        } else {
            return `${protocol}//${host}/ws`;
        }
    }
}

// Global auth manager instance
window.authManager = new AuthManager();
