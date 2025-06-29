/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

/**
 * ANSI Color and Control Codes Support for RetroTerminal
 * Provides parsing and rendering of ANSI escape sequences in green terminal theme
 */



// ANSI Color Mapping - Alle Farben werden in verschiedene Grüntöne konvertiert
const ANSI_GREEN_COLORS = {
    // Standard ANSI Colors (30-37, 90-97) -> Grüntöne
    black: '#001100',           // Sehr dunkles Grün
    red: '#004400',             // Dunkles Grün (für Rot)
    green: '#00aa00',           // Standard Terminal-Grün
    yellow: '#00cc44',          // Helles Gelbgrün
    blue: '#006666',            // Blaugrün
    magenta: '#004466',         // Dunkles Blaugrün
    cyan: '#00aaaa',            // Cyan-Grün
    white: '#00ff55',           // Helles Grün (für Weiß)
    
    // Bright Colors (90-97) -> Hellere Grüntöne
    brightBlack: '#003300',     // Dunkles Grün
    brightRed: '#006600',       // Mittleres Grün
    brightGreen: '#00ff00',     // Helles Grün
    brightYellow: '#88ff44',    // Sehr helles Gelbgrün
    brightBlue: '#00aacc',      // Helles Blaugrün
    brightMagenta: '#4488cc',   // Helles Blaugrün
    brightCyan: '#44ffaa',      // Helles Cyan-Grün
    brightWhite: '#aaffaa',     // Sehr helles Grün
    
    // Background colors - Dunklere Varianten
    bgBlack: '#000000',
    bgRed: '#002200',
    bgGreen: '#004400',
    bgYellow: '#006622',
    bgBlue: '#003333',
    bgMagenta: '#002244',
    bgCyan: '#004444',
    bgWhite: '#006644'
};

