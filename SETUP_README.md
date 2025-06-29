# TinyOS Setup Scripts

This directory contains setup scripts to configure your TinyOS environment.

## Environment Setup

### Windows (PowerShell)
```powershell
.\setup-env.ps1
```

### Linux/macOS (Bash)
```bash
chmod +x setup-env.sh
./setup-env.sh
```

Both scripts will:
- Generate a secure JWT secret key
- Set environment variables for the application
- Provide instructions for setting the DeepSeek API key

## SSL/TLS Certificate Setup (Let's Encrypt)

### Windows (PowerShell)
```powershell
.\setup-letsencrypt.ps1
```

### Linux/macOS (Bash)
```bash
chmod +x setup-letsencrypt.sh
./setup-letsencrypt.sh
```

The Let's Encrypt scripts will:
- Install SSL/TLS certificates from Let's Encrypt
- Configure automatic HTTPS redirect
- Set up certificate renewal
- Update your `settings.cfg` automatically

## Prerequisites

### Windows
- PowerShell 5.1 or newer
- For Let's Encrypt: Administrative privileges

### Linux/macOS
- Bash shell
- OpenSSL (for secure key generation)
- For Let's Encrypt: certbot installed and sudo access

## Security Notes

- Environment variables are stored in user profile
- JWT secrets are automatically generated with cryptographically secure randomness
- Certificate private keys are protected with appropriate file permissions
- All scripts follow security best practices

## After Setup

1. Restart your terminal/development environment
2. The application will automatically use the new environment variables
3. If using HTTPS, access your site via the configured HTTPS port
4. Set up automatic certificate renewal if using Let's Encrypt

## Troubleshooting

### Environment Variables Not Loading
- **Windows**: Restart PowerShell or your IDE
- **Linux/macOS**: Run `source ~/.bashrc` or restart terminal

### Let's Encrypt Issues
- Ensure your domain points to the server
- Check that ports 80 and 443 are accessible
- Verify no other web server is running on the required ports

### Manual API Key Setup
If you need to set the DeepSeek API key manually:

**Windows:**
```powershell
[Environment]::SetEnvironmentVariable("DEEPSEEK_API_KEY", "your-key-here", "User")
```

**Linux/macOS:**
```bash
echo 'export DEEPSEEK_API_KEY="your-key-here"' >> ~/.bashrc
source ~/.bashrc
```
