# Development Setup Script for TinyOS Retro Terminal
# This script sets up the development environment with full logging

Write-Host "=== TinyOS Retro Terminal - Development Setup ===" -ForegroundColor Green

# Check if development template exists
if (-not (Test-Path "settings.cfg.template")) {
    Write-Host "ERROR: settings.cfg.template not found!" -ForegroundColor Red
    exit 1
}

# Backup current settings if they exist
if (Test-Path "settings.cfg") {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    Copy-Item "settings.cfg" "settings.cfg.backup_$timestamp"
    Write-Host "Current settings backed up to settings.cfg.backup_$timestamp" -ForegroundColor Yellow
}

# Copy development settings
Copy-Item "settings.cfg.template" "settings.cfg"
Write-Host "Development settings activated" -ForegroundColor Green

# Enable full debug logging in development
$content = Get-Content "settings.cfg" -Raw
$content = $content -replace "enable_debug_logging = false", "enable_debug_logging = true"
$content = $content -replace "log_level = ERROR", "log_level = DEBUG"
$content = $content -replace "disable_legacy_logging = true", "disable_legacy_logging = false"
$content = $content -replace "log_websocket = false", "log_websocket = true"
$content = $content -replace "log_terminal = false", "log_terminal = true"
$content = $content -replace "log_chat = false", "log_chat = true"
$content = $content -replace "log_editor = false", "log_editor = true"
$content = $content -replace "log_filesystem = false", "log_filesystem = true"
$content = $content -replace "log_tinybasic = false", "log_tinybasic = true"
$content = $content -replace "log_database = false", "log_database = true"
$content = $content -replace "log_session = false", "log_session = true"
$content = $content -replace "log_config = false", "log_config = true"
$content = $content -replace "log_chess = false", "log_chess = true"
Set-Content "settings.cfg" $content

Write-Host "Development logging enabled" -ForegroundColor Green

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
Write-Host "`nDevelopment Logging Configuration:" -ForegroundColor Cyan
Write-Host "  Debug Logging: ENABLED" -ForegroundColor Green
Write-Host "  Legacy Logging: ENABLED" -ForegroundColor Yellow
Write-Host "  Log Level: DEBUG" -ForegroundColor Green
Write-Host "  All Log Areas: ENABLED" -ForegroundColor Green

Write-Host "`n=== Development Setup Complete ===" -ForegroundColor Green
Write-Host "The server is now configured for development with full logging." -ForegroundColor White
Write-Host "To start the server: .\main.exe" -ForegroundColor Yellow
