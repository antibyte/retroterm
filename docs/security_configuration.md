# Security Configuration Guide

This document explains the security configuration options for JWT tokens and DeepSeek AI integration.

## JWT Configuration

JWT (JSON Web Token) settings are now configurable via the `settings.cfg` file under the `[JWT]` section.

### Configuration Options

```ini
[JWT]
; JWT secret key for token signing - loaded from environment variable JWT_SECRET_KEY
; If environment variable is not set, fallback value will be used (NOT SECURE!)
secret_key = ENVIRONMENT_VARIABLE_NOT_SET_FALLBACK
; Token expiration in hours
token_expiration_hours = 24
```

### Security Best Practices

1. **Change the secret key**: The default secret key should be changed in production environments
2. **Use a strong secret**: Use a long, random string with mixed characters
3. **Keep it secret**: Never commit the actual production secret to version control
4. **Rotate regularly**: Consider rotating the JWT secret periodically

### Environment Variables

You can also set the JWT secret via environment variable:
```bash
export JWT_SECRET_KEY="your-production-secret-here"
```

## DeepSeek AI Configuration

DeepSeek AI API settings are configured under the `[DeepSeek]` section.

### Configuration Options

```ini
[DeepSeek]
; DeepSeek AI API configuration - loaded from environment variable DEEPSEEK_API_KEY
; SECURITY: Never commit real API keys to version control!
api_key = ENVIRONMENT_VARIABLE_NOT_SET
; Use environment variable DEEPSEEK_API_KEY instead of hardcoding
use_environment_variable = true
```

### Environment Variable (Recommended)

For security reasons, it's recommended to use environment variables for API keys:

```bash
export DEEPSEEK_API_KEY="your-actual-api-key-here"
```

When `use_environment_variable = true`, the system will:
1. First check for the `DEEPSEEK_API_KEY` environment variable
2. Fall back to the configured `api_key` in settings.cfg if no environment variable is found

### Security Considerations

1. **Never commit real API keys**: Use placeholder values in version control
2. **Use environment variables**: Preferred method for production deployments
3. **Restrict access**: Ensure only authorized personnel have access to API keys
4. **Monitor usage**: Keep track of API key usage and costs

## TLS/HTTPS Configuration

TinyOS supports automatic HTTPS/TLS certificate management through Let's Encrypt integration.

### Configuration Options

```ini
[TLS]
; Enable TLS/HTTPS support
enable_tls = true
; Enable automatic Let's Encrypt certificate management
enable_letsencrypt = true
; Your domain name (required for Let's Encrypt)
domain = your-domain.com
; Email for Let's Encrypt registration (required)
letsencrypt_email = admin@your-domain.com
; Certificate cache directory
cert_cache_dir = ./certs
; Force HTTPS redirect (redirect all HTTP traffic to HTTPS)
force_https_redirect = true
; HTTP port (for Let's Encrypt challenges and redirects)
http_port = 80
; HTTPS port
https_port = 443
```

### Let's Encrypt Setup

1. **Domain Configuration**: Point your domain to the server's IP address
2. **Port Configuration**: Ensure ports 80 (HTTP) and 443 (HTTPS) are accessible
3. **Email Setup**: Provide a valid email for Let's Encrypt registration
4. **Enable TLS**: Set `enable_tls = true` and `enable_letsencrypt = true`

### Manual Certificate Setup

For manual certificate management (not using Let's Encrypt):

```ini
[TLS]
enable_tls = true
enable_letsencrypt = false
cert_file = ./certs/server.crt
key_file = ./certs/server.key
```

### Security Considerations

1. **Certificate Renewal**: Let's Encrypt certificates auto-renew every 90 days
2. **Certificate Storage**: Certificates are cached in the specified directory
3. **Port Security**: Ensure firewall allows ports 80 and 443
4. **Domain Validation**: Let's Encrypt validates domain ownership via HTTP challenge

### Production Deployment

For production deployment with Let's Encrypt:

```ini
[TLS]
enable_tls = true
enable_letsencrypt = true
domain = yourdomain.com
letsencrypt_email = admin@yourdomain.com
cert_cache_dir = /etc/tinyos/certs
force_https_redirect = true
http_port = 80
https_port = 443
```

**Important**: Run the server with appropriate permissions to bind to ports 80 and 443.

## Implementation Details

### JWT Token Flow

1. **Session Creation**: Guest sessions generate a unique session ID
2. **Token Generation**: Session ID is embedded in a signed JWT token
3. **Token Validation**: Incoming requests validate the JWT signature and expiration
4. **Cookie Storage**: Tokens are stored as HTTP-only cookies for security

### Configuration Loading

The system loads configuration at startup:
- JWT secret and expiration from `[JWT]` section
- DeepSeek API key from `[DeepSeek]` section or environment variable
- Fallback values are provided for development environments

### Code Changes

Key files updated for configuration support:
- `pkg/auth/jwt.go`: JWT token generation and validation
- `pkg/auth/handlers.go`: HTTP endpoints for authentication
- `pkg/tinyos/deepseek.go`: DeepSeek AI integration
- `settings.cfg`: Configuration file with new sections

## Testing

All authentication tests continue to pass with the new configuration system:
```bash
go test ./pkg/auth -v
```

The test suite includes:
- JWT token generation and validation
- HTTP endpoint testing
- Cookie handling
- Error scenarios
- Performance benchmarks

## Migration Guide

### From Hardcoded Values

If upgrading from a version with hardcoded values:

1. **Update settings.cfg**: Add the new `[JWT]` and `[DeepSeek]` sections
2. **Set environment variables**: Configure `DEEPSEEK_API_KEY` in your environment
3. **Update deployment scripts**: Include environment variable configuration
4. **Test thoroughly**: Verify authentication still works correctly

### Production Deployment

For production environments:

1. **Generate new JWT secret**: Use a cryptographically secure random string
2. **Configure API keys**: Set up environment variables for sensitive keys
3. **Secure configuration**: Protect access to settings.cfg and environment
4. **Monitor logs**: Watch for authentication errors or API failures

## Example Production Configuration

```ini
[JWT]
secret_key = CHANGE_THIS_IN_PRODUCTION_TO_RANDOM_STRING_256_BITS_LONG
token_expiration_hours = 24

[DeepSeek]
api_key = PLACEHOLDER_USE_ENVIRONMENT_VARIABLE
use_environment_variable = true
```

With environment variables:
```bash
export JWT_SECRET_KEY="$(openssl rand -base64 32)"
export DEEPSEEK_API_KEY="sk-your-real-api-key-here"
```

This configuration provides better security, flexibility, and maintainability for the TinyOS authentication system.
