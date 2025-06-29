#!/usr/bin/env node
/**
 * Cross-platform build script for TinyOS Web Application
 * Creates a production-ready distribution with bundled and minified assets
 * Includes backend executables for multiple platforms
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Configuration
const config = {
    distDir: 'dist',
    jsDir: 'js',
    cssDir: 'css',    assetsDir: 'assets',
    chessGfxDir: 'chess_gfx',    examplesDir: 'examples',
    dysonDir: 'dyson',
    promptsDir: 'prompts',
    backendDir: 'backend',
    
    // Custom JS files to bundle (order matters!)
    bundleJsFiles: [
        'config.js',
        'dynamicViewport.js',
        'samInit.js',
        'retrographics.js',
        'spriteManager.js', 
        'vectorManager.js',
        'retrosound.js',
        'sidPlayer.js',
        'ansiParser.js',
        'auth.js',
        'retroconsole.js',
        'retroterminal.js'
    ],
    
    // External JS files to copy as-is
    externalJsFiles: [
        'samjs.min.js',
        'three.min.js',
        'jsSID.js'
    ],
    
    // Backend build targets
    backendTargets: [
        { os: 'windows', arch: 'amd64', ext: '.exe', dir: 'windows-x64' },
        { os: 'linux', arch: 'amd64', ext: '', dir: 'linux-x64' },
        { os: 'linux', arch: 'arm64', ext: '', dir: 'linux-arm64' }
    ],
    
    additionalFiles: [
        'SECURITY_SETUP.md',
        'settings.cfg.template'
    ]
};

// Parse command line arguments
const args = process.argv.slice(2);
const options = {
    clean: args.includes('--clean'),
    skipMinify: args.includes('--skip-minify'),
    help: args.includes('--help') || args.includes('-h')
};

// Colors for console output
const colors = {
    reset: '\x1b[0m',
    red: '\x1b[31m',
    green: '\x1b[32m',
    yellow: '\x1b[33m',
    blue: '\x1b[34m',
    magenta: '\x1b[35m',
    cyan: '\x1b[36m',
    white: '\x1b[37m'
};

function colorLog(message, color = 'white') {
    console.log(`${colors[color]}${message}${colors.reset}`);
}

function showHelp() {
    colorLog('=== TinyOS Build Script ===', 'cyan');
    console.log('');
    console.log('Usage: node build.js [options]');
    console.log('');
    console.log('Options:');
    console.log('  --clean       Clean the distribution directory before building');
    console.log('  --skip-minify Skip minification (useful for debugging)');
    console.log('  --help, -h    Show this help message');
    console.log('');
    console.log('Examples:');
    console.log('  node build.js                # Standard production build');
    console.log('  node build.js --clean        # Clean build');
    console.log('  node build.js --skip-minify  # Development build');
    console.log('');
}

function ensureDir(dirPath) {
    if (!fs.existsSync(dirPath)) {
        fs.mkdirSync(dirPath, { recursive: true });
    }
}

function copyFile(src, dest) {
    try {
        fs.copyFileSync(src, dest);
        return true;
    } catch (err) {
        colorLog(`  Warning: Could not copy ${src} - ${err.message}`, 'yellow');
        return false;
    }
}

function copyDirectory(src, dest) {
    try {
        if (!fs.existsSync(src)) {
            return false;
        }
        ensureDir(dest);
        const items = fs.readdirSync(src);
        for (const item of items) {
            const srcPath = path.join(src, item);
            const destPath = path.join(dest, item);
            const stat = fs.statSync(srcPath);
            if (stat.isDirectory()) {
                copyDirectory(srcPath, destPath);
            } else {
                copyFile(srcPath, destPath);
            }
        }
        return true;
    } catch (err) {
        colorLog(`  Warning: Could not copy directory ${src} - ${err.message}`, 'yellow');
        return false;
    }
}

function getFileSize(filePath) {
    try {
        return fs.statSync(filePath).size;
    } catch {
        return 0;
    }
}

function checkDependencies() {
    try {
        execSync('node --version', { stdio: 'ignore' });
        execSync('npm --version', { stdio: 'ignore' });
    } catch {
        colorLog('ERROR: Node.js and npm are required!', 'red');
        colorLog('Please install Node.js from https://nodejs.org/', 'yellow');
        process.exit(1);
    }

    try {
        execSync('npx terser --version', { stdio: 'ignore' });
    } catch {
        colorLog('Terser not found. Installing...', 'yellow');
        try {
            execSync('npm install terser --save-dev', { stdio: 'inherit' });
        } catch {
            colorLog('Failed to install terser. Please run: npm install terser --save-dev', 'red');
            process.exit(1);
        }
    }
}

function cleanDist() {
    if (fs.existsSync(config.distDir)) {
        colorLog('Cleaning distribution directory...', 'yellow');
        fs.rmSync(config.distDir, { recursive: true, force: true });
    }
}

function createDistStructure() {
    colorLog('Creating distribution directory structure...', 'green');
    ensureDir(config.distDir);
    ensureDir(path.join(config.distDir, 'js'));
    ensureDir(path.join(config.distDir, 'css'));
    ensureDir(path.join(config.distDir, 'assets'));
    ensureDir(path.join(config.distDir, 'chess_gfx'));
    ensureDir(path.join(config.distDir, 'examples'));
}

function copyExternalJSFiles() {
    colorLog('Copying external JavaScript libraries...', 'green');
    for (const file of config.externalJsFiles) {
        const srcPath = path.join(config.jsDir, file);
        const destPath = path.join(config.distDir, 'js', file);
        if (copyFile(srcPath, destPath)) {
            colorLog(`  Copied: ${file}`, 'green');
        }
    }
}

function bundleAndMinifyJS() {
    colorLog('Bundling and minifying custom JavaScript files...', 'green');
      // Create bundle content
    let bundleContent = '';
    let totalSize = 0;
    let configContent = '';
    
    // First, handle config.js separately to ensure it comes first
    const configFile = 'config.js';
    const configFilePath = path.join(config.jsDir, configFile);
    if (fs.existsSync(configFilePath)) {
        colorLog(`  Adding config first: ${configFile}`, 'green');
        configContent = fs.readFileSync(configFilePath, 'utf8');
        totalSize += configContent.length;
    } else {
        colorLog(`  Warning: ${configFile} not found`, 'yellow');
    }
    
    for (const file of config.bundleJsFiles) {
        // Skip config.js since we already handled it
        if (file === 'config.js') continue;
        
        const filePath = path.join(config.jsDir, file);
        if (fs.existsSync(filePath)) {
            colorLog(`  Adding to bundle: ${file}`, 'green');
            let fileContent = fs.readFileSync(filePath, 'utf8');
            
            // Fix variable conflicts
            // More robust regex to remove the CFG constant declaration from subsequent files
            fileContent = fileContent.replace(/^const\s+CFG\s*=\s*window\.CRT_CONFIG;/gm, '// Global CFG already defined');
            
            // Convert ES6 modules to global variables for bundling
            // Convert exports to global assignments
            fileContent = fileContent.replace(/export const (\w+)/g, 'window.$1');
            fileContent = fileContent.replace(/export function (\w+)/g, 'window.$1 = function $1');
            fileContent = fileContent.replace(/export \{([^}]+)\}/g, (match, exports) => {
                const exportList = exports.split(',').map(e => e.trim());
                return exportList.map(exp => `window.${exp} = ${exp};`).join('\n');
            });
            
            // Convert imports to global references (simple pattern)
            fileContent = fileContent.replace(/import \{([^}]+)\} from ['"]([^'"]+)['"];?/g, (match, imports) => {
                const importList = imports.split(',').map(i => i.trim());
                return `// Imports: ${importList.join(', ')} from global scope`;
            });
            
            // Convert const destructuring from window to direct access
            fileContent = fileContent.replace(/const \{([^}]+)\} = window\.(\w+) \|\| \{\};/g, 
                '// Global references: $1');
            
            bundleContent += `\n\n/* === ${file} === */\n`;
            bundleContent += fileContent;
            
            totalSize += fileContent.length;
        } else {
            colorLog(`  Warning: ${file} not found`, 'yellow');
        }
    }
    
    // Combine config first, then global CFG declaration, then the rest
    const finalBundle = configContent + '\n\n// Global configuration accessor\nconst CFG = window.CRT_CONFIG;\n' + bundleContent;
      if (options.skipMinify) {
        // Development mode - unminified bundle
        const bundlePath = path.join(config.distDir, 'js', 'retroterm.js');
        fs.writeFileSync(bundlePath, finalBundle);
        colorLog(`  Created development bundle: retroterm.js (${(totalSize / 1024).toFixed(1)} KB)`, 'green');
        return { original: totalSize, minified: totalSize };
    }
      // Write temporary bundle
    const tempBundle = 'temp_bundle.js';
    fs.writeFileSync(tempBundle, finalBundle);
    
    // Minify the bundle
    colorLog('Minifying bundle...', 'green');
    try {
        const terserCmd = `npx terser ${tempBundle} --compress --mangle --output ${path.join(config.distDir, 'js', 'retroterm.min.js')}`;
        execSync(terserCmd, { stdio: 'pipe' });
        
        const minifiedSize = fs.statSync(path.join(config.distDir, 'js', 'retroterm.min.js')).size;
        colorLog(`  Bundle created: retroterm.min.js (${(minifiedSize / 1024).toFixed(1)} KB)`, 'green');
        
        // Clean up temp file
        fs.unlinkSync(tempBundle);
        
        return { original: totalSize, minified: minifiedSize };
    } catch (err) {
        colorLog('  ERROR: Minification failed!', 'red');
        colorLog('  Fallback: Creating unminified bundle', 'yellow');
        const bundlePath = path.join(config.distDir, 'js', 'retroterm.js');
        copyFile(tempBundle, bundlePath);
        
        // Clean up temp file
        try { fs.unlinkSync(tempBundle); } catch {}
        
        return { original: totalSize, minified: totalSize };
    }
}

function buildBackendExecutables() {
    colorLog('Building backend executables...', 'green');
    
    // Create backend directory structure
    const backendDistDir = path.join(config.distDir, config.backendDir);
    ensureDir(backendDistDir);
    
    let buildErrors = [];
    
    for (const target of config.backendTargets) {
        const targetDir = path.join(backendDistDir, target.dir);
        ensureDir(targetDir);
        
        const outputFile = path.join(targetDir, `retroterm${target.ext}`);
        const buildCmd = `go build -ldflags="-s -w" -o "${outputFile}" .`;
        
        colorLog(`  Building ${target.os}/${target.arch}...`, 'green');
        
        try {
            // Set environment variables for cross-compilation
            const env = {
                ...process.env,
                GOOS: target.os,
                GOARCH: target.arch,
                CGO_ENABLED: '0'
            };
            
            execSync(buildCmd, { env, stdio: 'pipe' });
            
            const fileSize = fs.statSync(outputFile).size;
            colorLog(`    ✓ ${target.dir}/retroterm${target.ext} (${(fileSize / 1024 / 1024).toFixed(1)} MB)`, 'green');
              // Copy configuration files to each target directory
            for (const file of config.additionalFiles) {
                if (fs.existsSync(file)) {
                    copyFile(file, path.join(targetDir, file));
                }
            }
              // Create systemd service script for Linux targets
            if (target.os === 'linux') {
                createSystemdService(targetDir, target.dir);
            }
            
            // Copy security setup files to all backend directories
            copySecuritySetupFiles(targetDir);
            
        } catch (error) {
            const errorMsg = `Failed to build ${target.os}/${target.arch}: ${error.message}`;
            colorLog(`    ✗ ${errorMsg}`, 'red');
            buildErrors.push(errorMsg);
        }
    }
    
    if (buildErrors.length > 0) {
        colorLog('\nBackend build warnings:', 'yellow');
        buildErrors.forEach(error => colorLog(`  ${error}`, 'yellow'));
    }
}

function processCSS() {
    colorLog('Processing CSS files...', 'green');
    const cssFile = path.join(config.cssDir, 'retroterminal.css');
    
    if (!fs.existsSync(cssFile)) {
        colorLog('  Warning: retroterminal.css not found', 'yellow');
        return;
    }
    
    if (options.skipMinify) {
        copyFile(cssFile, path.join(config.distDir, 'css', 'retroterminal.css'));
        colorLog('  Copied (unminified): retroterminal.css', 'yellow');
    } else {
        try {
            execSync(`npx terser ${cssFile} --compress --output ${path.join(config.distDir, 'css', 'retroterminal.min.css')}`, { stdio: 'pipe' });
            colorLog('  Minified: retroterminal.css -> retroterminal.min.css', 'green');
        } catch {
            copyFile(cssFile, path.join(config.distDir, 'css', 'retroterminal.css'));
            colorLog('  Copied (unminified): retroterminal.css', 'yellow');
        }
    }
}

function copyAssets() {
    colorLog('Copying assets...', 'green');
    
    if (copyDirectory(config.assetsDir, path.join(config.distDir, 'assets'))) {
        colorLog('  Copied assets directory', 'green');
    }
    
    if (copyDirectory(config.chessGfxDir, path.join(config.distDir, 'chess_gfx'))) {
        colorLog('  Copied chess_gfx directory', 'green');
    }
      if (copyDirectory(config.examplesDir, path.join(config.distDir, 'examples'))) {
        colorLog('  Copied examples directory', 'green');
    }
      if (copyDirectory(config.dysonDir, path.join(config.distDir, 'dyson'))) {
        colorLog('  Copied dyson directory', 'green');
    }
    
    if (copyDirectory(config.promptsDir, path.join(config.distDir, 'prompts'))) {
        colorLog('  Copied prompts directory', 'green');
    }
}

function createProductionHTML() {
    colorLog('Creating production HTML file...', 'green');
    
    const htmlFile = 'retroterminal.html';
    if (!fs.existsSync(htmlFile)) {
        colorLog('  ERROR: retroterminal.html not found', 'red');
        return;
    }
    
    let htmlContent = fs.readFileSync(htmlFile, 'utf8');
      // Determine bundle filename based on whether minification succeeded
    const bundleFile = fs.existsSync(path.join(config.distDir, 'js', 'retroterm.min.js')) 
        ? 'retroterm.min.js' 
        : 'retroterm.js';
    
    // Create simplified script section
    const newScripts = `    <!-- Production JavaScript Bundle -->
    <script src="js/samjs.min.js"></script>
    <script src="js/three.min.js" id="threejs"></script>
    <script src="js/jsSID.js"></script>
    <script src="js/${bundleFile}"></script>`;    // Replace all individual custom JS script tags with the bundle
    // Handle all script tags for custom JS files
    htmlContent = htmlContent.replace(/<script src="js\/config\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/dynamicViewport\.js"><\/script>[^\r\n]*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/samjs\.min\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/samInit\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/three\.min\.js" id="threejs"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script type="module" src="js\/retrographics\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script type="module" src="js\/spriteManager\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script type="module" src="js\/vectorManager\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/retrosound\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/sidPlayer\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/ansiParser\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/auth\.js"><\/script>\s*/g, '');
    htmlContent = htmlContent.replace(/<script src="js\/retroconsole\.js"><\/script>\s*/g, '');
    
    // Remove the comment line
    htmlContent = htmlContent.replace(/<!-- Nur das Hauptmodul als Entry-Point laden: -->\s*/g, '');
    
    // Replace the final retroterminal.js script with our bundle
    htmlContent = htmlContent.replace(/<script type="module" src="js\/retroterminal\.js"><\/script>/, newScripts);
    
    // Update CSS reference
    const cssFile = fs.existsSync(path.join(config.distDir, 'css', 'retroterminal.min.css')) 
        ? 'css/retroterminal.min.css' 
        : 'css/retroterminal.css';
    htmlContent = htmlContent.replace(/href="retroterminal\.css"/g, `href="${cssFile}"`);
    
    // Write production HTML
    fs.writeFileSync(path.join(config.distDir, 'index.html'), htmlContent);
    colorLog('  Created: index.html', 'green');
}

function copyAdditionalFiles() {
    colorLog('Copying configuration files...', 'green');
    for (const file of config.additionalFiles) {
        if (fs.existsSync(file)) {
            copyFile(file, path.join(config.distDir, file));
            colorLog(`  Copied: ${file}`, 'green');
        }
    }
}

function calculateSizes(jsStats) {
    const compressionRatio = jsStats.original > 0 ? ((1 - jsStats.minified / jsStats.original) * 100).toFixed(1) : '0.0';
    
    colorLog('\n=== Build Complete ===', 'cyan');
    colorLog(`Distribution created in: ${config.distDir}`, 'green');
    colorLog(`Original JS size: ${(jsStats.original / 1024).toFixed(1)} KB`, 'yellow');
    colorLog(`Bundle size: ${(jsStats.minified / 1024).toFixed(1)} KB`, 'yellow');
    colorLog(`Compression ratio: ${compressionRatio}%`, 'green');
    
    // Check if backend was built
    const backendDir = path.join(config.distDir, config.backendDir);
    if (fs.existsSync(backendDir)) {
        colorLog('\nBackend executables included:', 'cyan');
        for (const target of config.backendTargets) {
            const targetDir = path.join(backendDir, target.dir);
            const execFile = path.join(targetDir, `retroterm${target.ext}`);
            if (fs.existsSync(execFile)) {
                const size = fs.statSync(execFile).size;
                colorLog(`  ${target.dir}/retroterm${target.ext} (${(size / 1024 / 1024).toFixed(1)} MB)`, 'green');
            }
        }
    }
    
    colorLog(`\nTo deploy, copy the contents of the '${config.distDir}' directory to your web server.`, 'cyan');
    colorLog(`For standalone deployment, use the backend executables in the '${config.backendDir}' directory.`, 'cyan');
}

function createSystemdService(targetDir, platformName) {
    colorLog(`  Creating systemd service for ${platformName}...`, 'cyan');
    
    // Create systemd service file content
    const serviceContent = `[Unit]
