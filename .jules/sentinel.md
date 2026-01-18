# Sentinel's Journal

## 2025-05-24 - Missing Security Middleware
**Vulnerability:** The application was serving HTTP responses without standard security headers (Content-Security-Policy, X-Frame-Options, X-Content-Type-Options, etc.).
**Learning:** `pkg/middleware` was referenced in memory as existing but was missing from the codebase, leaving the application unprotected against XSS, clickjacking, and MIME sniffing.
**Prevention:** Always verify that referenced security components actually exist in the codebase. Implemented `pkg/middleware/security.go` to enforce these headers globally.
