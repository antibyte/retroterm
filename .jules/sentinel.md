## 2026-01-31 - HTTP Security Headers Implementation
**Vulnerability:** Missing HTTP security headers (CSP, X-Frame-Options, X-Content-Type-Options) exposing the application to XSS, Clickjacking, and MIME sniffing.
**Learning:** Standard Go `http.ListenAndServe` does not apply security headers by default. Explicit middleware is required.
**Prevention:** Implement a `SecurityHeaders` middleware that wraps the default mux and enforces secure defaults.
