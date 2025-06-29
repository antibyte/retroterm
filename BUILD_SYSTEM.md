# Build System Documentation

This document describes the build system for the TinyOS retro terminal web application.

## Overview

The build system creates a production-ready distribution that includes:
- A single minified JavaScript bundle of all custom code
- External JavaScript libraries as separate files
- Cross-platform backend executables (Windows, Linux x64, Linux ARM64)
- Systemd service scripts for Linux distributions (automatic installation and startup)
- All necessary assets (CSS, images, examples, dyson content, prompts, etc.)
- Production HTML file with correct script references

## Prerequisites

- Node.js (v14 or later)
- Go (for backend compilation)
- npm packages: `terser`, `http-server`

## Build Commands

### Production Build
```bash
npm run build
```
Creates a minified, production-ready distribution in the `dist/` directory.

### Development Build
```bash
npm run build:dev
```
Creates an unminified bundle for easier debugging.

### Clean Build
```bash
npm run build -- --clean
```
Removes the existing `dist/` directory before building.

### Local Server
```bash
npm run serve
```
Starts a local HTTP server serving the `dist/` directory on port 8080.

## Build Script Configuration

The build configuration is in `build.js`:

### JavaScript Files to Bundle
Custom JavaScript files are bundled in this order:
1. `config.js` - Global configuration
2. `dynamicViewport.js` - Viewport management
3. `samInit.js` - Speech synthesis initialization
4. `retrographics.js` - Graphics engine
5. `spriteManager.js` - Sprite management
6. `vectorManager.js` - Vector graphics
7. `retrosound.js` - Sound system
8. `sidPlayer.js` - SID audio player
9. `ansiParser.js` - ANSI escape sequence parser
10. `auth.js` - Authentication system
11. `retroconsole.js` - Console interface
12. `retroterminal.js` - Main terminal logic

### External JavaScript Libraries
These files are copied as-is (not bundled):
- `samjs.min.js` - Speech synthesis
- `three.min.js` - 3D graphics library
- `jsSID.js` - SID audio codec

### Backend Targets
Cross-compiled Go executables:
- `windows-x64/retroterm.exe` - Windows 64-bit
- `linux-x64/retroterm` - Linux 64-bit
- `linux-arm64/retroterm` - Linux ARM64

## Distribution Structure

```
dist/
├── index.html                    # Production HTML file
├── SECURITY_SETUP.md            # Security configuration guide
├── settings.cfg.template         # Configuration template
├── assets/                       # Static assets
│   ├── background.png
│   └── floppy.mp3
├── backend/                      # Backend executables
│   ├── windows-x64/
│   │   ├── retroterm.exe
│   │   ├── SECURITY_SETUP.md
│   │   └── settings.cfg.template
│   ├── linux-x64/
│   │   ├── retroterm
│   │   ├── install-service.sh    # Systemd service installer
│   │   ├── uninstall-service.sh  # Systemd service remover
│   │   ├── tinyos.service         # Systemd service file
│   │   ├── SECURITY_SETUP.md
│   │   └── settings.cfg.template
│   └── linux-arm64/
│       ├── retroterm
│       ├── install-service.sh    # Systemd service installer
│       ├── uninstall-service.sh  # Systemd service remover
│       ├── tinyos.service         # Systemd service file
│       ├── SECURITY_SETUP.md
│       └── settings.cfg.template
├── chess_gfx/                    # Chess game graphics
├── css/
│   └── retroterminal.css         # Stylesheets
├── dyson/                        # Dyson storyline content
├── examples/                     # Example BASIC programs
├── IMPORTANT.txt                 # Deployment instructions
├── prompts/                      # Backend prompt templates
└── js/                           # JavaScript files
    ├── retroterm.min.js          # Bundled and minified custom code
    ├── samjs.min.js              # External: Speech synthesis
    ├── three.min.js              # External: 3D graphics
    └── jsSID.js                  # External: SID audio
```

## HTML References

The production HTML file (`dist/index.html`) includes only these JavaScript files:
```html
<script src="js/samjs.min.js"></script>
<script src="js/three.min.js" id="threejs"></script>
<script src="js/jsSID.js"></script>
<script src="js/retroterm.min.js"></script>
```

## Maintenance

