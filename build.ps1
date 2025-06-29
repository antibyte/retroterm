# Build Script for Windows (PowerShell)
# Creates a production-ready distribution of the TinyOS web application

param(
    [switch]$Clean = $false,
    [switch]$SkipMinify = $false
)

# Configuration
$DIST_DIR = "dist"
$JS_DIR = "js"
$CSS_DIR = "css"
$ASSETS_DIR = "assets"
$CHESS_GFX_DIR = "chess_gfx"
$EXAMPLES_DIR = "examples"

# Colors for output
$Green = "Green"
$Yellow = "Yellow"
$Red = "Red"
$Cyan = "Cyan"

Write-Host "=== TinyOS Build Script for Windows ===" -ForegroundColor $Cyan
Write-Host "Building production distribution..." -ForegroundColor $Green

# Check if Node.js and npm are available
try {
    $nodeVersion = node --version 2>$null
    $npmVersion = npm --version 2>$null
    Write-Host "Node.js version: $nodeVersion" -ForegroundColor $Green
    Write-Host "npm version: $npmVersion" -ForegroundColor $Green
} catch {
    Write-Host "ERROR: Node.js and npm are required but not found!" -ForegroundColor $Red
    Write-Host "Please install Node.js from https://nodejs.org/" -ForegroundColor $Yellow
    exit 1
}

# Check if terser is installed globally
try {
    $terserVersion = terser --version 2>$null
    Write-Host "Terser version: $terserVersion" -ForegroundColor $Green
} catch {
    Write-Host "Terser not found. Installing globally..." -ForegroundColor $Yellow
    npm install -g terser
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Failed to install terser!" -ForegroundColor $Red
        Write-Host "You can install it manually with: npm install -g terser" -ForegroundColor $Yellow
        exit 1
    }
}

# Clean dist directory if requested or if it exists
if ($Clean -or (Test-Path $DIST_DIR)) {
    Write-Host "Cleaning distribution directory..." -ForegroundColor $Yellow
    if (Test-Path $DIST_DIR) {
        Remove-Item -Recurse -Force $DIST_DIR
    }
}

# Create dist directory structure
Write-Host "Creating distribution directory structure..." -ForegroundColor $Green
New-Item -ItemType Directory -Force -Path $DIST_DIR | Out-Null
New-Item -ItemType Directory -Force -Path "$DIST_DIR\js" | Out-Null
New-Item -ItemType Directory -Force -Path "$DIST_DIR\css" | Out-Null
New-Item -ItemType Directory -Force -Path "$DIST_DIR\assets" | Out-Null
New-Item -ItemType Directory -Force -Path "$DIST_DIR\chess_gfx" | Out-Null
New-Item -ItemType Directory -Force -Path "$DIST_DIR\examples" | Out-Null

# List of JavaScript files in load order (from HTML)
$jsFiles = @(
    "config.js",
    "dynamicViewport.js", 
    "samjs.min.js",
    "samInit.js",
    "three.min.js",
    "retrographics.js",
    "spriteManager.js", 
    "vectorManager.js",
    "retrosound.js",
    "sidPlayer.js",
    "ansiParser.js",
    "auth.js",
    "retroconsole.js",
    "retroterminal.js"
)

# Files to copy without bundling (already minified or special handling)
$copyJsFiles = @(
    "samjs.min.js",
    "three.min.js",
    "jsSID.js"
)

# Copy already minified files
Write-Host "Copying pre-minified JavaScript files..." -ForegroundColor $Green
foreach ($file in $copyJsFiles) {
    if (Test-Path "$JS_DIR\$file") {
        Copy-Item "$JS_DIR\$file" "$DIST_DIR\js\$file"
        Write-Host "  Copied: $file" -ForegroundColor $Green
    } else {
        Write-Host "  Warning: $file not found" -ForegroundColor $Yellow
    }
}

# Bundle and minify custom JavaScript files
$bundleFiles = $jsFiles | Where-Object { $copyJsFiles -notcontains $_ }

if (-not $SkipMinify) {
    Write-Host "Bundling and minifying custom JavaScript files..." -ForegroundColor $Green
    
    # Create a temporary bundle file
    $tempBundle = "temp_bundle.js"
    $bundleContent = ""
    
    foreach ($file in $bundleFiles) {
        if (Test-Path "$JS_DIR\$file") {
            Write-Host "  Adding to bundle: $file" -ForegroundColor $Green
            $content = Get-Content "$JS_DIR\$file" -Raw
            # Add file separator comment
            $bundleContent += "`n`n/* === $file === */`n"
            $bundleContent += $content
        } else {
            Write-Host "  Warning: $file not found" -ForegroundColor $Yellow
        }
    }
    
    # Write temporary bundle
    $bundleContent | Out-File -FilePath $tempBundle -Encoding UTF8
    
    # Minify the bundle
    Write-Host "Minifying bundle..." -ForegroundColor $Green
    terser $tempBundle --compress --mangle --output "$DIST_DIR\js\app.min.js"
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  Bundle created: js/app.min.js" -ForegroundColor $Green
    } else {
        Write-Host "  ERROR: Minification failed!" -ForegroundColor $Red
        # Fall back to copying unminified
        Copy-Item $tempBundle "$DIST_DIR\js\app.js"
        Write-Host "  Fallback: Created unminified bundle" -ForegroundColor $Yellow
    }
    
    # Clean up temp file
    Remove-Item $tempBundle -ErrorAction SilentlyContinue
} else {
    Write-Host "Skipping minification, copying files individually..." -ForegroundColor $Yellow
    foreach ($file in $bundleFiles) {
        if (Test-Path "$JS_DIR\$file") {
            Copy-Item "$JS_DIR\$file" "$DIST_DIR\js\$file"
            Write-Host "  Copied: $file" -ForegroundColor $Green
        }
    }
}

