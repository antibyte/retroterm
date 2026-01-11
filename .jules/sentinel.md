## 2024-05-23 - Missing HTTP Security Headers
**Vulnerability:** The application was missing standard HTTP security headers (CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy).
**Learning:** These headers are often overlooked in Go applications because they are not set by default in `net/http` and require manual middleware implementation.
**Prevention:** I implemented a `SecurityHeadersMiddleware` in `pkg/middleware/security.go` and applied it to the global handler in `main.go`. I also added a test to verify the headers are set correctly. The CSP is configured to allow 'unsafe-inline' and 'unsafe-eval' which is necessary for the current architecture but should be tightened in the future.
