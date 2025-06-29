#!/bin/bash
# Build Script for Linux/Unix
# Creates a production-ready distribution of the TinyOS web application

# Configuration
DIST_DIR="dist"
JS_DIR="js"
CSS_DIR="css"
ASSETS_DIR="assets"
CHESS_GFX_DIR="chess_gfx"
EXAMPLES_DIR="examples"

# Parse command line arguments
CLEAN=false
SKIP_MINIFY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --skip-minify)
            SKIP_MINIFY=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--clean] [--skip-minify]"
            exit 1
            ;;
    esac
done

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}=== TinyOS Build Script for Linux/Unix ===${NC}"
echo -e "${GREEN}Building production distribution...${NC}"

# Check if Node.js and npm are available
if ! command -v node &> /dev/null; then
    echo -e "${RED}ERROR: Node.js is required but not found!${NC}"
    echo -e "${YELLOW}Please install Node.js from https://nodejs.org/ or via your package manager:${NC}"
    echo -e "${YELLOW}  Ubuntu/Debian: sudo apt install nodejs npm${NC}"
    echo -e "${YELLOW}  CentOS/RHEL: sudo yum install nodejs npm${NC}"
    echo -e "${YELLOW}  Arch: sudo pacman -S nodejs npm${NC}"
    exit 1
fi

if ! command -v npm &> /dev/null; then
    echo -e "${RED}ERROR: npm is required but not found!${NC}"
    echo -e "${YELLOW}Please install npm via your package manager${NC}"
    exit 1
fi

NODE_VERSION=$(node --version)
NPM_VERSION=$(npm --version)
echo -e "${GREEN}Node.js version: $NODE_VERSION${NC}"
echo -e "${GREEN}npm version: $NPM_VERSION${NC}"

# Check if terser is installed globally
if ! command -v terser &> /dev/null; then
    echo -e "${YELLOW}Terser not found. Installing globally...${NC}"
    if ! npm install -g terser; then
        echo -e "${RED}ERROR: Failed to install terser!${NC}"
        echo -e "${YELLOW}You can install it manually with: sudo npm install -g terser${NC}"
        echo -e "${YELLOW}Or without sudo if you have a local npm setup${NC}"
        exit 1
    fi
else
    TERSER_VERSION=$(terser --version)
    echo -e "${GREEN}Terser version: $TERSER_VERSION${NC}"
fi

# Clean dist directory if requested or if it exists
if [ "$CLEAN" = true ] || [ -d "$DIST_DIR" ]; then
    echo -e "${YELLOW}Cleaning distribution directory...${NC}"
    rm -rf "$DIST_DIR"
fi

# Create dist directory structure
echo -e "${GREEN}Creating distribution directory structure...${NC}"
mkdir -p "$DIST_DIR"/{js,css,assets,chess_gfx,examples}

# List of JavaScript files in load order (from HTML)
declare -a jsFiles=(
    "config.js"
    "dynamicViewport.js" 
    "samjs.min.js"
    "samInit.js"
    "three.min.js"
    "retrographics.js"
    "spriteManager.js" 
    "vectorManager.js"
    "retrosound.js"
    "sidPlayer.js"
    "ansiParser.js"
    "auth.js"
    "retroconsole.js"
    "retroterminal.js"
)

# Files to copy without bundling (already minified or special handling)
declare -a copyJsFiles=(
    "samjs.min.js"
    "three.min.js"
    "jsSID.js"
)

# Copy already minified files
echo -e "${GREEN}Copying pre-minified JavaScript files...${NC}"
for file in "${copyJsFiles[@]}"; do
    if [ -f "$JS_DIR/$file" ]; then
        cp "$JS_DIR/$file" "$DIST_DIR/js/$file"
        echo -e "  ${GREEN}Copied: $file${NC}"
    else
        echo -e "  ${YELLOW}Warning: $file not found${NC}"
    fi
done

# Bundle and minify custom JavaScript files
declare -a bundleFiles=()
for file in "${jsFiles[@]}"; do
    if [[ ! " ${copyJsFiles[@]} " =~ " ${file} " ]]; then
        bundleFiles+=("$file")
    fi
done

if [ "$SKIP_MINIFY" = false ]; then
    echo -e "${GREEN}Bundling and minifying custom JavaScript files...${NC}"
    
    # Create a temporary bundle file
    tempBundle="temp_bundle.js"
    : > "$tempBundle"  # Create empty file
    
    for file in "${bundleFiles[@]}"; do
        if [ -f "$JS_DIR/$file" ]; then
            echo -e "  ${GREEN}Adding to bundle: $file${NC}"
            echo -e "\n\n/* === $file === */" >> "$tempBundle"
            cat "$JS_DIR/$file" >> "$tempBundle"
        else
            echo -e "  ${YELLOW}Warning: $file not found${NC}"
        fi
    done
    
    # Minify the bundle
    echo -e "${GREEN}Minifying bundle...${NC}"
    if terser "$tempBundle" --compress --mangle --output "$DIST_DIR/js/app.min.js"; then
        echo -e "  ${GREEN}Bundle created: js/app.min.js${NC}"
    else
        echo -e "  ${RED}ERROR: Minification failed!${NC}"
        # Fall back to copying unminified
        cp "$tempBundle" "$DIST_DIR/js/app.js"
        echo -e "  ${YELLOW}Fallback: Created unminified bundle${NC}"
    fi
    
    # Clean up temp file
    rm -f "$tempBundle"
