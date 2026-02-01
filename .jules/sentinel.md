## 2026-02-01 - False Positives in Regex-based Security Scanning
**Vulnerability:** The `TestNoHardcodedSecrets` in `tests/security_test.go` flagged a variable assignment `password := os.Getenv(...)` as a hardcoded secret because the regex `password.*=.*` is too broad and matched the variable name combined with the assignment. It also flagged minified JS artifacts.
**Learning:** Regex-based secret scanning requires careful tuning. Variable names containing "password" can trigger false positives even when used securely (e.g. reading from env). Build artifacts must be explicitly excluded.
**Prevention:** Use specific variable names (e.g. `adminPwd` instead of `password`) to avoid regex matches. Ensure all build artifacts are in the exclusion list of the security scanner.
