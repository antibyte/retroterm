## 2026-01-16 - [Security Headers Implementation]
**Vulnerability:** Missing HTTP security headers (X-Content-Type-Options, X-Frame-Options, CSP) left the application vulnerable to MIME sniffing, Clickjacking, and XSS.
**Learning:** Legacy frontend code using `unsafe-inline` and `unsafe-eval` requires a carefully crafted Content Security Policy (CSP) that balances security with compatibility. Specifically, `blob:` and `data:` schemes were needed for audio/image assets.
**Prevention:** Implemented a reusable `WithSecurityHeaders` middleware in `pkg/middleware` that is applied to all HTTP/HTTPS handlers in `main.go`. This ensures consistent security posture across all endpoints.
