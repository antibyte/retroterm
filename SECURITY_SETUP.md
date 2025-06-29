# Security Setup Guide

## ⚠️ IMPORTANT SECURITY NOTICE

This project has been secured to prevent hardcoded secrets in the codebase. Follow these steps to set up your environment securely.

## Quick Setup

1. **Run the setup script** (PowerShell):
   ```powershell
   .\setup-env.ps1
   ```

2. **Copy the configuration template**:
   ```powershell
   Copy-Item settings.cfg.template settings.cfg
   ```

3. **Restart your development environment** to load the new environment variables.

## Manual Setup

If you prefer to set up manually:

### 1. Set Environment Variables

**JWT Secret** (generate a secure random key):
```powershell
$jwtSecret = [System.Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))
[Environment]::SetEnvironmentVariable("JWT_SECRET_KEY", $jwtSecret, "User")
```

**DeepSeek API Key**:
```powershell
[Environment]::SetEnvironmentVariable("DEEPSEEK_API_KEY", "your-actual-api-key-here", "User")
```

### 2. Configuration Priority

The application loads secrets in this order:
1. **Environment variables** (recommended for production)
2. **Configuration file** (development only)
3. **Fallback values** (not secure - will show warnings)

## Security Best Practices

✅ **DO:**
- Use environment variables for all secrets
- Keep `settings.cfg` in `.gitignore`
- Rotate secrets regularly
- Use different secrets for different environments

❌ **DON'T:**
- Commit `settings.cfg` to version control
- Share secrets in plain text
- Use default/fallback secrets in production
- Hardcode secrets in source code

## Files Overview

- `settings.cfg.template` - Template with placeholder values
- `settings.cfg` - Your actual configuration (git-ignored)
- `setup-env.ps1` - Automated setup script
- `.gitignore` - Excludes sensitive files from git

## Troubleshooting

**"Using fallback JWT secret" warning:**
- Set the `JWT_SECRET_KEY` environment variable
- Restart your application

**"DeepSeek API key not configured" error:**
- Set the `DEEPSEEK_API_KEY` environment variable
- Ensure your API key is valid

**Environment variables not loading:**
- Restart your terminal/IDE after setting variables
- Check variable scope (User vs System vs Process)
