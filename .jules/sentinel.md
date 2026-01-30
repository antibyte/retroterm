## 2026-01-30 - False Positives in Secret Scanning
**Vulnerability:** Security tests flagged `os.Getenv` assignments as hardcoded secrets because the variable name contained "password" (e.g., `password := os.Getenv(...)`).
**Learning:** Regex-based secret scanning is prone to false positives when variable names match patterns intended for string literals.
**Prevention:** Avoid naming variables `password`, `secret`, `apiKey` etc. in sensitive contexts if they might be scanned by regex. Use specific names like `adminPass` or exclusions.