// ANSI Parser für Escape-Sequenzen
class ANSIParser {
    constructor() {
        this.reset();
    }
      reset() {
        this.currentForeground = ANSI_GREEN_COLORS.white;
        this.currentBackground = ANSI_GREEN_COLORS.bgBlack;
        this.bold = false;
        this.dim = false;
        this.italic = false;
        this.underline = false;
        this.blink = false;
        this.reverse = false;
        this.hidden = false;
        this.strikethrough = false;
        this.cursorVisible = true; // Default cursor visibility
    }
      // Parse ANSI escape sequences and return formatted text segments
    parseText(text) {
        const segments = [];
        let currentPos = 0;
        let currentText = '';
        
        // Enhanced regex for ANSI escape sequences including private modes
        // Matches: ESC[ + optional ? + parameters + command letter
        const ansiRegex = /\x1b\[(\??)([0-9;]*?)([A-Za-z])/g;
        let match;
        
        while ((match = ansiRegex.exec(text)) !== null) {
            // Text vor der Escape-Sequenz hinzufügen
            const beforeText = text.substring(currentPos, match.index);
            if (beforeText) {
                segments.push({
                    text: beforeText,
                    style: this.getCurrentStyle()
                });
            }
            
            // Escape-Sequenz verarbeiten
            const isPrivate = match[1] === '?';
            let params = match[2] ? match[2].split(';').map(p => parseInt(p) || 0) : [0];
            const command = match[3];
            
            // Mark private mode parameters with offset to distinguish them
            if (isPrivate && params.length > 0) {
                params = params.map(p => p + 1000); // Add 1000 to mark as private mode
            }
            
            this.processANSICommand(params, command);
            
            currentPos = match.index + match[0].length;
        }
        
        // Restlichen Text hinzufügen
        const remainingText = text.substring(currentPos);
        if (remainingText) {
            segments.push({
                text: remainingText,
                style: this.getCurrentStyle()
            });
        }
        
        return segments;
    }
    
    // Main parse method used by frontend
    parse(data) {
        const lines = [];
        const commands = [];
        
        // Split data by lines but preserve line endings for processing
        const rawLines = data.split(/\r\n|\r|\n/);
        
        for (let i = 0; i < rawLines.length; i++) {
            const line = rawLines[i];
            
            // Process ANSI commands that affect terminal state
            this.extractTerminalCommands(line, commands);
            
            // Parse the line for styled text segments
            const segments = this.parseText(line);
            
            // Create line object
            const lineObj = {
                text: this.extractPlainText(line),
                segments: segments,
                raw: line
            };
            
            lines.push(lineObj);
        }
        
        return {
            lines: lines,
            commands: commands
        };
    }
      processANSICommand(params, command) {
        switch (command) {
            case 'm': // SGR (Select Graphic Rendition)
                this.processSGR(params);
                break;
            case 'K': // EL (Erase in Line)
                // Für Terminal-Darstellung ignorieren oder später implementieren
                break;
            case 'J': // ED (Erase in Display)
                // Für Terminal-Darstellung ignorieren oder später implementieren
                break;
            case 'H': // CUP (Cursor Position)
            case 'f': // HVP (Horizontal and Vertical Position)
                // Cursor-Positionierung - könnte später für LOCATE verwendet werden
                break;
            case 'A': // CUU (Cursor Up)
            case 'B': // CUD (Cursor Down)
            case 'C': // CUF (Cursor Forward)
            case 'D': // CUB (Cursor Back)
                // Cursor-Bewegung - für Terminal-Darstellung meist nicht relevant
                break;
            case 'h': // Set Mode (Private Mode with ?)
            case 'l': // Reset Mode (Private Mode with ?)
                this.processPrivateMode(params, command);
                break;            case 's': // Save Cursor Position (SCOSC)
            case 'u': // Restore Cursor Position (SCORC)
                // Cursor-Position speichern/wiederherstellen
                break;
            case 'S': // Scroll Up (SU)
            case 'T': // Scroll Down (SD)
                // Scrolling commands
                break;
            case 'L': // Insert Lines (IL)
            case 'M': // Delete Lines (DL)
                // Line insertion/deletion
                break;
            case 'P': // Delete Characters (DCH)
            case '@': // Insert Characters (ICH)
                // Character insertion/deletion
                break;
            case 'X': // Erase Characters (ECH)
                // Character erasure
                break;
            case 'G': // Cursor Horizontal Absolute (CHA)
            case '`': // Character Position Absolute (HPA)
                // Horizontal cursor positioning
                break;
            case 'd': // Line Position Absolute (VPA)
                // Vertical cursor positioning
                break;
            case 'n': // Device Status Report (DSR)
                // Status reports and cursor position queries
                break;
            default:
                // Unbekannte Sequenz ignorieren
                break;
        }
    }
    
    // Process DEC Private Mode sequences (ESC[?...h/l)
    processPrivateMode(params, command) {
        // Check if this is a private mode sequence (starts with ?)
        if (params.length > 0 && params[0] >= 1000) {
            // Extract the actual parameter by removing the ? indicator
            const mode = params[0] % 1000;
            
            switch (mode) {
                case 25: // DECTCEM - DEC Text Cursor Enable Mode
                    this.cursorVisible = (command === 'h');
                    break;
                case 1: // DECCKM - DEC Cursor Keys Mode
                    // Application cursor keys mode
                    break;
                case 7: // DECAWM - DEC Auto Wrap Mode
                    // Auto wrap mode
                    break;
                case 47: // Alternate Screen Buffer
                case 1047: // Alternate Screen Buffer (xterm)
                case 1049: // Alternate Screen Buffer with save/restore cursor
                    // Screen buffer switching
                    break;
                default:
                    // Unknown private mode
                    break;
            }
        }
    }
    
    processSGR(params) {
        if (params.length === 0) {
            params = [0];
        }
        
        for (let i = 0; i < params.length; i++) {
            const param = params[i];
            
            switch (param) {
                case 0: // Reset
                    this.reset();
                    break;
                case 1: // Bold
                    this.bold = true;
                    break;
                case 2: // Dim
                    this.dim = true;
                    break;
                case 3: // Italic
                    this.italic = true;
                    break;
                case 4: // Underline
                    this.underline = true;
                    break;
                case 5: // Blink (slow)
                case 6: // Blink (fast)
                    this.blink = true;
                    break;
                case 7: // Reverse
                    this.reverse = true;
                    break;
                case 8: // Hidden
                    this.hidden = true;
                    break;
                case 9: // Strikethrough
                    this.strikethrough = true;
                    break;
                case 22: // Normal intensity (turn off bold/dim)
                    this.bold = false;
                    this.dim = false;
                    break;
                case 23: // Not italic
                    this.italic = false;
                    break;
                case 24: // Not underlined
                    this.underline = false;
                    break;
                case 25: // Not blinking
                    this.blink = false;
                    break;
                case 27: // Not reversed
                    this.reverse = false;
                    break;
                case 28: // Not hidden
                    this.hidden = false;
                    break;
                case 29: // Not strikethrough
                    this.strikethrough = false;
                    break;
                    
                // Foreground colors
                case 30: this.currentForeground = ANSI_GREEN_COLORS.black; break;
                case 31: this.currentForeground = ANSI_GREEN_COLORS.red; break;
                case 32: this.currentForeground = ANSI_GREEN_COLORS.green; break;
                case 33: this.currentForeground = ANSI_GREEN_COLORS.yellow; break;
                case 34: this.currentForeground = ANSI_GREEN_COLORS.blue; break;
                case 35: this.currentForeground = ANSI_GREEN_COLORS.magenta; break;
                case 36: this.currentForeground = ANSI_GREEN_COLORS.cyan; break;
                case 37: this.currentForeground = ANSI_GREEN_COLORS.white; break;
                case 39: this.currentForeground = ANSI_GREEN_COLORS.white; break; // Default
                
                // Background colors
                case 40: this.currentBackground = ANSI_GREEN_COLORS.bgBlack; break;
                case 41: this.currentBackground = ANSI_GREEN_COLORS.bgRed; break;
                case 42: this.currentBackground = ANSI_GREEN_COLORS.bgGreen; break;
                case 43: this.currentBackground = ANSI_GREEN_COLORS.bgYellow; break;
                case 44: this.currentBackground = ANSI_GREEN_COLORS.bgBlue; break;
                case 45: this.currentBackground = ANSI_GREEN_COLORS.bgMagenta; break;
                case 46: this.currentBackground = ANSI_GREEN_COLORS.bgCyan; break;
                case 47: this.currentBackground = ANSI_GREEN_COLORS.bgWhite; break;
                case 49: this.currentBackground = ANSI_GREEN_COLORS.bgBlack; break; // Default
                
                // Bright foreground colors
                case 90: this.currentForeground = ANSI_GREEN_COLORS.brightBlack; break;
                case 91: this.currentForeground = ANSI_GREEN_COLORS.brightRed; break;
                case 92: this.currentForeground = ANSI_GREEN_COLORS.brightGreen; break;
                case 93: this.currentForeground = ANSI_GREEN_COLORS.brightYellow; break;
                case 94: this.currentForeground = ANSI_GREEN_COLORS.brightBlue; break;
                case 95: this.currentForeground = ANSI_GREEN_COLORS.brightMagenta; break;
                case 96: this.currentForeground = ANSI_GREEN_COLORS.brightCyan; break;
                case 97: this.currentForeground = ANSI_GREEN_COLORS.brightWhite; break;
                
                // 256-color and RGB color support
                case 38: // Set foreground color
                    if (i + 1 < params.length) {
                        if (params[i + 1] === 5) { // 256-color
                            i += 2; // Skip next two parameters
                            if (i < params.length) {
                                this.currentForeground = this.convert256ToGreen(params[i]);
                            }
                        } else if (params[i + 1] === 2) { // RGB
                            i += 4; // Skip next four parameters
                            if (i < params.length) {
                                const r = params[i - 2] || 0;
                                const g = params[i - 1] || 0;
                                const b = params[i] || 0;
                                this.currentForeground = this.convertRGBToGreen(r, g, b);
                            }
                        }
                    }
                    break;
                case 48: // Set background color
                    if (i + 1 < params.length) {
                        if (params[i + 1] === 5) { // 256-color
                            i += 2;
                            if (i < params.length) {
                                this.currentBackground = this.convert256ToGreen(params[i], true);
                            }
                        } else if (params[i + 1] === 2) { // RGB
                            i += 4;
                            if (i < params.length) {
                                const r = params[i - 2] || 0;
                                const g = params[i - 1] || 0;
                                const b = params[i] || 0;
                                this.currentBackground = this.convertRGBToGreen(r, g, b, true);
                            }
                        }
                    }
                    break;
            }
        }
    }
    
    // Convert 256-color index to green tone
    convert256ToGreen(colorIndex, isBackground = false) {
        // Vereinfachte Konvertierung: verwende den Index um eine Grünschattierung zu bestimmen
        const intensity = Math.floor((colorIndex % 256) / 255 * 15);
        const greenValue = Math.max(0, Math.min(255, intensity * 17));
        const baseColor = isBackground ? '#000000' : '#00ff00';
        
        if (isBackground) {
            return `rgb(0, ${Math.floor(greenValue / 4)}, 0)`;
        } else {
            return `rgb(0, ${greenValue}, ${Math.floor(greenValue / 2)})`;
        }
    }
    
    // Convert RGB to green tone
    convertRGBToGreen(r, g, b, isBackground = false) {
        // Konvertiere RGB zu Graustufen und dann zu Grüntönen
        const gray = Math.floor(0.299 * r + 0.587 * g + 0.114 * b);
        
        if (isBackground) {
            return `rgb(0, ${Math.floor(gray / 4)}, 0)`;
        } else {
            return `rgb(0, ${gray}, ${Math.floor(gray / 2)})`;
        }
    }
    
    getCurrentStyle() {
        let foreground = this.currentForeground;
        let background = this.currentBackground;
        
        // Handle reverse video
        if (this.reverse) {
            [foreground, background] = [background, foreground];
        }
        
        // Handle dim/bold
        if (this.bold) {
            // Brighten the foreground color
            foreground = this.brightenColor(foreground);
        }
        if (this.dim) {
            // Darken the foreground color
            foreground = this.darkenColor(foreground);
        }
          return {
            color: foreground,
            backgroundColor: background,
            fontWeight: this.bold ? 'bold' : 'normal',
            fontStyle: this.italic ? 'italic' : 'normal',
            textDecoration: [
                this.underline ? 'underline' : '',
                this.strikethrough ? 'line-through' : ''
            ].filter(d => d).join(' ') || 'none',
            opacity: this.hidden ? 0 : 1,
            animation: this.blink ? 'ansi-blink 1s infinite' : 'none',
            cursorVisible: this.cursorVisible
        };
    }
    
    brightenColor(color) {
        // Einfache Helligkeitsanpassung für Grüntöne
        const match = color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (match) {
            const r = Math.min(255, parseInt(match[1]) + 40);
            const g = Math.min(255, parseInt(match[2]) + 40);
            const b = Math.min(255, parseInt(match[3]) + 20);
            return `rgb(${r}, ${g}, ${b})`;
        }
        return color;
    }
    
    darkenColor(color) {
        // Einfache Dunkelanpassung für Grüntöne
        const match = color.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/);
        if (match) {
            const r = Math.max(0, parseInt(match[1]) - 20);
            const g = Math.max(0, parseInt(match[2]) - 20);
            const b = Math.max(0, parseInt(match[3]) - 10);
            return `rgb(${r}, ${g}, ${b})`;
        }
        return color;
    }
      // Extract terminal control commands (clear screen, cursor movement, etc.)
    extractTerminalCommands(text, commands) {
        // Enhanced regex to capture private mode sequences
        const ansiRegex = /\x1b\[(\??)([0-9;]*?)([A-Za-z])/g;
        let match;
        
        while ((match = ansiRegex.exec(text)) !== null) {
            const isPrivate = match[1] === '?';
            let params = match[2] ? match[2].split(';').map(p => parseInt(p) || 0) : [0];
            const command = match[3];
            
            switch (command) {
                case 'H': // Cursor Home
                case 'f': // Cursor Position
                    const row = (params[0] || 1) - 1; // Convert to 0-based
                    const col = (params[1] || 1) - 1; // Convert to 0-based
                    commands.push({
                        type: 'cursor',
                        action: 'position',
                        row: row,
                        col: col
                    });
                    break;
                    
                case 'J': // Clear Display
                    const clearType = params[0] || 0;
                    if (clearType === 0) {
                        commands.push({ type: 'clear', action: 'toEnd' });
                    } else if (clearType === 1) {
                        commands.push({ type: 'clear', action: 'toStart' });
                    } else if (clearType === 2) {
                        commands.push({ type: 'clear', action: 'all' });
                    }
                    break;
                    
                case 'K': // Clear Line
                    const lineType = params[0] || 0;
                    if (lineType === 0) {
                        commands.push({ type: 'clearLine', action: 'toEnd' });
                    } else if (lineType === 1) {
                        commands.push({ type: 'clearLine', action: 'toStart' });
                    } else if (lineType === 2) {
                        commands.push({ type: 'clearLine', action: 'all' });
                    }
                    break;
                    
                case 'A': // Cursor Up
                    commands.push({ type: 'cursor', action: 'up', count: params[0] || 1 });
                    break;
                case 'B': // Cursor Down
                    commands.push({ type: 'cursor', action: 'down', count: params[0] || 1 });
                    break;
                case 'C': // Cursor Forward
                    commands.push({ type: 'cursor', action: 'right', count: params[0] || 1 });
                    break;
                case 'D': // Cursor Back
                    commands.push({ type: 'cursor', action: 'left', count: params[0] || 1 });
                    break;
                    
                case 'h': // Set Mode
                case 'l': // Reset Mode
                    if (isPrivate) {
                        // Handle private mode sequences
                        for (const param of params) {
                            switch (param) {
                                case 25: // DECTCEM - Cursor visibility
                                    commands.push({
                                        type: 'cursor',
                                        action: 'visibility',
                                        visible: command === 'h'
                                    });
                                    break;
                                case 1: // DECCKM - Cursor Keys Mode
                                    commands.push({
                                        type: 'mode',
                                        action: 'cursorKeys',
                                        application: command === 'h'
                                    });
                                    break;
                                case 7: // DECAWM - Auto Wrap Mode
                                    commands.push({
                                        type: 'mode',
                                        action: 'autoWrap',
                                        enabled: command === 'h'
                                    });
                                    break;
                                case 47: // Alternate Screen
                                case 1047:
                                case 1049:
                                    commands.push({
                                        type: 'screen',
                                        action: 'alternate',
                                        enabled: command === 'h'
                                    });
                                    break;
                            }
                        }
                    }
                    break;
                    
                case 's': // Save Cursor Position
                    commands.push({ type: 'cursor', action: 'save' });
                    break;
                case 'u': // Restore Cursor Position
                    commands.push({ type: 'cursor', action: 'restore' });
                    break;
            }
        }
    }
    
    // Handle xterm-specific terminal queries and capability requests
    handleTerminalQuery(sequence, terminalHandler) {

        
        // Cursor Position Report (CPR) - ESC[6n
        if (sequence.includes('6n')) {
            const row = (terminalHandler.getCurrentRow?.() || 0) + 1; // Convert to 1-based
            const col = (terminalHandler.getCurrentCol?.() || 0) + 1; // Convert to 1-based
            const response = `\x1b[${row};${col}R`;

            terminalHandler.sendResponse?.(response);
            return true;
        }        // Device Attributes (DA) - ESC[0c or ESC[c
        if (sequence.includes('0c') || sequence === '\x1b[c') {
            // Standard xterm response: VT100 compatible with advanced video option
            // ?1 = VT100 compatible, ?2 = advanced video option (ANSI color)
            const response = '\x1b[?1;2c';

            terminalHandler.sendResponse?.(response);
            return true;
        }
          // Secondary Device Attributes - ESC[>0c
        if (sequence.includes('>0c') || sequence.includes('>c')) {
            // Standard xterm response: Terminal type 0 (VT100), version 95, no ROM cartridge
            // This matches common xterm implementations
            const response = '\x1b[>0;95;0c';

            terminalHandler.sendResponse?.(response);
            return true;
        }
        
        // Terminal Parameters - ESC[x
        if (sequence.includes('x')) {
            // Terminal answerback - just confirm
            const response = '\x1b[2;1;1;120;120;1;0x';

            terminalHandler.sendResponse?.(response);
            return true;
        }
        
        // Request Terminal Name - ESC[>q (XTVERSION)
        if (sequence.includes('>q')) {
            const response = '\x1bP>|ANSI Terminal\x1b\\';

            terminalHandler.sendResponse?.(response);
            return true;
        }
          // DECSLRM queries and other extended sequences
        if (sequence.includes('?')) {
            // Handle various DEC private mode queries
            if (sequence.includes('?25h')) {
                // Show cursor
                this.cursorVisible = true;
                return true;
            }
            if (sequence.includes('?25l')) {
                // Hide cursor
                this.cursorVisible = false;
                return true;
            }
            if (sequence.includes('?1049h') || sequence.includes('?1049l')) {
                // Alternate screen buffer - acknowledge
                return true;
            }
            if (sequence.includes('?1h') || sequence.includes('?1l')) {
                // Application cursor keys - acknowledge
                return true;
            }
            if (sequence.includes('?47h') || sequence.includes('?47l')) {
                // Alternate screen - acknowledge
                return true;
            }
            if (sequence.includes('?7h') || sequence.includes('?7l')) {
                // Auto wrap mode - acknowledge
                return true;
            }
        }
        
        // Operating System Command (OSC) - ESC]...BEL or ESC]...ST
        if (sequence.includes('\x1b]')) {
            // Handle title setting and other OSC commands

            return true;
        }
        
        return false;
    }
      // Extract plain text without ANSI sequences
    extractPlainText(text) {
        // Enhanced regex to remove all ANSI sequences including private modes
        return text.replace(/\x1b\[(\??)([0-9;]*?)([A-Za-z@`])/g, '')
                  .replace(/\x1b\[[0-9;]*[A-Za-z@`]/g, '')
                  .replace(/\x1b\][^\x07\x1b]*(\x07|\x1b\\)/g, '') // OSC sequences
                  .replace(/\x1b[()][AB012]/g, '') // Character set selection
                  .replace(/\x1b[78]/g, ''); // Save/restore cursor (short form)
    }
    
    // Get current cursor visibility state
    isCursorVisible() {
        return this.cursorVisible;
    }
    
    // Get current parser state for debugging
    getState() {
        return {
            foreground: this.currentForeground,
            background: this.currentBackground,
            bold: this.bold,
            dim: this.dim,
            italic: this.italic,
            underline: this.underline,
            blink: this.blink,
            reverse: this.reverse,
            hidden: this.hidden,
            strikethrough: this.strikethrough,
            cursorVisible: this.cursorVisible
        };
    }
    
    // Get information about supported ANSI sequences
    static getSupportedSequences() {
        return {
            cursor: {
                'ESC[?25h': 'Show cursor (DECTCEM)',
                'ESC[?25l': 'Hide cursor (DECTCEM)',
                'ESC[H': 'Cursor home position',
                'ESC[row;colH': 'Set cursor position',
                'ESC[A': 'Cursor up',
                'ESC[B': 'Cursor down', 
                'ESC[C': 'Cursor right',
                'ESC[D': 'Cursor left',
                'ESC[s': 'Save cursor position',
                'ESC[u': 'Restore cursor position',
                'ESC[G': 'Cursor horizontal absolute',
                'ESC[d': 'Line position absolute'
            },
            erase: {
                'ESC[J': 'Clear display (0=to end, 1=to start, 2=all)',
                'ESC[K': 'Clear line (0=to end, 1=to start, 2=all)',
                'ESC[X': 'Erase characters'
            },
            text: {
                'ESC[m': 'Reset all text attributes',
                'ESC[1m': 'Bold text',
                'ESC[3m': 'Italic text',
                'ESC[4m': 'Underline text',
                'ESC[5m': 'Blinking text',
                'ESC[7m': 'Reverse video',
                'ESC[30-37m': 'Foreground colors',
                'ESC[40-47m': 'Background colors',
                'ESC[90-97m': 'Bright foreground colors'
            },
            modes: {
                'ESC[?1h/l': 'Application cursor keys mode',
                'ESC[?7h/l': 'Auto wrap mode',
                'ESC[?47h/l': 'Alternate screen buffer',
                'ESC[?1047h/l': 'Alternate screen buffer (xterm)',
                'ESC[?1049h/l': 'Alternate screen buffer with save/restore'
            },
            scroll: {
                'ESC[S': 'Scroll up',
                'ESC[T': 'Scroll down',
                'ESC[L': 'Insert lines',
                'ESC[M': 'Delete lines'
            },
            queries: {
                'ESC[6n': 'Cursor position report',
                'ESC[c': 'Device attributes',
                'ESC[>c': 'Secondary device attributes',
                'ESC[n': 'Device status report'
            }
        };
    }
}

// Global ANSIParser class available
window.ANSIParser = ANSIParser;

