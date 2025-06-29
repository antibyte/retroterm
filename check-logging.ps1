# Logging Status Verification Script
# Shows current logging configuration

Write-Host "=== TinyOS Retro Terminal - Logging Status ===" -ForegroundColor Cyan

if (-not (Test-Path "settings.cfg")) {
    Write-Host "ERROR: settings.cfg not found!" -ForegroundColor Red
    exit 1
}

$config = Get-Content "settings.cfg"

# Extract debug settings
$debugEnabled = ($config | Select-String "enable_debug_logging = (.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
$legacyDisabled = ($config | Select-String "disable_legacy_logging = (.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
$logLevel = ($config | Select-String "log_level = (.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })

Write-Host "`nCurrent Configuration:" -ForegroundColor Yellow
Write-Host "  Debug Logging: $debugEnabled" -ForegroundColor $(if ($debugEnabled -eq "true") { "Red" } else { "Green" })
Write-Host "  Legacy Logging Disabled: $legacyDisabled" -ForegroundColor $(if ($legacyDisabled -eq "true") { "Green" } else { "Red" })
Write-Host "  Log Level: $logLevel" -ForegroundColor White

Write-Host "`nLogging Areas:" -ForegroundColor Yellow

$logAreas = @(
    "log_websocket",
    "log_terminal", 
    "log_auth",
    "log_chat",
    "log_editor",
    "log_filesystem",
    "log_resources",
    "log_security",
    "log_bansystem",
    "log_tinybasic",
    "log_database",
    "log_session",
    "log_config",
    "log_chess"
)

foreach ($area in $logAreas) {
    $value = ($config | Select-String "$area = (.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
    if ($value) {
        $color = if ($value -eq "true") { "Yellow" } else { "Gray" }
        $status = if ($value -eq "true") { "ENABLED" } else { "disabled" }
        Write-Host "  $area : $status" -ForegroundColor $color
    }
}

Write-Host "`nProduction Readiness Check:" -ForegroundColor Cyan
$isProd = $true

if ($debugEnabled -eq "true") {
    Write-Host "  [X] Debug logging is ENABLED (should be false for production)" -ForegroundColor Red
    $isProd = $false
}

if ($legacyDisabled -ne "true") {
    Write-Host "  [X] Legacy logging is ENABLED (should be disabled for production)" -ForegroundColor Red
    $isProd = $false
}

if ($logLevel -ne "ERROR") {
    Write-Host "  [X] Log level is $logLevel (should be ERROR for production)" -ForegroundColor Red
    $isProd = $false
}

# Count enabled log areas (only critical ones should be enabled)
$criticalAreas = @("log_auth", "log_security", "log_bansystem")
$enabledAreas = @()
$nonCriticalEnabled = @()

foreach ($area in $logAreas) {
    $value = ($config | Select-String "$area = (.+)" | ForEach-Object { $_.Matches[0].Groups[1].Value })
    if ($value -eq "true") {
        $enabledAreas += $area
        if ($area -notin $criticalAreas) {
            $nonCriticalEnabled += $area
        }
    }
}

if ($nonCriticalEnabled.Count -gt 0) {
    Write-Host "  [!] Non-critical logging areas enabled: $($nonCriticalEnabled -join ', ')" -ForegroundColor Yellow
}

if ($isProd -and $nonCriticalEnabled.Count -eq 0) {
    Write-Host "  [OK] Configuration is PRODUCTION READY" -ForegroundColor Green
    Write-Host "       - Debug logging disabled" -ForegroundColor Green
    Write-Host "       - Legacy logging disabled" -ForegroundColor Green
    Write-Host "       - Only critical areas enabled" -ForegroundColor Green
} elseif ($debugEnabled -eq "true" -and $legacyDisabled -ne "true") {
    Write-Host "  [DEV] Configuration is set for DEVELOPMENT" -ForegroundColor Blue
} else {
    Write-Host "  [!] Configuration needs review" -ForegroundColor Yellow
}

Write-Host ""