Description=TinyOS Retro Terminal Server
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
User=tinyos
WorkingDirectory=%h/tinyos
ExecStart=%h/tinyos/retroterm
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`;

    // Create installation script
    const installScript = `#!/bin/bash
# TinyOS Service Installation Script for Debian-compatible systems
# This script sets up TinyOS as a systemd service

set -e

# Configuration
SERVICE_NAME="tinyos"
SERVICE_USER="tinyos"
INSTALL_DIR="/home/$SERVICE_USER/tinyos"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

echo "Starting TinyOS service installation..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Create service user if it doesn't exist
if ! id "$SERVICE_USER" &>/dev/null; then
    echo "Creating service user: $SERVICE_USER"
    useradd --system --create-home --shell /bin/false "$SERVICE_USER"
else
    echo "Service user $SERVICE_USER already exists"
fi

# Create installation directory
echo "Setting up installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"

# Copy all files from current directory to installation directory
echo "Copying TinyOS files..."
cp -r . "$INSTALL_DIR/"

# Set correct permissions
echo "Setting file permissions..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod +x "$INSTALL_DIR/retroterm"

# Create systemd service file
echo "Creating systemd service file..."
cat > "$SERVICE_FILE" << 'EOF'
[Unit]
Description=TinyOS Retro Terminal Server
After=network.target
StartLimitIntervalSec=0

