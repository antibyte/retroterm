## 2026-01-12 - Fixed Hardcoded JWT Secret
**Vulnerability:** A hardcoded fallback JWT secret ('fallback_secret_change_in_production') was present in 'pkg/auth/jwt.go' and used as a default if no configuration was provided.
**Learning:** Hardcoded fallbacks, even with warnings, are dangerous because they can easily end up in production if configuration is missed. Automated security scanners (regex-based) are useful but can produce false positives on variable names or build artifacts (like 'distjsretroterm.min.js').
**Prevention:** Use 'crypto/rand' to generate secure random secrets at runtime when no configuration is provided. This fails securely (invalidating sessions on restart) rather than remaining vulnerable. Exclude build artifacts from source code scans.
