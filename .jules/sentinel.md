## 2024-05-22 - Content Security Policy Configuration
**Vulnerability:** Cross-Site Scripting (XSS) and Data Injection risks.
**Learning:** The application requires specific external resources (Google Fonts, webonastick.com) and relies on inline scripts and styles, making a strict CSP challenging without refactoring.
**Prevention:** Use a CSP that explicitly allows these sources while restricting others. The working policy is: `default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; font-src 'self' https://fonts.gstatic.com https://webonastick.com data:; connect-src 'self' ws: wss:;`

## 2024-05-22 - Configuration Security Test Inconsistency
**Vulnerability:** Potential configuration drift or false positive test failures.
**Learning:** `TestConfigurationSecurity` in `tests/security_test.go` fails when `settings.cfg` is auto-generated because the default generation logic does not include the environment variable placeholders that the test expects (unlike `settings.cfg.template`).
**Prevention:** Ensure `settings.cfg` is initialized from `settings.cfg.template` in CI/CD pipelines rather than relying on auto-generation.