[Service]
Type=simple
Restart=always
RestartSec=1
User=tinyos
WorkingDirectory=/home/tinyos/tinyos
ExecStart=/home/tinyos/tinyos/retroterm
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd daemon
echo "Reloading systemd daemon..."
systemctl daemon-reload

# Enable service for automatic startup
echo "Enabling $SERVICE_NAME service for automatic startup..."
systemctl enable "$SERVICE_NAME"

# Start the service immediately
echo "Starting $SERVICE_NAME service..."
systemctl start "$SERVICE_NAME"

# Check service status
echo "Checking service status..."
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ TinyOS service is running successfully!"
    echo ""
    echo "Service management commands:"
    echo "  Status:  sudo systemctl status $SERVICE_NAME"
    echo "  Stop:    sudo systemctl stop $SERVICE_NAME"
    echo "  Start:   sudo systemctl start $SERVICE_NAME"
    echo "  Restart: sudo systemctl restart $SERVICE_NAME"
    echo "  Logs:    sudo journalctl -u $SERVICE_NAME -f"
    echo ""
    echo "The service will automatically start on system boot."
else
    echo "❌ Failed to start TinyOS service"
    echo "Check logs with: sudo journalctl -u $SERVICE_NAME"
    exit 1
fi

echo "Installation completed successfully!"
`;

    // Create uninstall script
    const uninstallScript = `#!/bin/bash
