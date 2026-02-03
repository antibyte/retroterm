## 2024-05-23 - Build Artifact Scanning
**Vulnerability:** False positives in security scans due to scanning minified build artifacts (`distjsretroterm.min.js`).
**Learning:** Security tests scanning the entire filesystem can flag high-entropy strings in minified code as secrets.
**Prevention:** Explicitly exclude build artifacts and `dist` directories from security test file walkers.

## 2024-05-23 - Configuration File Handling in Tests
**Vulnerability:** Flaky security tests when handling auto-generated configuration files (`settings.cfg`).
**Learning:** Development environments often generate config files that don't match production security templates (missing placeholders).
**Prevention:** Add logic to security tests to detect and skip auto-generated or development-specific configuration files.
