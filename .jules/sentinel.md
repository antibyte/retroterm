## 2025-10-26 - [Hardcoded Default User Creation]
**Vulnerability:** The function `CreateDefaultUsers` in `pkg/tinyos/db.go` creates a default user "dyson" with a hardcoded password "daniel" if it doesn't exist.
**Learning:** Even though `settings.cfg` suggests using environment variables for secrets, the application initialization logic has hardcoded fallback credentials for a specific user, which is also an admin.
**Prevention:** Initialization logic should require secrets to be provided via environment variables and fail or generate random secure credentials if not present, rather than falling back to hardcoded values.

## 2025-10-26 - [Missing Middleware Architecture]
**Vulnerability:** The application lacked a centralized middleware system for applying HTTP security headers, relying on default `http.ServeMux` behavior.
**Learning:** Security headers like CSP and HSTS were completely missing, exposing the application to XSS and clickjacking.
**Prevention:** Implemented a reusable `middleware` package to wrap the main handler, enforcing security headers globally.
