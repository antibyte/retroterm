## 2024-05-23 - Hardcoded Default Credentials
**Vulnerability:** Found hardcoded credentials (`dyson:daniel`) in `pkg/tinyos/db.go`. This account had admin privileges and was created by default if it didn't exist.
**Learning:** Hardcoded credentials in source code are a major risk as they are visible to anyone with code access and often remain unchanged in production.
**Prevention:** Always use configuration files or environment variables for initial credentials. Ensure default installations do not create accounts with known passwords unless explicitly configured by the user.
