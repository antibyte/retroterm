## 2026-02-05 - Hardcoded Default Admin Credentials
**Vulnerability:** Found a hardcoded password ("daniel") for the default "dyson" user in `pkg/tinyos/db.go`. This would allow anyone with knowledge of the codebase to gain admin access to any deployment.
**Learning:** Default users created during initialization are often overlooked. The comment "his son's name" suggests this was a personal touch or placeholder that made it into production code.
**Prevention:** Always use environment variables for initial secrets (`INITIAL_ADMIN_PASSWORD`). If not provided, generate a secure random password and log it (once) or require manual setup. Added a test case `TestSecurity_DefaultUserPassword` to enforce this policy.
