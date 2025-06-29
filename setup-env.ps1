# TinyOS Environment Setup Script - Simple Version

Write-Host "TinyOS Security Setup" -ForegroundColor Green
Write-Host "=====================" -ForegroundColor Green

# Generate secure JWT secret
Write-Host ""
Write-Host "Generating secure JWT secret..." -ForegroundColor Yellow
$jwtSecret = [System.Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(32))

# Set JWT secret
[Environment]::SetEnvironmentVariable("JWT_SECRET_KEY", $jwtSecret, "User")
Write-Host "✓ JWT_SECRET_KEY set: $($jwtSecret.Substring(0,8))..." -ForegroundColor Green

Write-Host ""
Write-Host "Environment variables set successfully!" -ForegroundColor Green
Write-Host "Please restart your development environment to load the new variables." -ForegroundColor Yellow

Write-Host ""
Write-Host "To set DeepSeek API key manually:" -ForegroundColor Cyan
Write-Host '[Environment]::SetEnvironmentVariable("DEEPSEEK_API_KEY", "your-key-here", "User")' -ForegroundColor Gray

Write-Host ""
Write-Host "Security setup complete!" -ForegroundColor Green
