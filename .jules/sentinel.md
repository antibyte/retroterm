## 2026-01-26 - Retro Terminal CSP Requirements
**Vulnerability:** XSS (mitigated by CSP)
**Learning:** The retroterm frontend heavily relies on inline scripts, styles, and potentially `eval` (via dependencies like samjs or just legacy code). A strict CSP breaks the application.
**Prevention:** Use a CSP that allows `'unsafe-inline'` and `'unsafe-eval'` for scripts/styles but restricts sources to `'self'` and specific trusted CDNs (Google Fonts, webonastick).
