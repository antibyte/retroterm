## 2024-05-23 - Hardcoded Admin Credentials
**Vulnerability:** The default admin user 'dyson' had a hardcoded password 'daniel' in the source code.
**Learning:** Hardcoded credentials in initialization logic can easily be overlooked.
**Prevention:** Always read initial credentials from configuration or generate them randomly at runtime.
