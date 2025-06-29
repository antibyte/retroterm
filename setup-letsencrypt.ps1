# TinyOS Let's Encrypt Setup Script
# Run this script to configure Let's Encrypt SSL/TLS certificates

Write-Host "TinyOS Let's Encrypt Setup" -ForegroundColor Green
Write-Host "============================" -ForegroundColor Green

# Prompt for domain
Write-Host ""
$domain = Read-Host "Enter your domain name (e.g., yourdomain.com)"
if ([string]::IsNullOrWhiteSpace($domain)) {
    Write-Host "Domain is required. Exiting." -ForegroundColor Red
    exit 1
}

# Prompt for email
Write-Host ""
$email = Read-Host "Enter your email address for Let's Encrypt registration"
if ([string]::IsNullOrWhiteSpace($email)) {
    Write-Host "Email is required. Exiting." -ForegroundColor Red
    exit 1
}

# Ask about HTTPS redirect
Write-Host ""
$redirectChoice = Read-Host "Force HTTPS redirect (redirect all HTTP traffic to HTTPS)? (y/N)"
$forceRedirect = $redirectChoice -eq "y" -or $redirectChoice -eq "Y"

# Ask about ports
Write-Host ""
Write-Host "Port Configuration:" -ForegroundColor Yellow
Write-Host "For Let's Encrypt to work, you need:"
Write-Host "- Port 80 (HTTP) accessible for domain validation"
Write-Host "- Port 443 (HTTPS) accessible for secure connections"
Write-Host ""

$useStandardPorts = Read-Host "Use standard ports (80/443)? (Y/n)"
if ($useStandardPorts -eq "n" -or $useStandardPorts -eq "N") {
    $httpPort = Read-Host "Enter HTTP port (default: 8080)"
    $httpsPort = Read-Host "Enter HTTPS port (default: 8443)"
    if ([string]::IsNullOrWhiteSpace($httpPort)) { $httpPort = "8080" }
    if ([string]::IsNullOrWhiteSpace($httpsPort)) { $httpsPort = "8443" }
} else {
    $httpPort = "80"
    $httpsPort = "443"
}

# Create certificate directory
$certDir = "./certs"
if (!(Test-Path $certDir)) {
    New-Item -ItemType Directory -Path $certDir -Force | Out-Null
    Write-Host "Created certificate directory: $certDir" -ForegroundColor Green
}

# Update settings.cfg
Write-Host ""
Write-Host "Updating settings.cfg..." -ForegroundColor Yellow

$settingsPath = "./settings.cfg"
if (!(Test-Path $settingsPath)) {
    Write-Host "settings.cfg not found. Please run this script from the TinyOS directory." -ForegroundColor Red
    exit 1
}

# Read current settings
$settings = Get-Content $settingsPath

# Create new TLS section
$tlsSection = @"

[TLS]
; TLS/HTTPS Configuration - configured by setup script
enable_tls = true
; Let's Encrypt automatic certificate management
enable_letsencrypt = true
; Domain name for Let's Encrypt
domain = $domain
; Email for Let's Encrypt registration
letsencrypt_email = $email
; Certificate cache directory
cert_cache_dir = $certDir
; Force HTTPS redirect
force_https_redirect = $($forceRedirect.ToString().ToLower())
; TLS certificate file path (not used with Let's Encrypt)
cert_file = ./certs/server.crt
; TLS key file path (not used with Let's Encrypt)
key_file = ./certs/server.key
; HTTP port (used for Let's Encrypt challenge and optional HTTP redirect)
http_port = $httpPort
; HTTPS port (used when TLS is enabled)
https_port = $httpsPort
"@

# Check if TLS section already exists
$tlsSectionExists = $settings | Where-Object { $_ -match "^\[TLS\]" }

if ($tlsSectionExists) {
    Write-Host "TLS section already exists in settings.cfg" -ForegroundColor Yellow
    $overwrite = Read-Host "Overwrite existing TLS configuration? (y/N)"
    if ($overwrite -ne "y" -and $overwrite -ne "Y") {
        Write-Host "Configuration not updated." -ForegroundColor Yellow
        exit 0
    }
    
    # Remove existing TLS section
    $newSettings = @()
    $inTlsSection = $false
    foreach ($line in $settings) {
        if ($line -match "^\[TLS\]") {
            $inTlsSection = $true
            continue
        }
        if ($line -match "^\[.*\]" -and $inTlsSection) {
            $inTlsSection = $false
        }
        if (!$inTlsSection) {
            $newSettings += $line
        }
    }
    $settings = $newSettings
}

# Add TLS section
$settings += $tlsSection
$settings | Set-Content $settingsPath

Write-Host "✓ Settings updated successfully!" -ForegroundColor Green

# Display summary
Write-Host ""
Write-Host "Configuration Summary:" -ForegroundColor Cyan
Write-Host "Domain: $domain" -ForegroundColor White
Write-Host "Email: $email" -ForegroundColor White
Write-Host "HTTP Port: $httpPort" -ForegroundColor White
Write-Host "HTTPS Port: $httpsPort" -ForegroundColor White
Write-Host "Force HTTPS Redirect: $forceRedirect" -ForegroundColor White
Write-Host "Certificate Directory: $certDir" -ForegroundColor White

Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Yellow
Write-Host "1. Ensure your domain points to this server's IP address" -ForegroundColor White
Write-Host "2. Ensure ports $httpPort and $httpsPort are accessible from the internet" -ForegroundColor White
Write-Host "3. Restart TinyOS to apply the new configuration" -ForegroundColor White
Write-Host "4. Let's Encrypt will automatically obtain and manage certificates" -ForegroundColor White

if ($httpPort -eq "80" -and $httpsPort -eq "443") {
    Write-Host ""
    Write-Host "Important:" -ForegroundColor Red
    Write-Host "You are using standard ports 80 and 443." -ForegroundColor White
    Write-Host "You may need to run TinyOS with administrator privileges." -ForegroundColor White
}

Write-Host ""
Write-Host "Let's Encrypt setup complete!" -ForegroundColor Green
