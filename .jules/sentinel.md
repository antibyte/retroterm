## 2025-02-02 - Missing Security Headers
**Vulnerability:** Application was missing standard HTTP security headers (CSP, X-Frame-Options, etc.).
**Learning:** Default Go HTTP server does not set security headers by default. Middleware is required.
**Prevention:** Use `pkg/middleware/security.go` to wrap all HTTP handlers.
