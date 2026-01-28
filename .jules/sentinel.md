## 2026-01-28 - Hardcoded Default Credentials
**Vulnerability:** The default admin user "dyson" was created with a hardcoded password "daniel" in `pkg/tinyos/db.go`.
**Learning:** AI-generated code or quick prototypes often contain hardcoded "placeholder" credentials that persist into production.
**Prevention:** Always use environment variables for initial credentials. Implement a "first run" check that generates random credentials if none are provided.
