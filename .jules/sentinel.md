## 2025-05-20 - Missing Security Middleware
**Vulnerability:** Missing Security Headers (CSP, X-Frame-Options, etc.) despite expectations.
**Learning:** The codebase was expected to have `pkg/middleware` with security headers, but it was completely missing. This highlights the importance of verifying the actual codebase state against documentation or assumptions.
**Prevention:** Always audit the codebase for the actual implementation of security controls, regardless of what documentation or previous knowledge suggests.