# TinyOS Service Uninstallation Script

set -e

SERVICE_NAME="tinyos"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

echo "Uninstalling TinyOS service..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root (use sudo)"
    exit 1
fi

# Stop service if running
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Stopping $SERVICE_NAME service..."
    systemctl stop "$SERVICE_NAME"
fi

# Disable service
if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    echo "Disabling $SERVICE_NAME service..."
    systemctl disable "$SERVICE_NAME"
fi

# Remove service file
if [ -f "$SERVICE_FILE" ]; then
    echo "Removing service file..."
    rm "$SERVICE_FILE"
fi

# Reload systemd daemon
echo "Reloading systemd daemon..."
systemctl daemon-reload

echo "TinyOS service uninstalled successfully!"
echo "Note: The service user and installation directory were not removed."
echo "To remove them manually:"
echo "  sudo userdel tinyos"
echo "  sudo rm -rf /home/tinyos"
`;

    // Write service file
    const serviceFilePath = path.join(targetDir, 'tinyos.service');
    fs.writeFileSync(serviceFilePath, serviceContent);
    
    // Write installation script
    const installScriptPath = path.join(targetDir, 'install-service.sh');
    fs.writeFileSync(installScriptPath, installScript);
    
    // Write uninstall script
    const uninstallScriptPath = path.join(targetDir, 'uninstall-service.sh');
    fs.writeFileSync(uninstallScriptPath, uninstallScript);
      colorLog(`    Created systemd service files for ${platformName}`, 'green');
}

function copySecuritySetupFiles(targetDir) {
    // Copy security setup files to backend directory
    const securityFiles = [
        'SECURITY_SETUP.md',
        'setup-env.ps1',
        'settings.cfg.template'
    ];
    
    for (const file of securityFiles) {
        if (fs.existsSync(file)) {
            copyFile(file, path.join(targetDir, file));
        }
    }
}

function createImportantTxt() {
    colorLog('Creating important deployment instructions...', 'green');
    
    const importantContent = `IMPORTANT DEPLOYMENT INSTRUCTIONS