### Adding New JavaScript Files
1. Create the new JavaScript file in the `js/` directory
2. Add the filename to the `bundleJsFiles` array in `build.js`
3. Update this documentation

### Adding New External Libraries
1. Add the library file to the `js/` directory
2. Add the filename to the `externalJsFiles` array in `build.js`
3. Update the HTML template (`retroterminal.html`) to include the library
4. Update this documentation

### Build System Reminders
All custom JavaScript files include a comment at the top reminding maintainers to update the build system when making changes:

```javascript
/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */
```

## Deployment

### Web Deployment
Copy the entire contents of the `dist/` directory to your web server.

### Standalone Deployment
Use the appropriate executable from `dist/backend/` for your target platform:
- Windows: `windows-x64/retroterm.exe`
- Linux 64-bit: `linux-x64/retroterm`
- Linux ARM64: `linux-arm64/retroterm`

Each platform directory includes the necessary configuration files.

## Deployment Instructions

The build system creates an `IMPORTANT.txt` file in the `dist/` root directory with detailed deployment instructions.

### Key Deployment Requirements
1. **Copy ALL files from the appropriate backend directory** to your project root
2. **Run the executable from the project root directory** - NOT from within the backend subdirectory
3. **Ensure folder structure is maintained** - the executable needs access to:
   - `examples/` - Example BASIC programs (direct file access)
   - `chess_gfx/` - Chess game graphics (direct file access)  
   - `dyson/` - Storyline content (direct file access)
   - `prompts/` - Backend prompt templates (internal use)
   - `js/`, `css/`, `assets/` - Static web files (served by backend)

### Security Setup
Each backend directory includes security configuration files:
- `SECURITY_SETUP.md` - Detailed security setup guide
- `setup-env.ps1` - PowerShell script for environment variable setup
- `settings.cfg.template` - Configuration template

Follow the security setup guide before running in production.

## Linux Service Installation

For Linux deployments, the build system automatically creates systemd service scripts that allow TinyOS to run as a system service.

### Automatic Installation
Each Linux backend directory includes:
- `install-service.sh` - Complete installation and service setup script
- `uninstall-service.sh` - Service removal script  
- `tinyos.service` - Systemd service definition

### Installation Process
1. Copy the appropriate Linux backend directory to your target system
2. Run the installation script as root:
   ```bash
   sudo ./install-service.sh
   ```

The installation script will:
- Create a dedicated `tinyos` system user
- Install TinyOS to `/home/tinyos/tinyos/`
- Set up the systemd service
- Enable automatic startup on boot
- Start the service immediately

### Service Management
After installation, use standard systemd commands:
```bash
sudo systemctl status tinyos     # Check service status
sudo systemctl stop tinyos      # Stop service
sudo systemctl start tinyos     # Start service  
sudo systemctl restart tinyos   # Restart service
sudo journalctl -u tinyos -f    # View logs
```

### Uninstallation
To remove the service:
```bash
sudo ./uninstall-service.sh
```

Note: The service user and installation directory are preserved and must be removed manually if desired.

## Troubleshooting

### Minification Fails
If minification fails due to JavaScript syntax errors, the build system will automatically fall back to an unminified bundle. Check the terminal output for specific error messages.

### Backend Build Fails
Ensure Go is installed and available in your PATH. The build system will skip backend compilation if Go is not found.

### Missing Files
If assets are missing from the dist directory, check that the source files exist and that the file paths in `build.js` are correct.

## Performance

The build system achieves significant size reduction:
- Original custom JavaScript: ~363 KB
- Minified bundle: ~138 KB
- Compression ratio: ~62%

This reduction improves load times and reduces bandwidth usage in production.

## Backend Architecture

### Static File Serving
The backend now uses efficient directory-based static file serving for web assets:
- `/js/` - JavaScript files (served from js/ directory)
- `/css/` - Stylesheets (served from css/ directory)  
- `/assets/` - Static assets (served from assets/ directory)

### Direct File Access
The following directories are accessed directly by the backend (not served via HTTP):
- `examples/` - Example BASIC programs (direct filesystem access)
- `chess_gfx/` - Chess graphics (direct filesystem access)
- `dyson/` - Dyson storyline content (direct filesystem access)
- `prompts/` - Backend prompt templates (internal use only)

This approach improves performance and reduces maintenance overhead compared to the previous individual file handler approach.
