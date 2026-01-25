## 2026-01-25 - Hardcoded Admin Password Removal
**Vulnerability:** The default admin user "dyson" was created with a hardcoded password "daniel" in the compiled binary. This meant any deployment of the software would have a known, insecure admin account unless manually changed immediately.
**Learning:** Hardcoded credentials for default accounts often slip through regex scans if they don't look like standard keys (e.g., short strings, non-assignment syntax). `bcrypt.GenerateFromPassword([]byte("daniel"), ...)` was missed by `password.*=.*` regexes.
**Prevention:**
1. Always use environment variables for initial credentials.
2. Implement "random by default" logic where secrets are generated at runtime if not configured.
3. Add specific checks for known default values in security tests, not just generic patterns.