# Copy and minify CSS
Write-Host "Processing CSS files..." -ForegroundColor $Green
if (Test-Path "$CSS_DIR\retroterminal.css") {
    try {
        # Try to minify CSS with terser (it can handle CSS too)
        terser "$CSS_DIR\retroterminal.css" --compress --output "$DIST_DIR\css\retroterminal.min.css" 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  Minified: retroterminal.css -> retroterminal.min.css" -ForegroundColor $Green
        } else {
            throw "Terser CSS minification failed"
        }
    } catch {
        # Fallback: just copy the CSS file
        Copy-Item "$CSS_DIR\retroterminal.css" "$DIST_DIR\css\retroterminal.css"
        Write-Host "  Copied (unminified): retroterminal.css" -ForegroundColor $Yellow
    }
} else {
    Write-Host "  Warning: retroterminal.css not found" -ForegroundColor $Yellow
}

# Copy assets
Write-Host "Copying assets..." -ForegroundColor $Green
if (Test-Path $ASSETS_DIR) {
    Copy-Item "$ASSETS_DIR\*" "$DIST_DIR\assets\" -Recurse -Force
    Write-Host "  Copied assets directory" -ForegroundColor $Green
}

if (Test-Path $CHESS_GFX_DIR) {
    Copy-Item "$CHESS_GFX_DIR\*" "$DIST_DIR\chess_gfx\" -Recurse -Force
    Write-Host "  Copied chess_gfx directory" -ForegroundColor $Green
}

if (Test-Path $EXAMPLES_DIR) {
    Copy-Item "$EXAMPLES_DIR\*" "$DIST_DIR\examples\" -Recurse -Force
    Write-Host "  Copied examples directory" -ForegroundColor $Green
}

# Create production HTML file
Write-Host "Creating production HTML file..." -ForegroundColor $Green
$htmlContent = Get-Content "retroterminal.html" -Raw

# Replace script includes with bundled version
if (-not $SkipMinify) {
    $scriptPattern = '(?s)<!-- Build: Start Scripts -->.*?<!-- Build: End Scripts -->'
    $newScripts = @"
    <!-- Production JavaScript Bundle -->
    <script src="js/samjs.min.js"></script>
    <script src="js/three.min.js" id="threejs"></script>
    <script src="js/jsSID.js"></script>
    <script src="js/app.min.js"></script>
"@
    
    # If the build markers don't exist, replace the script section manually
    if ($htmlContent -notmatch $scriptPattern) {
        # Replace individual script tags with bundle
        $htmlContent = $htmlContent -replace '<script src="js/config\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/dynamicViewport\.js"></script>[^\r\n]*', ''
        $htmlContent = $htmlContent -replace '<script src="js/samInit\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script type="module" src="js/retrographics\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script type="module" src="js/spriteManager\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script type="module" src="js/vectorManager\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/retrosound\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/sidPlayer\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/ansiParser\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/auth\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script src="js/retroconsole\.js"></script>', ''
        $htmlContent = $htmlContent -replace '<script type="module" src="js/retroterminal\.js"></script>', $newScripts
    } else {
        $htmlContent = $htmlContent -replace $scriptPattern, $newScripts
    }
}

# Update CSS reference to minified version if it exists
if (Test-Path "$DIST_DIR\css\retroterminal.min.css") {
    $htmlContent = $htmlContent -replace 'href="retroterminal\.css"', 'href="css/retroterminal.min.css"'
} else {
    $htmlContent = $htmlContent -replace 'href="retroterminal\.css"', 'href="css/retroterminal.css"'
}

# Write production HTML
$htmlContent | Out-File -FilePath "$DIST_DIR\index.html" -Encoding UTF8
Write-Host "  Created: index.html" -ForegroundColor $Green

# Copy additional files that might be needed
$additionalFiles = @("SECURITY_SETUP.md", "settings.cfg.template")
foreach ($file in $additionalFiles) {
    if (Test-Path $file) {
        Copy-Item $file "$DIST_DIR\$file"
        Write-Host "  Copied: $file" -ForegroundColor $Green
    }
}

# Calculate sizes
$originalSize = 0
$distSize = 0

Get-ChildItem -Path $JS_DIR -Filter "*.js" | ForEach-Object {
    $originalSize += $_.Length
}

Get-ChildItem -Path "$DIST_DIR\js" -Filter "*.js" | ForEach-Object {
    $distSize += $_.Length
}

$compressionRatio = if ($originalSize -gt 0) { [math]::Round((1 - $distSize / $originalSize) * 100, 1) } else { 0 }

Write-Host "`n=== Build Complete ===" -ForegroundColor $Cyan
Write-Host "Distribution created in: $DIST_DIR" -ForegroundColor $Green
Write-Host "Original JS size: $([math]::Round($originalSize / 1KB, 1)) KB" -ForegroundColor $Yellow
Write-Host "Minified JS size: $([math]::Round($distSize / 1KB, 1)) KB" -ForegroundColor $Yellow
Write-Host "Compression ratio: $compressionRatio%" -ForegroundColor $Green
Write-Host "`nTo deploy, copy the contents of the '$DIST_DIR' directory to your web server." -ForegroundColor $Cyan
