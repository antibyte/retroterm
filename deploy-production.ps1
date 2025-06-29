# Production Deployment Script for TinyOS Retro Terminal
# This script sets up the production environment with minimal logging

Write-Host "=== TinyOS Retro Terminal - Production Deployment ===" -ForegroundColor Green

# Check if production config exists
if (-not (Test-Path "settings.production.cfg")) {
    Write-Host "ERROR: settings.production.cfg not found!" -ForegroundColor Red
    exit 1
}

# Backup current settings if they exist
if (Test-Path "settings.cfg") {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    Copy-Item "settings.cfg" "settings.cfg.backup_$timestamp"
    Write-Host "Current settings backed up to settings.cfg.backup_$timestamp" -ForegroundColor Yellow
}

# Copy production settings
Copy-Item "settings.production.cfg" "settings.cfg"
Write-Host "Production settings activated" -ForegroundColor Green

# Check environment variables
Write-Host "`nChecking environment variables..." -ForegroundColor Cyan

$envVars = @("JWT_SECRET_KEY", "DEEPSEEK_API_KEY")
$missingVars = @()

foreach ($var in $envVars) {
    if ([string]::IsNullOrEmpty([Environment]::GetEnvironmentVariable($var))) {
        $missingVars += $var
        Write-Host "WARNING: $var not set" -ForegroundColor Yellow
    } else {
        Write-Host "$var is set" -ForegroundColor Green
    }
}

if ($missingVars.Count -gt 0) {
    Write-Host "`nIMPORTANT: The following environment variables are missing:" -ForegroundColor Red
    foreach ($var in $missingVars) {
        Write-Host "  - $var" -ForegroundColor Red
    }
    Write-Host "Set them using:" -ForegroundColor Yellow
    Write-Host '  $env:JWT_SECRET_KEY = "your-secret-key-here"' -ForegroundColor White
    Write-Host '  $env:DEEPSEEK_API_KEY = "your-api-key-here"' -ForegroundColor White
}

# Build the application
Write-Host "`nBuilding application..." -ForegroundColor Cyan
try {
    & go build -v .
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build successful!" -ForegroundColor Green
    } else {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "Build error: $_" -ForegroundColor Red
    exit 1
}

# Show current logging configuration
Write-Host "`nProduction Logging Configuration:" -ForegroundColor Cyan
Write-Host "  Debug Logging: DISABLED" -ForegroundColor Green
Write-Host "  Legacy Logging: DISABLED" -ForegroundColor Green
Write-Host "  Log Level: ERROR only" -ForegroundColor Green
Write-Host "  Active Log Areas:" -ForegroundColor Yellow
Write-Host "    - Authentication: ENABLED" -ForegroundColor White
Write-Host "    - Security: ENABLED" -ForegroundColor White
Write-Host "    - Ban System: ENABLED" -ForegroundColor White
Write-Host "    - All others: DISABLED" -ForegroundColor Gray

Write-Host "`n=== Production Setup Complete ===" -ForegroundColor Green
Write-Host "The server is now configured for production with minimal logging." -ForegroundColor White
Write-Host "To start the server: .\main.exe" -ForegroundColor Yellow