else
    echo -e "${YELLOW}Skipping minification, copying files individually...${NC}"
    for file in "${bundleFiles[@]}"; do
        if [ -f "$JS_DIR/$file" ]; then
            cp "$JS_DIR/$file" "$DIST_DIR/js/$file"
            echo -e "  ${GREEN}Copied: $file${NC}"
        fi
    done
fi

# Copy and minify CSS
echo -e "${GREEN}Processing CSS files...${NC}"
if [ -f "$CSS_DIR/retroterminal.css" ]; then
    # Try to minify CSS with terser (it can handle CSS too)
    if terser "$CSS_DIR/retroterminal.css" --compress --output "$DIST_DIR/css/retroterminal.min.css" 2>/dev/null; then
        echo -e "  ${GREEN}Minified: retroterminal.css -> retroterminal.min.css${NC}"
    else
        # Fallback: just copy the CSS file
        cp "$CSS_DIR/retroterminal.css" "$DIST_DIR/css/retroterminal.css"
        echo -e "  ${YELLOW}Copied (unminified): retroterminal.css${NC}"
    fi
else
    echo -e "  ${YELLOW}Warning: retroterminal.css not found${NC}"
fi

# Copy assets
echo -e "${GREEN}Copying assets...${NC}"
if [ -d "$ASSETS_DIR" ]; then
    cp -r "$ASSETS_DIR"/* "$DIST_DIR/assets/" 2>/dev/null || true
    echo -e "  ${GREEN}Copied assets directory${NC}"
fi

if [ -d "$CHESS_GFX_DIR" ]; then
    cp -r "$CHESS_GFX_DIR"/* "$DIST_DIR/chess_gfx/" 2>/dev/null || true
    echo -e "  ${GREEN}Copied chess_gfx directory${NC}"
fi

if [ -d "$EXAMPLES_DIR" ]; then
    cp -r "$EXAMPLES_DIR"/* "$DIST_DIR/examples/" 2>/dev/null || true
    echo -e "  ${GREEN}Copied examples directory${NC}"
fi

# Create production HTML file
echo -e "${GREEN}Creating production HTML file...${NC}"
if [ -f "retroterminal.html" ]; then
    htmlContent=$(cat "retroterminal.html")
    
    # Replace script includes with bundled version
    if [ "$SKIP_MINIFY" = false ]; then
        # Create new scripts section
        newScripts='    <!-- Production JavaScript Bundle -->
    <script src="js/samjs.min.js"></script>
    <script src="js/three.min.js" id="threejs"></script>
    <script src="js/jsSID.js"></script>
    <script src="js/app.min.js"></script>'
        
        # Replace individual script tags with bundle using sed
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/config\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/dynamicViewport\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/samInit\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script type="module" src="js\/retrographics\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script type="module" src="js\/spriteManager\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script type="module" src="js\/vectorManager\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/retrosound\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/sidPlayer\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/ansiParser\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/auth\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed '/<script src="js\/retroconsole\.js"><\/script>/d')
        htmlContent=$(echo "$htmlContent" | sed "s|<script type=\"module\" src=\"js/retroterminal\.js\"></script>|$newScripts|")
    fi
    
    # Update CSS reference to minified version if it exists
    if [ -f "$DIST_DIR/css/retroterminal.min.css" ]; then
        htmlContent=$(echo "$htmlContent" | sed 's|href="retroterminal\.css"|href="css/retroterminal.min.css"|')
    else
        htmlContent=$(echo "$htmlContent" | sed 's|href="retroterminal\.css"|href="css/retroterminal.css"|')
    fi
    
    # Write production HTML
    echo "$htmlContent" > "$DIST_DIR/index.html"
    echo -e "  ${GREEN}Created: index.html${NC}"
else
    echo -e "  ${RED}ERROR: retroterminal.html not found${NC}"
fi

# Copy additional files that might be needed
additionalFiles=("SECURITY_SETUP.md" "settings.cfg.template")
for file in "${additionalFiles[@]}"; do
    if [ -f "$file" ]; then
        cp "$file" "$DIST_DIR/$file"
        echo -e "  ${GREEN}Copied: $file${NC}"
    fi
done

# Calculate sizes
originalSize=0
distSize=0

if [ -d "$JS_DIR" ]; then
    for file in "$JS_DIR"/*.js; do
        if [ -f "$file" ]; then
            size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
            originalSize=$((originalSize + size))
        fi
    done
fi

if [ -d "$DIST_DIR/js" ]; then
    for file in "$DIST_DIR/js"/*.js; do
        if [ -f "$file" ]; then
            size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
            distSize=$((distSize + size))
        fi
    done
fi

if [ $originalSize -gt 0 ]; then
    compressionRatio=$(echo "scale=1; (1 - $distSize / $originalSize) * 100" | bc -l 2>/dev/null || echo "0.0")
else
    compressionRatio="0.0"
fi

originalKB=$(echo "scale=1; $originalSize / 1024" | bc -l 2>/dev/null || echo "0.0")
distKB=$(echo "scale=1; $distSize / 1024" | bc -l 2>/dev/null || echo "0.0")

echo -e "\n${CYAN}=== Build Complete ===${NC}"
echo -e "${GREEN}Distribution created in: $DIST_DIR${NC}"
echo -e "${YELLOW}Original JS size: ${originalKB} KB${NC}"
echo -e "${YELLOW}Minified JS size: ${distKB} KB${NC}"
echo -e "${GREEN}Compression ratio: ${compressionRatio}%${NC}"
echo -e "\n${CYAN}To deploy, copy the contents of the '$DIST_DIR' directory to your web server.${NC}"