====================================

To deploy TinyOS, you must copy ALL files from the appropriate backend directory 
to your project root directory.

DEPLOYMENT STEPS:
1. Choose your platform directory:
   - Windows: backend/windows-x64/
   - Linux 64-bit: backend/linux-x64/
   - Linux ARM64: backend/linux-arm64/

2. Copy ALL files from the chosen backend directory to your project root
   (the same directory where you want to run the retroterm executable)

3. The directory structure must include these folders:
   - assets/
   - css/
   - js/
   - examples/
   - chess_gfx/
   - dyson/
   - prompts/

4. For security setup, follow the instructions in SECURITY_SETUP.md

IMPORTANT NOTES:
- The backend executable must be run from the project root directory
- Do NOT run the executable from within the backend/ subdirectory
- The executable needs access to examples/, chess_gfx/, and dyson/ folders
- Static files (js/, css/, assets/) are served by the backend
- The prompts/ folder is used internally by the backend

For detailed deployment instructions, see BUILD_SYSTEM.md
`;
    
    const importantPath = path.join(config.distDir, 'IMPORTANT.txt');
    fs.writeFileSync(importantPath, importantContent);
    colorLog('  Created IMPORTANT.txt with deployment instructions', 'green');
}

// Main build process
function main() {
    if (options.help) {
        showHelp();
        return;
    }
    
    colorLog('=== TinyOS Build Script ===', 'cyan');
    colorLog('Building production distribution...', 'green');
    
    checkDependencies();
    
    if (options.clean) {
        cleanDist();
    }
    
    createDistStructure();
    copyExternalJSFiles();
    const jsStats = bundleAndMinifyJS();
    buildBackendExecutables();
    processCSS();
    copyAssets();    createProductionHTML();
    copyAdditionalFiles();
    createImportantTxt();
    calculateSizes(jsStats);
}

// Run the build
main();
