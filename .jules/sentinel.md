## 2026-01-23 - Default Admin Backdoor
**Vulnerability:** Found hardcoded credentials `dyson:daniel` in `pkg/tinyos/db.go` that were created if the user didn't exist.
**Learning:** Initial setup routines can hide backdoors or insecure defaults that persist if not changed. Hardcoding passwords in code is always risky, even if hashed, because the source reveals the plaintext.
**Prevention:** Always generate random passwords for default accounts and log them once, or force the user to set a password on first launch.
