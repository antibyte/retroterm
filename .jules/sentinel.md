## 2026-01-24 - Hardcoded Admin Password
**Vulnerability:** The default "dyson" user was created with a hardcoded password "daniel" in `pkg/tinyos/db.go`. This meant every installation shared the same admin credentials.
**Learning:** Even when hashing passwords, the source entropy must not be static or hardcoded in the codebase.
**Prevention:** Always check for environment variables for initial secrets. If not provided, generate a cryptographically secure random value during initialization and log it securely for the admin to retrieve once.
