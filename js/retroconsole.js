/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// retroconsole.js – Text-Rendering und Cursor für das Retro-Terminal
const CFG = window.CRT_CONFIG;
if (!CFG) {
    throw new Error("CRT_CONFIG is not defined on window. Please define window.CRT_CONFIG before loading retroconsole.js");
}
// Check for required CRT_CONFIG properties
const requiredCfgKeys = [
    "SCREEN_PADDING_LEFT", "SCREEN_PADDING_TOP", "SCREEN_PADDING_RIGHT", "SCREEN_PADDING_BOTTOM",
    "VIRTUAL_CRT_WIDTH", "VIRTUAL_CRT_HEIGHT", "FONT_SIZE_PX", "FONT_FAMILY", "TEXT_ROWS", "TEXT_COLS"
];
for (const key of requiredCfgKeys) {
    if (typeof CFG[key] === "undefined") {
        throw new Error(`CRT_CONFIG is missing required property: ${key}`);
    }
}
// Zuordnungstabelle für Backend-Response-Typen
const RESPONSE_TYPE_MAP = {
    0: 'TEXT',          // Textausgabe
    1: 'CLEAR',         // Bildschirm löschen
    2: 'BEEP',          // Beep-Ton
    3: 'SPEAK',         // Sprachausgabe
    4: 'GRAPHICS',      // Grafikbefehl
    5: 'SOUND',         // Soundkommando
    6: 'SPEAK_DONE',    // Sprachausgabe beendet (SAY_DONE)
    7: 'MODE',          // Moduswechsel
    8: 'SESSION',       // Session-ID
    9: 'INPUT_CONTROL', // Eingabesteuerung
    10: 'SPRITE',       // Sprite-Befehl (neuer Kanal)
    11: 'VECTOR',       // Vektorgrafik-Befehl (neuer Kanal)
    12: 'PROMPT',       // Prompt anzeigen
    13: 'NOISE',        // Noise-Befehl
    14: 'INPUT',        // Eingabezeile aktualisieren
    15: 'CHAT',         // Chat-Modus aktivieren
    16: 'KEY_DOWN',     // Taste gedrückt (für INKEY$)
    17: 'KEY_UP',       // Taste losgelassen (für INKEY$)
    18: 'LOCATE',       // Cursor-Position setzen (LOCATE x,y)
    19: 'INVERSE',      // Inverser Text-Modus (INVERSE ON/OFF)
    20: 'EDITOR',       // Editor-Modus
    21: 'PAGER',        // Pager-Modus    22: 'CURSOR',       // Cursor-Steuerung (show/hide)
    23: 'TELNET',       // Telnet-Modus
    24: 'AUTO_EXECUTE', // Automatische Eingabe-Ausführung (autorun)
    25: 'BITMAP',       // Bitmap-Übertragung (PNG) mit Platzierung/Skalierung/Rotation
    26: 'EVIL',         // Evil effect - dramatic noise increase for MCP
    27: 'AUTH_REFRESH', // Auth token refresh required
    28: 'IMAGE',        // Image commands (LOAD, SHOW, HIDE, ROTATE)
    29: 'PARTICLE'      // Particle system commands
};

// Zentrales RetroConsole-Objekt global anlegen, falls noch nicht vorhanden
if (!window.RetroConsole) window.RetroConsole = {};

// Eigenschaften und Methoden an das globale RetroConsole anhängen
Object.assign(window.RetroConsole, {
    lines: ["SKYNET COMPUTERS INC.", "TERMINAL READY", ""],
    // Paralleles Array zu lines: Für jede Zeile ein Array von Booleans (true=invers)
    inverseLines: [[], [], []],
    promptSymbol: "> ",
    input: "",
    cursorPos: 0,    inputEnabled: true, // Standard: Eingabe aktiviert
    inputMode: 0,       // 0: OS_SHELL, 1: BASIC, 2: CHESS, 3: EDITOR, 4: TELNET, 5: PAGER
    
    // Command history for up/down arrow navigation
    commandHistory: [],
    historyIndex: -1,    // -1 means no history selection, 0+ means history position
    maxHistorySize: 30,
    runMode: false,     // RUN-Modus (INKEY$/Strg+C aktiv, normale Eingabe deaktiviert)
    passwordMode: false, // Passwort-Modus (Eingabe wird als * angezeigt)
    pagerMode: false,   // Pager-Modus (Einzeltasten-Eingabe für CAT-Pager)
    terminalStatus: "", // Status line for terminal mode (like pager status)
    // Telnet-Support
    telnetMode: false,  // Telnet mode activated
    pagerPrompt: "",    // Prompt for pager mode (e.g., cat)
    telnetEchoBuffer: [], // Buffer for tracking sent characters to suppress echo {char, timestamp}
    telnetServerEcho: false, // Whether server handles echoing (prevents double characters)
    chatMode: false,    // Chat mode activated
      // LOCATE-Support
    cursorX: 0,         // Aktuelle Cursor-X-Position (0-basiert)
    cursorY: 0,         // Aktuelle Cursor-Y-Position (0-basiert)
    locateMode: false,  // Ob LOCATE aktiv ist
    
    // 2D-Array für positioned text (für LOCATE)
    screenBuffer: null, // Wird dynamisch erstellt: [y][x] = {char, inverse}
    
    // INVERSE-Support
    inverseMode: false, // Inverser Text-Modus aktiviert
      textCanvas: null,
    _textCanvasDirty: true, // Flag to indicate if the text canvas needs redrawing
    _cursorBlinkInterval: null, // Interval ID for cursor blinking
    _lastCursorState: false, // Track the last cursor blink state
    isTextCanvasDirty: function() {
        // Redraw if content is dirty OR if the cursor's blink state has changed.
        const currentCursorState = this.inputEnabled && !this.forcedHideCursor && (Math.floor(Date.now() / 500) % 2 === 0);
        return this._textCanvasDirty || (currentCursorState !== this._lastCursorState);
    },
    markTextCanvasClean: function() {
        this._textCanvasDirty = false;
        // Also update the last known cursor state
        this._lastCursorState = this.inputEnabled && !this.forcedHideCursor && (Math.floor(Date.now() / 500) % 2 === 0);
    },
    textTexture: null,
    startCursorBlink: function() {
        if (this._cursorBlinkInterval === null) {
            this._cursorBlinkInterval = setInterval(() => {
                if (this.inputEnabled && !this.editorMode && !this.filenameInputMode) {
                    this.drawTerminal();
                }
            }, 500); // Blink every 500ms
        }
    },
    stopCursorBlink: function() {
        if (this._cursorBlinkInterval !== null) {
            clearInterval(this._cursorBlinkInterval);
            this._cursorBlinkInterval = null;
            // Ensure cursor is hidden when blinking stops
            this.drawTerminal();
        }
    },
    CHAR_WIDTH: 10,
    CHAR_HEIGHT: 20,
    debugMode: false, // Debug mode enabled for chess debugging    // Editor-Variablen
    editorMode: false,
    editorLines: [],
    editorCursorLine: 0,      // Relative Cursor-Position im sichtbaren Bereich
    editorCursorCol: 0,
    editorScrollLine: 0,      // Scroll-Offset (erste sichtbare Zeile)
    editorAbsCursorLine: 0,   // Absolute Cursor-Position im Dokument
    editorFilename: "",
    editorModified: false,
    editorStatus: "",
    editorRows: 24,
    editorCols: 80,
    editorData: null, // Hält alle relevanten Editor-Daten vom Backend

    // Chess help text to be shown above the input line in chess mode
    chessHelpText: "",
    
    // Filename input mode variables
    filenameInputMode: false,
    filenameInput: "",
    filenamePrompt: "",
    editorStatus: "",
    editorRows: 24,
    editorCols: 80,
    editorData: null, // Hält alle relevanten Editor-Daten vom Backend
    terminalLinesBackup: [], // Zum Speichern der Terminal-Lines vor dem Editor-Start
    terminalInverseLinesBackup: [], // Zum Speichern der inversen Terminal-Lines
    terminalScreenBufferBackup: null, // Zum Speichern des ScreenBuffers

    // Initialize RetroConsole's own textCanvas and textTexture
    initTextCanvas: function() {
        if (this.textCanvas) {

            return;
        }
        
        // Create textCanvas
        this.textCanvas = document.createElement('canvas');
        this.textCanvas.width = CFG.VIRTUAL_CRT_WIDTH;
        this.textCanvas.height = CFG.VIRTUAL_CRT_HEIGHT;
        
        // Disable image smoothing
        const ctx = this.textCanvas.getContext('2d');
        if (ctx) {
            ctx.imageSmoothingEnabled = false;
        }
          // Create THREE.js texture from canvas
        if (typeof THREE !== 'undefined') {
            this.textTexture = new THREE.CanvasTexture(this.textCanvas);
            this.textTexture.minFilter = THREE.NearestFilter;
            this.textTexture.magFilter = THREE.NearestFilter;
            this.textTexture.flipY = false; // Verhindert vertikale Spiegelung der Canvas-Textur        } else {

        }        
        // Update character metrics
        this.updateCharMetrics();
    },
    
    // Global function for automatic line wrapping
    wrapText: function(text, maxLength) {
        if (!text) {
            return [""];
        }
        
        // First, split by existing line breaks (\n)
        var existingLines = text.split('\n');
        var wrappedLines = [];
        
        for (var i = 0; i < existingLines.length; i++) {
            var line = existingLines[i];
            if (line.length <= maxLength) {
                wrappedLines.push(line);
            } else {
                // Line is too long, need to wrap it
                var remaining = line;
                
                while (remaining.length > 0) {
                    if (remaining.length <= maxLength) {
                        wrappedLines.push(remaining);
                        break;
                    }
                    
                    // Try to break at word boundaries when possible
                    var cutPos = maxLength;
                    var spacePos = remaining.lastIndexOf(' ', maxLength);
                    
                    // If a space is found in the first 75% of the line, break there
                    if (spacePos > maxLength * 0.75) {
                        cutPos = spacePos;
                        // Don't skip the space, preserve it in the next line
                        wrappedLines.push(remaining.substring(0, cutPos));
                        remaining = remaining.substring(cutPos + 1); // +1 to skip only the space at break point
                    } else {
                        // Hard break at maxLength, don't remove any characters
                        wrappedLines.push(remaining.substring(0, cutPos));
                        remaining = remaining.substring(cutPos);
                    }
                }
            }
        }
        
        return wrappedLines;
    },
    
    updateCharMetrics: function() {
        const CFG = window.CRT_CONFIG;

        // Horizontales Padding:
        this.CHAR_PADDING_X = CFG.SCREEN_PADDING_LEFT;
        const horizontalPaddingTotal = CFG.SCREEN_PADDING_LEFT + CFG.SCREEN_PADDING_RIGHT;

        // Vertikales Padding:
        this.CHAR_PADDING_Y = CFG.SCREEN_PADDING_TOP;
        const verticalPaddingTotal = CFG.SCREEN_PADDING_TOP + CFG.SCREEN_PADDING_BOTTOM;

        // Berechne den verfügbaren Platz nach Abzug des Paddings
        const availableWidth = CFG.VIRTUAL_CRT_WIDTH - horizontalPaddingTotal;
        const availableHeight = CFG.VIRTUAL_CRT_HEIGHT - verticalPaddingTotal;

        // Berechne die finalen Zeichendimensionen basierend auf dem verfügbaren Platz
        // CHAR_WIDTH (OHNE Math.floor für diesen Test)
        this.CHAR_WIDTH = availableWidth / CFG.TEXT_COLS; 
        // cursorAdvanceRate (präzise) für Cursor-Positionierung und -Breite
        this.cursorAdvanceRate = availableWidth / CFG.TEXT_COLS;

        this.CHAR_HEIGHT = availableHeight / CFG.TEXT_ROWS;
        this.CHAR_FONT_SIZE = CFG.FONT_SIZE_PX; // FONT_SIZE_PX bleibt als Referenz für die Schriftgröße
    },
    
    // Diese Funktion setzt das textCanvas und textTexture aus dem retrographics.js-Modul
    setCanvasAndTexture: function(canvas, texture) {
        if (!canvas || !texture) {

            return;
        }
        
        this.textCanvas = canvas;
        this.textTexture = texture;

        
        // NEU: imageSmoothingEnabled für den Text-Kontext deaktivieren
        if (this.textCanvas) {
            const tempCtx = this.textCanvas.getContext('2d');
            if (tempCtx) {
                tempCtx.imageSmoothingEnabled = false;

            }
        }

        // Zeichenmetriken aktualisieren, bevor wir das Terminal zeichnen
        this.updateCharMetrics();
        
        // Initial einmal zeichnen, um sicherzustellen, dass die Textur Inhalt hat        this.drawTerminal(); // Dieser Aufruf ist wichtig für den initialen Text
    },
    
    checkAndScroll: function() {
        const maxVisibleRows = CFG.TEXT_ROWS;
        const totalLines = this.lines.length;
        
        // Prüfe, ob die Anzahl der Zeilen den sichtbaren Bereich überschreitet
        if (totalLines >= maxVisibleRows) {
            const scrollLines = totalLines - maxVisibleRows + 1; // +1 für die aktuelle Eingabezeile
            // Debug-Log entfernt für bessere Performance
            
            // Entferne die obersten Zeilen
            this.lines.splice(0, scrollLines);
            
            // BUGFIX: Auch inverseLines entsprechend scrollen
            this.inverseLines.splice(0, scrollLines);

            // BUGFIX: ScreenBuffer (LOCATE-Text) auch scrollen
            if (this.screenBuffer) {
                this.screenBuffer.splice(0, scrollLines);
                // Leere Zeilen am Ende hinzufügen für neuen Platz
                for (let i = 0; i < scrollLines; i++) {
                    this.screenBuffer.push(Array(CFG.TEXT_COLS).fill({char: ' ', inverse: false}));
                }
            }
        }
    },

    drawTerminal: function() {
        // WICHTIG: Im Editor-Modus nicht das Terminal zeichnen!
        if (this.editorMode) {
            return;
        }
        
        // Debug: Wer ruft drawTerminal auf?
        
        if (!this.textCanvas) {

            return;
        }
        const ctx = this.textCanvas.getContext('2d');
        if (!ctx) {

            return;
        }

        // NEU: Sicherstellen, dass imageSmoothingEnabled auch hier deaktiviert ist
        ctx.imageSmoothingEnabled = false;

        // Hintergrund des Text-Canvas schwarz füllen (für alle Modi)
        ctx.fillStyle = '#000000';
        ctx.fillRect(0, 0, this.textCanvas.width, this.textCanvas.height);
        
        // Zeichenbereich - verwende berechnete Padding-Werte
        const padL = CFG.SCREEN_PADDING_LEFT; // Direkte Verwendung aus CFG
        const padT = CFG.SCREEN_PADDING_TOP;   // Direkte Verwendung aus CFG

        // Entferne gerundete Padding-Werte, verwende padL, padT direkt
        // const rPadL = Math.round(padL);
        // const rPadT = Math.round(padT);

        ctx.font = `${this.CHAR_FONT_SIZE}px ${CFG.FONT_FAMILY}`;
        // Die Textfarbe entspricht der hellsten Stufe in der Helligkeitsskala (Index 15)
        ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Textfarbe nach fillRect wieder setzen
        ctx.textBaseline = 'top';

        // WICHTIG: Messe die tatsächliche Zeichenbreite der Schrift
        // const actualCharWidth = ctx.measureText('M').width; // 'M' ist normalerweise das breiteste Zeichen
        // Statt 'M' verwenden wir einen Durchschnitt oder einen repräsentativen Charakter,
        // oder verlassen uns auf die berechnete this.CHAR_WIDTH, wenn die Schriftart wirklich monospaced ist.
        // Für den Moment verwenden wir die berechnete Breite, da FONT_FAMILY="monospace" gesetzt ist.
        this.ACTUAL_CHAR_WIDTH = this.CHAR_WIDTH; // Annahme: Monospace-Schrift
        
        // Text zeichnen - zuerst normale Lines, dann Screen Buffer für LOCATE
        // Im Telnet-Modus: eine Zeile für Statuszeile reservieren
        // Im Normal-Modus: ebenfalls eine Zeile für Statuszeile reservieren um Overflow zu vermeiden
        const availableRows = CFG.TEXT_ROWS - 1;
        
        // Ensure lines arrays are initialized
        if (!this.lines) this.lines = [];
        if (!this.inverseLines) this.inverseLines = [];
        
        // Initialize empty arrays if needed
        if (this.lines.length === 0) {
            // No default content needed - let the backend provide initial messages
        }
        
        // Declare variables outside the conditional block for later use
        let visibleLines = this.lines.slice(-availableRows);
        let visibleInverse = this.inverseLines.slice(-availableRows);
        
        // In chess mode, suppress normal text lines to allow LOCATE-positioned text to show
        if (this.inputMode !== 2 /* Chess */) {
            // Render normal text lines (non-chess modes or debug info)
            for (let i = 0; i < visibleLines.length; i++) {
            let line = visibleLines[i];
            let invArr = visibleInverse[i] || [];
            for (let col = 0; col < Math.min(line.length, CFG.TEXT_COLS); col++) {
                const char = line[col];
                const inverse = invArr[col] === true;
                const x = padL + col * this.CHAR_WIDTH;
                const y = padT + i * this.CHAR_HEIGHT;
                
                if (inverse) {
                    ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
                    // Inverser Hintergrund: 2 Pixel vor Oberkante, 2 Pixel unter Grundlinie
                    const bgY = y - 2; // 2 Pixel vor der Oberkante des Textes
                    const bgHeight = this.CHAR_HEIGHT - 2; // Text-Höhe minus 2 Pixel (endet 2 Pixel vor nächster Linie)
                    ctx.fillRect(x, bgY, this.CHAR_WIDTH, bgHeight);
                    ctx.fillStyle = '#000000';
                } else {
                    ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
                }
                ctx.fillText(char, x, y);
            }
        }
        } // End of if (this.inputMode !== 2) block
        // Screen Buffer rendern (LOCATE-Text überlagert normale Lines)
        // Sowohl im Telnet- als auch Normal-Modus: auf verfügbare Zeilen begrenzen
        // Im Schachmodus: screenBuffer wird separat in der Chess-Eingabelogik verarbeitet, aber Titel wird hier gerendert
        if (this.screenBuffer) {
            const maxBufferRows = CFG.TEXT_ROWS - 1;
            for (let row = 0; row < Math.min(this.screenBuffer.length, maxBufferRows); row++) {
                for (let col = 0; col < Math.min(this.screenBuffer[row].length, CFG.TEXT_COLS); col++) {
                    const cell = this.screenBuffer[row][col];
                    if (cell.char !== ' ') { // Nur nicht-leere Zeichen rendern
                        const x = padL + col * this.CHAR_WIDTH;
                        const y = padT + row * this.CHAR_HEIGHT;
                        
                        // In chess mode: show all screenBuffer content
                        // This includes status text, help text, and positioned content
                        
                        if (cell.inverse) {
                            // Inverser Text: schwarzer Text auf hellem Hintergrund
                            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Heller Hintergrund
                            // Inverser Hintergrund: 2 Pixel vor Oberkante, 2 Pixel unter Grundlinie
                            const bgY = y - 2; // 2 Pixel vor der Oberkante des Textes
                            const bgHeight = this.CHAR_HEIGHT - 2; // Text-Höhe minus 2 Pixel (endet 2 Pixel vor nächster Linie)
                            ctx.fillRect(x, bgY, this.CHAR_WIDTH, bgHeight);
                            ctx.fillStyle = '#000000'; // Schwarzer Text
                        }else {
                            // Normaler Text
                            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
                        }
                        
                        ctx.fillText(cell.char, x, y);
                        // Textfarbe für nächstes Zeichen zurücksetzen
                        ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
                    }
                }
            }
        }        // Calculate inputStartRow for both telnet and normal modes
        // Input area should start directly after the last output line
        let inputStartRow;
        if (this.inputMode === 2 /* InputModeChess */) {
            // In chess mode: input starts at cursor position from backend
            inputStartRow = this.cursorY;
        } else {
            let visibleLines = this.lines.slice(-availableRows);
            inputStartRow = visibleLines.length;
        }
        
        // Eingabezeile - unterstützt jetzt auch mehrzeilige Eingabe
        // Im runMode verstecken wir den Prompt, aber zeigen weiterhin Eingaben an (z.B. INPUT-Befehle)
        // WICHTIG: Im Telnet-Modus wird KEINE lokale Eingabezeile gezeichnet (alles kommt vom Server)
        // Im Schachmodus wird die Eingabezeile vom Schach-UI selbst gezeichnet, aber der Cursor wird von hier verwaltet.
        if (!this.telnetMode && this.inputMode !== 2 /* InputModeChess */) {
            let displayInput = this.input;
            
            // Im Passwort-Modus Eingabe durch Sterne ersetzen
            if (this.passwordMode) {
                displayInput = '*'.repeat(this.input.length);
            }
            
            let inputLine;
            if (this.pagerMode) {
                // Im Pager-Modus keine normale Eingabezeile anzeigen
                inputLine = "";
            } else if (this.runMode) {
                inputLine = displayInput;
            } else {
                inputLine = this.promptSymbol + displayInput;
            }
            // Bei längeren Eingaben das Umbrechen in mehrere Zeilen unterstützen
            const maxLineLength = CFG.TEXT_COLS;
            const firstLineLength = Math.min(maxLineLength, inputLine.length);              // Berechne die Anzahl der Zeilen, die die Eingabe einnimmt
            const totalInputLines = Math.ceil(inputLine.length / maxLineLength) || 1;

            // Erste Zeile der Eingabe zeichnen
            ctx.fillText(
                inputLine.substring(0, firstLineLength),
                padL, // Start-X-Position für den Text (ungerundet)
                padT + inputStartRow * this.CHAR_HEIGHT // Y-Position ungerundet
            );

            // Falls nötig, weitere Zeilen der Eingabe zeichnen
            if (inputLine.length > maxLineLength) {
                const remainingText = inputLine.substring(maxLineLength);
                // In Segmenten von maxLineLength Zeichen aufteilen und zeichnen
                for (let i = 0; i < remainingText.length; i += maxLineLength) {
                    const lineIndex = inputStartRow + 1 + Math.floor(i / maxLineLength);                    const lineText = remainingText.substring(i, i + maxLineLength);
                    ctx.fillText(
                        lineText,
                        padL, // Start-X-Position für den Text (ungerundet)
                        padT + lineIndex * this.CHAR_HEIGHT // Y-Position ungerundet
                    );
                }
            }
        } else if (this.inputMode === 2 /* InputModeChess */) {
            // In chess mode: only draw input buffer at cursor position from backend
            // The cursor position comes from LOCATE messages sent by the backend
            // Draw the input buffer at the precise cursor position set by backend LOCATE
            let displayInput = this.input;
            
            // In password mode, replace input with stars
            if (this.passwordMode) {
                displayInput = '*'.repeat(this.input.length);
            }
            
            // Chess mode respects LOCATE positioning from backend
            const x = padL + this.cursorX * this.CHAR_WIDTH;
            const y = padT + this.cursorY * this.CHAR_HEIGHT;
            // console.log(`[CHESS-DEBUG] Drawing input at cursorX=${this.cursorX}, cursorY=${this.cursorY}, screenX=${x}, screenY=${y}, input="${displayInput}"`);
            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
            ctx.fillText(displayInput, x, y);
        }
        
        // Check if LOCATE text (screenBuffer) extends beyond normal text lines for both modes
        if (this.screenBuffer) {
            // Find the last row in screenBuffer that contains text
            let lastUsedRow = -1;
            for (let row = 0; row < this.screenBuffer.length; row++) {
                for (let col = 0; col < this.screenBuffer[row].length; col++) {
                    if (this.screenBuffer[row][col] && this.screenBuffer[row][col].char !== ' ') {
                        lastUsedRow = row;
                        break;
                    }
                }
            }
            
            // If LOCATE text extends beyond normal lines, adjust input start position
            if (lastUsedRow >= 0) {
                inputStartRow = Math.max(inputStartRow, lastUsedRow + 1);
            }
        }
        
        // CURSOR LOGIC FOR ALL MODES (including Telnet)
        // Cursor - ensure it's always visible with authentic 80s scrolling
        // Check for forcedHideCursor to explicitly hide cursor
        if (this.inputEnabled && !this.forcedHideCursor) {
            // Blinking cursor (on/off every 500ms)
            const showCursor = Math.floor(Date.now() / 500) % 2 === 0;
            
            // Calculate cursor position
            let cursorCol, cursorRow;
            
            if (this.telnetMode) {
                // In telnet mode: Use server-managed cursor position
                cursorCol = this.cursorX;
                cursorRow = this.cursorY;
            } else if (this.inputMode === 2 /* InputModeChess */) { // NEW CONDITION FOR CHESS MODE
                // In chess mode: cursor position is managed by backend LOCATE messages
                // Input follows the prompt, so cursor is at cursorX + input position
                cursorCol = this.cursorX + this.cursorPos;
                cursorRow = this.cursorY;
                // console.log(`[CHESS-DEBUG] Cursor position: cursorX=${this.cursorX}, cursorPos=${this.cursorPos}, cursorCol=${cursorCol}, cursorRow=${cursorRow}`);
            } else if (this.inputMode === 9 /* InputModeBoard */) { // BOARD MODE
                // In board mode: Similar to normal mode but with bounds checking and auto-scroll
                const promptLength = this.runMode ? 0 : this.promptSymbol.length;
                cursorCol = (promptLength + this.cursorPos) % CFG.TEXT_COLS;
                let calculatedRow = inputStartRow + Math.floor((promptLength + this.cursorPos) / CFG.TEXT_COLS);
                
                // Check if cursor would exceed screen bounds and auto-scroll if needed
                const maxVisibleRow = CFG.TEXT_ROWS - 1;
                if (calculatedRow >= maxVisibleRow) {
                    // Force scroll up to keep cursor visible
                    const excessRows = calculatedRow - maxVisibleRow + 1;
                    // Directly remove excess lines from the top
                    for (let i = 0; i < excessRows; i++) {
                        if (this.lines.length > 0) {
                            this.lines.shift();
                            if (this.inverseLines.length > 0) {
                                this.inverseLines.shift();
                            }
                        }
                    }
                    // Recalculate after scrolling
                    calculatedRow = maxVisibleRow;
                }
                cursorRow = calculatedRow;
            } else {
                // In normal mode: Calculate cursor position based on input
                const promptLength = this.runMode ? 0 : this.promptSymbol.length;
                cursorCol = (promptLength + this.cursorPos) % CFG.TEXT_COLS;
                cursorRow = inputStartRow + Math.floor((promptLength + this.cursorPos) / CFG.TEXT_COLS);
            }
            
            if (showCursor) {
                ctx.save();
                ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[12]; // Slightly darker green for cursor
                
                let finalCursorScreenX, finalCursorScreenY;
                
                if (this.telnetMode) {
                    // In telnet mode: Simple position calculation
                    finalCursorScreenX = padL + cursorCol * this.cursorAdvanceRate;
                    finalCursorScreenY = padT + cursorRow * this.CHAR_HEIGHT;
                } else if (this.inputMode === 2 /* InputModeChess */) {
                    // In chess mode: Cursor position is directly from this.cursorX, this.cursorY
                    finalCursorScreenX = padL + (this.cursorX + this.cursorPos) * this.CHAR_WIDTH;
                    finalCursorScreenY = padT + this.cursorY * this.CHAR_HEIGHT;
                } else {
                    // In normal mode: Precise text measurement
                    const absoluteCursorCharIndex = (this.runMode ? 0 : this.promptSymbol.length) + this.cursorPos;
                    const startOfCurrentInputLineIndex = Math.floor(absoluteCursorCharIndex / CFG.TEXT_COLS) * CFG.TEXT_COLS;
                    const textOnCurrentLineBeforeCursor = (this.runMode ? this.input : (this.promptSymbol + this.input)).substring(startOfCurrentInputLineIndex, absoluteCursorCharIndex);
                    const cursorTextMeasureX = ctx.measureText(textOnCurrentLineBeforeCursor).width;
                    finalCursorScreenX = padL + cursorTextMeasureX;
                    finalCursorScreenY = padT + cursorRow * this.CHAR_HEIGHT;
                }
                
                const cursorBlockWidth = this.cursorAdvanceRate * 0.9;
                
                ctx.fillRect(
                    finalCursorScreenX,
                    finalCursorScreenY,
                    cursorBlockWidth,
                    this.CHAR_HEIGHT * 0.85
                );

                ctx.fillStyle = '#041208';
                let charUnderCursor;
                if (this.telnetMode) {
                    charUnderCursor = ' '; // In telnet mode, just show space
                } else {
                    charUnderCursor = (this.cursorPos < this.input.length) ? this.input[this.cursorPos] : ' ';
                }
                
                ctx.fillText(
                    charUnderCursor,
                    finalCursorScreenX,
                    finalCursorScreenY
                );
                
                ctx.restore();
            }
        }
        
        // Robuste Text-Textur-Aktualisierung
        if (this.textTexture) {
            // Versuche zuerst die neue robuste Update-Funktion
            if (window.RetroGraphics && window.RetroGraphics.forceTextTextureUpdate) {
                window.RetroGraphics.forceTextTextureUpdate();
            }
            
            // Material neu erstellen für Shader-Updates - Dies sollte seltener passieren, nur wenn der Shader sich ändert.
            // Für reine Textur-Updates ist es nicht immer nötig und kann Performance kosten.
            // if (window.RetroGraphics && window.RetroGraphics.recreateMaterial) {
            //     window.RetroGraphics.recreateMaterial();
            // }
            
            // Fallback: Traditionelle Update-Methoden (sollten durch forceTextTextureUpdate abgedeckt sein)
            this.textTexture.needsUpdate = true;
        }
        
        // TELNET-SPEZIFISCHE ERGÄNZUNGEN: Statuszeile in der letzten Zeile
        if (this.telnetMode || this.terminalStatus) {
            this.drawTerminalStatusLine(ctx, padL, padT);
        }
    },

    // Draws the terminal status line in the last row (for pager mode etc.)
    drawTerminalStatusLine: function(ctx, padL, padT) {
        if (!this.terminalStatus) return;
        
        // Status line is always drawn in the last row (TEXT_ROWS - 1)
        const statusRow = CFG.TEXT_ROWS - 1;
        const statusY = padT + statusRow * this.CHAR_HEIGHT;
        
        // Status line background (inverted) - same as editor and telnet
        ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Bright background
        const bgY = statusY - 2; // 2 pixels before text top edge
        const bgHeight = this.CHAR_HEIGHT - 2; // Text height minus 2 pixels
        ctx.fillRect(padL, bgY, CFG.TEXT_COLS * this.CHAR_WIDTH, bgHeight);
        
        // Draw status text with black text on bright background
        ctx.fillStyle = '#000000'; // Black text for inverted display
        for (let col = 0; col < Math.min(this.terminalStatus.length, CFG.TEXT_COLS); col++) {
            const char = this.terminalStatus[col];
            const x = padL + col * this.CHAR_WIDTH;
            ctx.fillText(char, x, statusY);
        }
        
        // Reset text color for subsequent characters
        ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
    },
    
    // Backend-API-Handler for text output, input, clear etc.
    handleBackendMessage: function(event) {
        try {
            let raw = typeof event.data === 'string' ? event.data.trim() : event.data;
            // console.log('[RetroConsole-DEBUG] Received raw backend message:', raw);
            if (typeof raw === 'string' && raw.length === 0) {
                // console.log('[FRONTEND-DEBUG] Empty string data received, ignoring.');
                return;
            }
            
            const processAndRouteResponse = (responseObject) => {
                const typeName = RESPONSE_TYPE_MAP[responseObject.type] || 'UNKNOWN';
                
                // Debug: Log all chess-related messages
                if (this.inputMode === 2) {
                    // console.log(`[CHESS-DEBUG] Received message type ${responseObject.type} (${typeName}):`, responseObject);
                }

                if (typeName === 'EDITOR') {
                    this.handleEditorMessage(responseObject);
                    return false; // Editor message handled, potentially no drawTerminal needed from here
                }
                
                this._processBackendResponse(responseObject, typeName); // Pass typeName
                return true;  // Normal message, main handler might call drawTerminal
            };

            if (typeof raw === 'string') {
                const openBraces = (raw.match(/{/g) || []).length;
                const closeBraces = (raw.match(/}/g) || []).length;
                
                if (openBraces > 1 && closeBraces > 1 && raw.match(/}\s*{/)) {
                    let parts = raw.split(/}\s*{/).map((part, idx, arr) => {
                        if (idx === 0) return part + '}';
                        if (idx === arr.length - 1) return '{' + part;
                        return '{' + part + '}';
                    });
                    let shouldDrawTerminalAfterAllParts = true;
                    parts.forEach((jsonStr) => {
                        try {
                            const responseObject = JSON.parse(jsonStr);
                            if (!processAndRouteResponse(responseObject)) {
                                shouldDrawTerminalAfterAllParts = false;
                            }
                        } catch (e) {
                            // console.error('[RetroConsole-ERROR] Error parsing part of multi-JSON:', e, 'Part:', jsonStr);
                        }
                    });
                    if (shouldDrawTerminalAfterAllParts) {
                        this.drawTerminal();
                    }
                } else { // Single JSON string
                    try {
                        const responseObject = JSON.parse(raw);
                        if (processAndRouteResponse(responseObject)) {
                            this.drawTerminal();
                        }
                    } catch (e) {
                        // console.error('[RetroConsole-ERROR] Error parsing single JSON string:', e, 'String:', raw);
                    }
                }
            } else if (typeof raw === 'object' && raw !== null) { // raw is already an object
                if (processAndRouteResponse(raw)) { // raw is the responseObject
                    this.drawTerminal();
                }
            } else {
                // console.warn('[RetroConsole-RAW] Received data is not a string or object:', raw);
            }
        } catch (e) {
            // console.error('[RetroConsole-ERROR] General error in handleBackendMessage:', e, 'Raw data:', event.data);
        }
    },
    
    _processBackendResponse: function(response, typeName) { // typeName is now an argument
        // This switch handles all other types.
        switch (typeName) {
            case 'TEXT':
                // Special handling: If chessHelp flag is set, store as chess help text
                if (response.chessHelp === true && response.content !== undefined) {
                    this.chessHelpText = String(response.content);
                    return;
                }
                // Process TEXT messages directly in telnet mode
                if (response.content !== undefined && response.content !== null) {
                    const contentStr = String(response.content);

                    // Process BREAK messages
                    if (contentStr === "BREAK") {
                        // Stop SID music on BREAK (Ctrl-C / program interruption)
                        try {
                            if (window.RetroSound && typeof window.RetroSound.stopSidMusic === 'function') {
                                window.RetroSound.stopSidMusic();
                            }
                        } catch (error) {
                            // Ignore errors
                        }
                    }
                    // Check if content is empty (whitespace only or empty string)
                    if (contentStr.trim().length === 0) {
                        // Don't add new line for empty content
                        return;
                    }

                    // LOCATE mode: place text at specific position
                    if (this.locateMode && this.cursorX !== undefined && this.cursorY !== undefined) {
                        // Use response.inverse if available, otherwise this.inverseMode
                        const isInverseLocate = (typeof response.inverse === 'boolean') ? response.inverse : this.inverseMode;
                        this.setTextAtPosition(this.cursorX, this.cursorY, contentStr, isInverseLocate);
                        this.locateMode = false; // Reset after positioning
                        // Text was positioned - no further processing needed
                        // this.drawTerminal(); // drawTerminal is called by the main handler
                        return;
                    } else {
                        // Normal sequential text mode with inverse support
                        const isInverse = (typeof response.inverse === 'boolean') ? response.inverse : this.inverseMode;
                        if (response.noNewline) {
                            // Append to last line
                            if (this.lines.length > 0) {
                                const lastIdx = this.lines.length - 1;
                                this.lines[lastIdx] += contentStr;
                                // Fill inverse array
                                if (!this.inverseLines[lastIdx]) this.inverseLines[lastIdx] = [];
                                for (let i = 0; i < contentStr.length; i++) {
                                    this.inverseLines[lastIdx].push(isInverse);
                                }
                            } else {
                                this.lines.push(contentStr);
                                this.inverseLines.push(Array(contentStr.length).fill(isInverse));
                            }
                        } else {
                            // Add new line(s)
                            const wrappedLines = this.wrapText(contentStr, CFG.TEXT_COLS);
                            for (let i = 0; i < wrappedLines.length; i++) {
                                this.lines.push(wrappedLines[i]);
                                this.inverseLines.push(Array(wrappedLines[i].length).fill(isInverse));
                            }
                        }
                    }
                }
                break;
            case 'CLEAR':
                this.lines = [];
                this.inverseLines = []; // Also clear inverse markings
                this.input = ""; // Clear input line
                this.cursorPos = 0; // Reset cursor position
                
                // Clear screen buffer (LOCATE text)
                this.screenBuffer = null;
                this.locateMode = false;
                this.cursorX = 0;
                this.cursorY = 0;
                this.inverseMode = false;
                
                // Also clear graphics canvas when CLEAR is received
                if (window.RetroGraphics && typeof window.RetroGraphics.handleClearScreen === 'function') {
                    window.RetroGraphics.handleClearScreen();
                }
                // Explicitly re-render the persistent graphics canvas after clearing
                if (window.vectorManager && typeof window.vectorManager.renderVectors === 'function' && 
                    window.persistentGraphicsContext && window.persistentGraphicsCanvas) {
                    window.vectorManager.renderVectors(window.persistentGraphicsContext, 
                        window.persistentGraphicsCanvas.width, window.persistentGraphicsCanvas.height);                }
                break;
            case 'PROMPT':
                if (typeof response.inputEnabled === 'boolean') this.inputEnabled = response.inputEnabled;
                if (typeof response.promptSymbol === 'string') this.promptSymbol = response.promptSymbol;
                // Special handling for cursor reset
                if (response.content === 'reset_cursor') {
                    // Add empty line to ensure output starts on new line
                    this.lines.push("");
                }
                // Immediately redraw to show prompt update
                this.drawTerminal();
                break;
            case 'INPUT':
                // ... existing INPUT logic ...
                if (typeof response.input === 'string') this.input = response.input;
                if (typeof response.cursorPos === 'number') this.cursorPos = response.cursorPos;
                break;
            case 'INPUT_CONTROL':
                // ... existing INPUT_CONTROL logic ...
                if (response.content === 'enable') {
                    this.inputEnabled = true; this.runMode = false; this.passwordMode = false;
                    this.startCursorBlink();
                } else if (response.content === 'disable') {
                    this.inputEnabled = false;
                    this.stopCursorBlink();
                } else if (response.content === 'run_mode') {
                    this.inputEnabled = false; this.runMode = true;
                    this.stopCursorBlink();
                } else if (response.content === 'password_mode_on') {
                    this.passwordMode = true; this.inputEnabled = true;
                    this.startCursorBlink();
                } else if (response.content === 'password_mode_off') {
                    this.passwordMode = false;
                }
                break;
                break;
            case 'BEEP':
                // ... existing BEEP logic ...
                if (window.RetroSound && typeof window.RetroSound.playBeep === 'function') {
                    window.RetroSound.playBeep();
                }
                break;
            case 'SPEAK':
                // ... existing SPEAK logic ...
                if (window.RetroSound && typeof window.RetroSound.speakTextWithID === 'function') {
                    if (response.content && typeof response.speechId !== 'undefined') {
                        window.RetroSound.speakTextWithID(response.content, response.speechId);
                    }                }
                break;
            case 'SOUND':
                if (window.RetroSound && typeof window.RetroSound.playSound === 'function') {
                    let freq, duration;
                    
                    // Parameter aus response.params lesen
                    if (response.params && typeof response.params === 'object') {
                        // SID Music Actions
                        if (response.params.action) {
                            const action = response.params.action;
                            switch (action) {
                                case 'music_open':
                                    if (window.RetroSound && typeof window.RetroSound.openSidMusic === 'function') {
                                        window.RetroSound.openSidMusic(response.params.filename);
                                    }
                                    break;
                                case 'music_play':
                                    if (window.RetroSound && typeof window.RetroSound.playSidMusic === 'function') {
                                        window.RetroSound.playSidMusic();
                                    }
                                    break;
                                case 'music_stop':
                                    if (window.RetroSound && typeof window.RetroSound.stopSidMusic === 'function') {
                                        window.RetroSound.stopSidMusic();
                                    }
                                    break;
                                case 'music_pause':
                                    if (window.RetroSound && typeof window.RetroSound.pauseSidMusic === 'function') {
                                        window.RetroSound.pauseSidMusic();
                                    }
                                    break;
                            }
                            break;
                        }
                        // Noise handling
                        else if (response.params.type === "noise") {
                            const pitch = response.params.pitch;
                            const attack = response.params.attack;
                            const decay = response.params.decay;
                            
                            if (pitch !== undefined && attack !== undefined && decay !== undefined) {
                                if (window.RetroSound.unlockAudio) {
                                    window.RetroSound.unlockAudio();
                                }
                                
                                try {
                                    if (typeof window.RetroSound.playNoiseComplex === 'function') {
                                        window.RetroSound.playNoiseComplex(pitch, attack, decay);
                                    } else {
                                        window.RetroSound.playNoise("white", decay);
                                    }
                                } catch (e) {
                                    // Noise generation failed
                                }
                            }
                            break;
                        } else {
                            freq = response.params.frequency;
                            duration = response.params.duration;
                        }
                    } else if (response.content) {
                        if (response.content === "beep") {
                            freq = 800;
                            duration = 200;
                        } else if (response.content === "floppy") {
                            if (window.RetroSound && typeof window.RetroSound.playFloppySound === 'function') {
                                window.RetroSound.playFloppySound();
                            }
                            break;
                        } else {
                            const parts = response.content.split(',');
                            if (parts.length === 2) {
                                freq = parseFloat(parts[0]);
                                duration = parseFloat(parts[1]);
                            }
                        }
                    }
                    
                    if (freq !== undefined && duration !== undefined && !isNaN(freq) && !isNaN(duration)) {
                        try {
                            window.RetroSound.playSound(freq, duration);
                        } catch (e) {
                            // Sound playback failed
                        }
                    }
                } else if (window.RetroSound && typeof window.RetroSound.playBeep === 'function') {
                    window.RetroSound.playBeep();                }
                break;
            case 'NOISE':
                if (window.RetroSound && typeof window.RetroSound.playNoise === 'function') {
                    let pitch, attack, decay, type, duration;
                    
                    if (response.params && typeof response.params === 'object') {
                        if (response.params.pitch !== undefined && response.params.attack !== undefined && response.params.decay !== undefined) {
                            pitch = response.params.pitch;
                            attack = response.params.attack;
                            decay = response.params.decay;
                            
                            if (pitch !== undefined && attack !== undefined && decay !== undefined) {
                                if (window.RetroSound.unlockAudio) {
                                    window.RetroSound.unlockAudio();
                                }
                                
                                try {
                                    if (typeof window.RetroSound.playNoiseComplex === 'function') {
                                        window.RetroSound.playNoiseComplex(pitch, attack, decay);
                                    } else {
                                        window.RetroSound.playNoise("white", decay);
                                    }
                                } catch (e) {
                                    // Noise generation failed
                                }
                            }
                        } else {
                            type = response.params.type || "white";
                            duration = response.params.duration;
                            
                            if (duration !== undefined && !isNaN(duration)) {
                                try {
                                    window.RetroSound.playNoise(type, duration);
                                } catch (e) {
                                    // Noise playback failed
                                }
                            }
                        }
                    } else if (response.content) {
                        try {
                            const parts = response.content.split(',').map(p => p.trim());
                            if (parts.length === 3) {
                                pitch = parseFloat(parts[0]);
                                attack = parseFloat(parts[1]);
                                decay = parseFloat(parts[2]);
                                
                                if (typeof window.RetroSound.playNoiseComplex === 'function') {
                                    window.RetroSound.playNoiseComplex(pitch, attack, decay);
                                } else {
                                    window.RetroSound.playNoise("white", decay);
                                }
                            } else if (parts.length === 2) {
                                type = parts[0] || "white";
                                duration = parseFloat(parts[1]);
                                window.RetroSound.playNoise(type, duration);
                            } else if (parts.length === 1) {
                                duration = parseFloat(parts[0]);
                                window.RetroSound.playNoise("white", duration);
                            }                        } catch (e) {
                            // Content parsing failed
                        }
                    }
                }
                break;
            case 'EVIL':
                // MCP Evil Effect - Dramatic noise increase for 1 second, then fade back over 5 seconds
                if (window.RetroGraphics && typeof window.RetroGraphics.triggerEvilEffect === 'function') {
                    window.RetroGraphics.triggerEvilEffect();
                } else {
                    console.error('[RetroConsole-EVIL] ERROR: Cannot trigger Evil Effect - RetroGraphics or triggerEvilEffect not available!');
                }
                break;
            case 'MODE':
                if (response.content !== undefined) {
                    const mode = String(response.content).toUpperCase();
                    switch (mode) {
                        case 'OS_SHELL':
                            this.inputMode = 0;
                            break;
                        case 'BASIC':
                            this.inputMode = 1;
                            break;
                        case 'CHESS':
                            this.inputMode = 2;
                            break;
                        case 'EDITOR':
                            this.inputMode = 3;
                            break;
                        case 'TELNET':
                            this.inputMode = 4;
                            break;
                        case 'PAGER':
                            this.inputMode = 5;
                            break;
                        case 'BOARD':
                            this.inputMode = 9;
                            break;
                        default:
                            this.inputMode = 0; // Default to OS_SHELL if unknown
                    }
                }
                if (window.RetroGraphics && typeof window.RetroGraphics.handleModeChange === 'function') {
                    window.RetroGraphics.handleModeChange(response.content); 
                }
                break;            case 'SESSION':
                // ... existing SESSION logic ...
                const newSessionId = response.sessionId || response.sessionID || response.content;
                if (newSessionId) {
                    if (window.sessionStorage) window.sessionStorage.setItem('sessionId', newSessionId);
                    window.sessionId = newSessionId;
                    
                    // Auto-refresh token when session changes (e.g., after login)
                    this.refreshTokenForSession(newSessionId);
                    
                    if (typeof window.sendTerminalConfig === 'function') {
                        setTimeout(window.sendTerminalConfig, 100);
                    }
                }
                break;
            case 'SPRITE':                // Debug logging for SPRITE messages
                if (response.command) {
                    switch (response.command) {                        case 'DEFINE_SPRITE':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleDefineSprite === 'function') {
                                window.RetroGraphics.handleDefineSprite(response);
                            }
                            break;
                        case 'UPDATE_SPRITE':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleUpdateSprite === 'function') {
                                window.RetroGraphics.handleUpdateSprite(response);
                            }
                            break;                        case 'DEFINE_VIRTUAL_SPRITE':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleDefineVirtualSprite === 'function') {
                                window.RetroGraphics.handleDefineVirtualSprite(response);
                            }
                            break;                        
                        case 'CLEAR_ALL_SPRITES':
                            if (window.RetroGraphics && typeof window.RetroGraphics.clearAllSpriteData === 'function') {
                                window.RetroGraphics.clearAllSpriteData();                            
                            }
                            break;                        default:
                            // Unknown sprite command
                            break;
                    }
                }
                break;
            case 'GRAPHICS':
                if (response.command) {
                    const graphicsCommand = {
                        type: response.command,
                        ...response.params
                    };
                    
                    switch (response.command) {
                        case 'PLOT':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handlePlot === 'function') {
                                window.RetroGraphics.handlePlot(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handlePlot not available');
                            }
                            break;
                        case 'LINE':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleLine === 'function') {
                                window.RetroGraphics.handleLine(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handleLine not available');
                            }
                            break;
                        case 'RECT':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleRect === 'function') {
                                window.RetroGraphics.handleRect(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handleRect not available');
                            }
                            break;
                        case 'CIRCLE':
                            // Special debug logging for CIRCLE commands
                            if (this.debugMode) {
                                // console.log('[RetroConsole-CIRCLE] *** CIRCLE command received! ***');
                                // console.log('[RetroConsole-CIRCLE] Graphics command object:', graphicsCommand);
                                // console.log('[RetroConsole-CIRCLE] Raw response:', response);
                            }
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleCircle === 'function') {
                                if (this.debugMode) {
                                    // console.log('[RetroConsole-CIRCLE] Calling RetroGraphics.handleCircle');
                                }
                                window.RetroGraphics.handleCircle(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-CIRCLE] RetroGraphics.handleCircle not available');
                                // console.log('[RetroConsole-CIRCLE] window.RetroGraphics:', window.RetroGraphics);
                            }
                            break;
                        case 'FILL':
                             if (window.RetroGraphics && typeof window.RetroGraphics.handleFill === 'function') {
                                window.RetroGraphics.handleFill(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handleFill not available');
                            }
                            break;
                        case 'CLEAR_SCREEN': 
                            this.lines = [];
                            this.input = "";
                            this.cursorPos = 0;
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleClearScreen === 'function') {
                                window.RetroGraphics.handleClearScreen();
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handleClearScreen not available');
                            }
                            break;
                        case 'UPDATE_VECTOR':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleUpdateVector === 'function') {
                                window.RetroGraphics.handleUpdateVector(graphicsCommand);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.handleUpdateVector not available');
                            }
                            break;
                        case 'CLEAR_ALL_VECTORS':
                            if (window.RetroGraphics && typeof window.RetroGraphics.clearAllVectors === 'function') {
                                window.RetroGraphics.clearAllVectors();
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] RetroGraphics.clearAllVectors not available');
                            }
                            break;                        
                        default:
                            if (this.debugMode) {
                                // console.log('[RetroConsole-GRAPHICS] Unknown graphics command:', response.command);
                            }
                    }
                } else if (this.debugMode) {
                    // console.log('[RetroConsole-GRAPHICS] No command in graphics message');
                }
                break;
            case 'VECTOR':
                // Debug logging for VECTOR messages
                if (this.debugMode) {
                    // console.log('[RetroConsole-VECTOR] Received VECTOR message:', response);
                }
                if (response.command) {
                    switch (response.command) {
                        case 'UPDATE_VECTOR':
                            if (window.RetroGraphics && typeof window.RetroGraphics.handleUpdateVector === 'function') {
                                window.RetroGraphics.handleUpdateVector(response);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-VECTOR] RetroGraphics.handleUpdateVector not available');
                            }
                            break;
                        case 'CLEAR_ALL_VECTORS':
                            if (window.RetroGraphics && typeof window.RetroGraphics.clearAllVectors === 'function') {
                                window.RetroGraphics.clearAllVectors();
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-VECTOR] RetroGraphics.clearAllVectors not available');
                            }
                            break;
                        case 'UPDATE_NODE':
                            if (window.vectorManager && typeof window.vectorManager.handleUpdateNode === 'function') {
                                window.vectorManager.handleUpdateNode(response);
                            } else if (this.debugMode) {
                                // console.log('[RetroConsole-VECTOR] vectorManager.handleUpdateNode not available');
                            }
                            break;
                        default:
                            if (this.debugMode) {
                                // console.log('[RetroConsole-VECTOR] Unknown vector command:', response.command);
                            }
                    }
                } else if (this.debugMode) {
                    // console.log('[RetroConsole-VECTOR] No command in vector message');
                }
                break;
            case 'CHAT':
                this.initiateChatMode();
                break;
            case 'LOCATE':
                // LOCATE x,y - Set cursor position
                if (response.content) {
                    const coords = response.content.split(',');
                    if (coords.length === 2) {
                        // BUGFIX: BASIC uses 1-based coordinates, JavaScript 0-based
                        // Backend already sends 0-based coordinates
                        const newX = Math.max(0, parseInt(coords[0]) || 0);
                        const newY = Math.max(0, parseInt(coords[1]) || 0);
                        // console.log(`[CHESS-DEBUG] LOCATE received: ${response.content} -> setting cursorX=${newX}, cursorY=${newY} (inputMode=${this.inputMode})`);
                        this.cursorX = newX;
                        this.cursorY = newY;
                        if (this.cursorY >= CFG.TEXT_ROWS) this.cursorY = CFG.TEXT_ROWS - 1;
                        if (this.cursorX >= CFG.TEXT_COLS) this.cursorX = CFG.TEXT_COLS - 1;
                        // LOCATE activates text placement mode for all modes
                        this.locateMode = true;
                        // this.drawTerminal(); // drawTerminal is called by the main handler
                    }
                }
                break;
            case 'INVERSE':
                // ... existing INVERSE logic ...
                if (response.content !== undefined) {
                    const mode = String(response.content).toUpperCase();
                    if (mode === 'ON') this.inverseMode = true;
                    else if (mode === 'OFF') this.inverseMode = false;
                    // this.drawTerminal(); // drawTerminal is called by the main handler
                }
                break;
            case 'PAGER':
                // ... existing PAGER logic ...
                this.pagerPrompt = response.pagerPrompt || ""; // Store the prompt
                if (response.content === 'activate') {
                    this.pagerMode = true; this.inputEnabled = false; this.input = ""; this.cursorPos = 0;
                } else if (response.content === 'deactivate') {
                    this.pagerMode = false; this.inputEnabled = true;
                }
                // this.drawTerminal(); // drawTerminal is called by the main handler
                break;
            case 'CURSOR': // This case is handled by drawEditor/drawTerminal directly
                // ... existing CURSOR logic ...
                if (response.content === 'hide') {
                    this.forcedHideCursor = true;
                    if (this.editorMode && this.editorData) this.editorData.hideCursor = true;
                } else if (response.content === 'show') {
                    this.forcedHideCursor = false;
                    if (this.editorMode && this.editorData && !this.editorData.readOnly) {
                        this.editorData.hideCursor = false;
                    }                }
                // Immediate redraw is handled by the main handler calling drawTerminal or drawEditor
                if (this.editorMode) this.drawEditor(); else this.drawTerminal();
                break;
            case 'TELNET':
                this.handleTelnetMessage(response); // Telnet has its own drawTerminal calls
                break;
            // EDITOR case should not be hit here if routed correctly by handleBackendMessage
            // but as a fallback:
            case 'EDITOR': 
                // console.warn('[RetroConsole-INTERNAL-PROC] EDITOR message type unexpectedly reached _processBackendResponse. Routing to handleEditorMessage.');
                this.handleEditorMessage(response);
                break;            case 'BITMAP':
                // Debug logging for BITMAP messages - removed for production
                this.handleBitmapMessage(response);
                break;
            case 'IMAGE':
                // Handle IMAGE messages (LOAD, SHOW, HIDE, ROTATE)
                this.handleImageMessage(response);
                break;
            case 'PARTICLE':
                // Handle PARTICLE messages (CREATE, MOVE, SHOW, HIDE, GRAVITY)
                this.handleParticleMessage(response);
                break;
                
            default:
                // console.warn('[EDITOR-CONSOLE] Unknown editor command:', message.editorCommand, message);
                break;
        }
    },

    handleEditorMessage: function(message) {
        // console.log('[EDITOR-CONSOLE] Received editor command:', message.editorCommand, 'Data:', message);
        if (!this.editorData && message.editorCommand !== "start" && message.editorCommand !== "render") {
            this.editorData = { lines: [], filename: "", readOnly: false, cols: CFG.TEXT_COLS, rows: CFG.TEXT_ROWS, cursorX: 0, cursorY: 0, scrollY: 0, hideCursor: false, statusText:"" };
        }

        switch (message.editorCommand) {            case "start":
                // Only create backup if we're not already in editor mode (first start command)
                if (!this.editorMode) {
                    this.terminalLinesBackup = [...this.lines];
                    this.terminalInverseLinesBackup = [...this.inverseLines];
                    this.terminalScreenBufferBackup = this.screenBuffer ? JSON.parse(JSON.stringify(this.screenBuffer)) : null;
                    this.stopCursorBlink(); // Stop blinking when entering editor mode
                }
                
                this.editorMode = true;// Process content with fallbacks and better debugging
                let contentLines = [];
                  // Check if content is in message.params.content (from render response)
                if (message.params && message.params.content) {
                    // Debug output removed for production
                    const content = message.params.content;
                    
                    // Try different line break formats
                    if (content.includes('\r\n')) {
                        contentLines = content.split('\r\n');
                    } else if (content.includes('\n')) {
                        contentLines = content.split('\n');
                    } else {
                        contentLines = [content];
                    }                    
                    // Debug output removed for production
                }
                // Check message.content as fallback
                else if (message.content) {
                    // console.log('[EDITOR-CONSOLE] "start": Processing message.content, length:', message.content.length);
                    
                    // Check for different formats of line breaks from backend
                    if (message.content.includes('\\r\\n')) {
                        contentLines = message.content.split('\\r\\n');
                        // console.log('[EDITOR-CONSOLE] "start": Split content on \\r\\n, got', contentLines.length, 'lines');
                    } else if (message.content.includes('\\\\r\\\\n')) {
                        contentLines = message.content.split('\\\\r\\\\n');
                        // console.log('[EDITOR-CONSOLE] "start": Split content on \\\\r\\\\n, got', contentLines.length, 'lines');
                    } else if (message.content.includes('\n')) {
                        contentLines = message.content.split('\n');
                        // console.log('[EDITOR-CONSOLE] "start": Split content on \\n, got', contentLines.length, 'lines');
                    } else {
                        // Default: whole content as one line
                        contentLines = [message.content]; 
                        // console.log('[EDITOR-CONSOLE] "start": No line breaks found, using single line');
                    }
                }
                
                // Check message.params.lines as another fallback (from render response)
                else if (message.params && Array.isArray(message.params.lines)) {
                    contentLines = [...message.params.lines];
                    // console.log('[EDITOR-CONSOLE] "start": Using message.params.lines directly, got', contentLines.length, 'lines');
                }
                
                this.editorData = {
                    lines: contentLines,
                    filename: message.params?.filename || "",
                    readOnly: message.params?.readOnly || false,
                    cols: message.params?.cols || CFG.TEXT_COLS,
                    rows: message.params?.rows || CFG.TEXT_ROWS,
                    cursorX: message.params?.cursorX ?? 0,
                    cursorY: message.params?.cursorY ?? 0,
                    scrollY: message.params?.scrollY ?? 0,
                    hideCursor: message.params?.hideCursor ?? false,
                    statusText: `File: ${message.params?.filename || ""} | Ctrl+S Save, Esc Close`
                };
                // Debug output removed for production
                
                this.lines = [...this.editorData.lines]; 
                this.cursorX = this.editorData.cursorX;                this.cursorY = this.editorData.cursorY;
                
                // Debug output removed for production
                this.drawEditor();
                // Debug output removed for production
                
                if (typeof window.sendEditorCommand === 'function') {
                    // Debug output removed for production
                    window.sendEditorCommand("ready", { status: "Editor is ready" });
                } else {
                    console.error('[EDITOR-CONSOLE] "start": sendEditorCommand IS NOT available. CANNOT send "ready".'); // UPDATED LOG
                }
                break;
            case "render":
                // console.log('[EDITOR-CONSOLE] "render": Received render command', message);
                
                if (!this.editorData) { 
                    // console.log('[EDITOR-CONSOLE] "render": Creating new editorData');
                    this.editorData = { 
                        lines: [], 
                        filename: message.params?.filename || "",
                        readOnly: message.params?.readOnly || false,
                        cols: message.params?.textCols || CFG.TEXT_COLS,
                        rows: message.params?.textRows || CFG.TEXT_ROWS,
                        cursorX: 0,
                        cursorY: 0,
                        scrollY: 0,
                        hideCursor: false,
                        statusText: `File: ${message.params?.filename || ""} | Ctrl+S Save, Esc Close`,
                        _lastUpdateTime: Date.now() // Ensure cursor is immediately visible
                    };
                }                // Process render message parameters
                if (message.params) {
                    // console.log('[EDITOR-CONSOLE] "render": Processing render parameters');
                    
                    // Koordinatensystem für den Editor einheitlich verwalten
                    // Logik: Das Backend sendet sowohl die originalen Cursor-Koordinaten (cursorCol/cursorLine)
                    // als auch die für die Anzeige korrigierten Positionen (visibleCursorCol/visibleCursorLine)                    // Store cursor position data from backend  
                    // Backend sends cursorX/cursorY as the logical cursor position in renderParams
                    if (typeof message.params.cursorX === 'number') {
                        this.editorData.cursorX = message.params.cursorX;
                        this.editorData.cursorCol = message.params.cursorX; // Keep both for compatibility
                        // console.log('[EDITOR-CONSOLE] "render": Stored cursorX/cursorCol =', this.editorData.cursorX);
                    }
                    
                    if (typeof message.params.cursorY === 'number') {
                        this.editorData.cursorY = message.params.cursorY;
                        this.editorData.cursorLine = message.params.cursorY; // Keep both for compatibility
                        // console.log('[EDITOR-CONSOLE] "render": Stored cursorY/cursorLine =', this.editorData.cursorY);
                    }
                    
                    // Store visible cursor positions for debugging/fallback
                    if (typeof message.params.visibleCursorCol === 'number') {
                        this.editorData.visibleCursorCol = message.params.visibleCursorCol;
                        // console.log('[EDITOR-CONSOLE] "render": Stored visibleCursorCol =', this.editorData.visibleCursorCol);
                    }
                    
                    if (typeof message.params.visibleCursorLine === 'number') {
                        this.editorData.visibleCursorLine = message.params.visibleCursorLine;
                        // console.log('[EDITOR-CONSOLE] "render": Stored visibleCursorLine =', this.editorData.visibleCursorLine);
                    }
                    
                    if (typeof message.params.mappingSuccess === 'boolean') {
                        this.editorData.mappingSuccess = message.params.mappingSuccess;
                        // console.log('[EDITOR-CONSOLE] "render": Stored mappingSuccess =', this.editorData.mappingSuccess);
                    }                    
                    // Debug output for cursor mapping validation
                    if (typeof message.params.mappingSuccess === 'boolean' && 
                        message.params.mappingSuccess === false) {
                        console.warn('[EDITOR-CONSOLE] WARNING: Backend reported cursor mapping failure!');
                        // console.log('[EDITOR-DEBUG] Full message params:', JSON.stringify(message.params));
                    }
                    
                    // Handle dimensions
                    if (typeof message.params.textCols === 'number') {
                        this.editorData.cols = message.params.textCols;
                        // console.log('[EDITOR-CONSOLE] "render": Set cols =', this.editorData.cols);
                    }
                    if (typeof message.params.textRows === 'number') {
                        this.editorData.rows = message.params.textRows;
                        // console.log('[EDITOR-CONSOLE] "render": Set rows =', this.editorData.rows);
                    }
                    
                    // Handle scroll position
                    if (typeof message.params.scrollY === 'number') {
                        this.editorData.scrollY = message.params.scrollY;
                        // console.log('[EDITOR-CONSOLE] "render": Set scrollY =', this.editorData.scrollY);
                    }
                    
                    // Handle cursor visibility
                    if (typeof message.params.hideCursor === 'boolean') {
                        this.editorData.hideCursor = message.params.hideCursor;
                        // console.log('[EDITOR-CONSOLE] "render": Set hideCursor =', this.editorData.hideCursor);
                    }
                    
                    // Handle status text
                    if (message.params.status) {
                        this.editorData.statusText = message.params.status;
                        // console.log('[EDITOR-CONSOLE] "render": Set statusText =', this.editorData.statusText);
                    }
                    
                    // Handle content lines
                    if (message.params.lines && Array.isArray(message.params.lines)) {
                        this.editorData.lines = message.params.lines;
                        this.lines = [...this.editorData.lines];
                        // console.log('[EDITOR-CONSOLE] "render": Set lines from params.lines, length =', this.editorData.lines.length);
                    }
                }
                
                // Update cursor tracking variables
                this.cursorX = this.editorData.cursorX;
                this.cursorY = this.editorData.cursorY;
                
                // console.log('[EDITOR-CONSOLE] "render": Calling drawEditor');
                this.drawEditor();
                break;
            case "status":
                if (this.editorMode && this.editorData) {
                    // Editor mode: set editor status
                    this.editorData.statusText = message.editorStatus || message.statusText || this.editorData.statusText;
                    this.drawEditor();
                } else {
                    // Terminal mode: set terminal status for pager
                    this.terminalStatus = message.editorStatus || message.statusText || "";
                    this.drawTerminal();
                }
                break;            case "filename_input":
                // Start filename input mode
                this.stopCursorBlink(); // Stop blinking when entering filename input mode
                this.startFilenameInput(message);
                break;
            case "filename_input_complete":
                // Complete filename input
                this.stopFilenameInput();
                this.startCursorBlink(); // Start blinking when exiting filename input mode
                break;            case "stop":
                this.editorMode = false;
                this.filenameInputMode = false;
                this.inputEnabled = true; // Re-enable terminal input after editor exit
                this.forcedHideCursor = false; // Ensure cursor is visible again after editor exit
                this.lines = [...this.terminalLinesBackup];
                this.inverseLines = [...this.terminalInverseLinesBackup];
                this.screenBuffer = this.terminalScreenBufferBackup ? JSON.parse(JSON.stringify(this.terminalScreenBufferBackup)) : null;
                this.editorData = null; 
                this.drawTerminal(); 
                this.startCursorBlink(); // Start blinking when exiting editor mode
                break;
            default:
                // console.warn('[EDITOR-CONSOLE] Unknown editor command:', message.editorCommand, message);
                break;
        }
    },    drawEditor: function() {
        if (!this.editorMode || !this.editorData) {
            console.warn("[EDITOR-CONSOLE] drawEditor called without editorMode or editorData.");
            return;
        }

        if (!this.textCanvas) {
            console.error("[EDITOR-CONSOLE] textCanvas is not available for drawEditor.");
            return;
        }
        const ctx = this.textCanvas.getContext('2d');
        if (!ctx) {
            console.error("[EDITOR-CONSOLE] Canvas context is not available for drawEditor.");
            return;
        }// Reduzierte Debug-Logging - nur bei tatsächlichen Debugging-Maßnahmen ausgeben
        if (this.debugMode && this.editorData.debug) {
            let cursorDebug = '';
            if (this.editorData.visibleCursorLine !== undefined && this.editorData.visibleCursorCol !== undefined) {
                cursorDebug = `Visible Cursor: Line ${this.editorData.visibleCursorLine}, Col ${this.editorData.visibleCursorCol}`;
            } else {
                cursorDebug = `Original Cursor: Line ${this.editorData.cursorLine}, Col ${this.editorData.cursorCol}`;
            }
            // console.log(`[EDITOR-DEBUG] ${cursorDebug}, Scroll: ${this.editorData.scrollY}, Total wrapped: ${this.editorData.totalLines}`);
        }

        ctx.imageSmoothingEnabled = false;
        ctx.fillStyle = '#000000'; 
        ctx.fillRect(0, 0, this.textCanvas.width, this.textCanvas.height);

        const padL = CFG.SCREEN_PADDING_LEFT;
        const padT = CFG.SCREEN_PADDING_TOP;
        ctx.font = `${this.CHAR_FONT_SIZE}px ${CFG.FONT_FAMILY}`;
        ctx.textBaseline = 'top';

        const visibleRows = this.editorData.rows - 1;                // Zeichne die vom Backend bereitgestellten vorbereiteten Zeilen
        // Diese wurden bereits umgebrochen und entsprechen genau dem, was im Viewport sichtbar sein soll
        for (let i = 0; i < visibleRows; i++) {
            if (i < this.editorData.lines.length) {
                // Zeile abrufen und für Unicode korrekt als Array von Zeichen behandeln
                const line = this.editorData.lines[i];
                let lineChars = Array.from(line || ""); // Konvertiere zu Array von Zeichen für korrekte Unicode-Behandlung
                
                ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Standard-Grün für Text
                
                // Optimiert: Zeichne jeden Buchstaben einzeln für bessere Kontrolle
                for (let col = 0; col < Math.min(lineChars.length, this.editorData.cols); col++) {
                    const char = lineChars[col];
                    if (char === undefined || char === null) continue; // Überspringen ungültiger Zeichen
                    
                    const x = padL + col * this.CHAR_WIDTH;
                    const y = padT + i * this.CHAR_HEIGHT;
                    
                    // Zeichne den Buchstaben
                    ctx.fillText(char, x, y);
                }
                
                // Verbesserte Debug-Visualisierung für Zeilenumbrüche
                if (this.editorData.debug === true) {
                    // Zeile, die Teil eines Umbruchs sein könnte, mit subtiler visueller Markierung versehen
                    if (line && (line.length >= this.editorData.cols - 1)) {
                        // Subtiles Symbol am rechten Rand, das den Umbruch anzeigt, ohne störend zu sein
                        const debugX = padL + (this.editorData.cols - 1) * this.CHAR_WIDTH;
                        const debugY = padT + i * this.CHAR_HEIGHT;
                        ctx.fillStyle = '#335533'; // Dunkleres Grün für die Debug-Markierung
                        ctx.fillText('⤶', debugX, debugY);
                    }
                }            }
        }
          // Draw cursor and status bar using dedicated function
        this.drawEditorCursorAndStatus(ctx, padL, padT);
                
        // Update text texture if needed
        if (this.textTexture) {
            this.textTexture.needsUpdate = true;
        }    },drawEditorCursorAndStatus: function(ctx, padL, padT) {
        if (!this.editorMode || !this.editorData) {
            return;
        }        // Use logical cursor position from backend (cursorX/cursorY from backend params)
        // These represent the actual cursor position that should be displayed
        let cursorScreenX = -1;
        let cursorScreenY = -1;
        
        // Backend sends cursor position as cursorX/cursorY in renderParams
        if (typeof this.editorData.cursorX === 'number' && typeof this.editorData.cursorY === 'number') {
            cursorScreenX = this.editorData.cursorX;
            cursorScreenY = this.editorData.cursorY;
            // console.log('[EDITOR-CURSOR] Using cursor position from cursorX/cursorY:', cursorScreenY, cursorScreenX);
        } else if (typeof this.editorData.cursorCol === 'number' && typeof this.editorData.cursorLine === 'number') {
            // Fallback to cursorCol/cursorLine if cursorX/cursorY not available
            cursorScreenX = this.editorData.cursorCol;
            cursorScreenY = this.editorData.cursorLine;
            // console.log('[EDITOR-CURSOR] Using fallback cursor position from cursorCol/cursorLine:', cursorScreenY, cursorScreenX);
        } else if (typeof this.editorData.visibleCursorCol === 'number' && typeof this.editorData.visibleCursorLine === 'number') {
            // Final fallback to visible cursor positions
            cursorScreenX = this.editorData.visibleCursorCol;
            cursorScreenY = this.editorData.visibleCursorLine;
            // console.log('[EDITOR-CURSOR] Using final fallback cursor position from visibleCursor:', cursorScreenY, cursorScreenX);
        }
        
        // Draw cursor if position is valid and cursor is not hidden
        if (!this.editorData.hideCursor && 
            cursorScreenX >= 0 && cursorScreenY >= 0 &&
            cursorScreenY < CFG.TEXT_ROWS - 1) { // Leave space for status line
            
            // Draw cursor as inverted background (like a classic text cursor)
            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Bright green for cursor background
            const x = padL + cursorScreenX * this.CHAR_WIDTH;
            const y = padT + cursorScreenY * this.CHAR_HEIGHT;
            ctx.fillRect(x, y, this.CHAR_WIDTH, this.CHAR_HEIGHT);
            
            // If there's a character at cursor position, draw it in inverted colors
            if (cursorScreenY < this.editorData.lines.length) {
                const line = this.editorData.lines[cursorScreenY];
                if (line && cursorScreenX < line.length) {
                    const char = line[cursorScreenX];
                    ctx.fillStyle = '#000000'; // Black text on bright cursor
                    ctx.fillText(char, x, y);
                }
            }
            
            // console.log('[EDITOR-CURSOR] Drew cursor at screen position:', cursorScreenY, cursorScreenX);
        }

        // Status line - draw exactly like Telnet status line
        if (this.editorData.statusText) {
            const statusRow = CFG.TEXT_ROWS - 1; // Last row
            const statusY = padT + statusRow * this.CHAR_HEIGHT;
            
            // Hintergrund der Statuszeile (invertiert) - exact copy from drawTelnetStatusLine
            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15]; // Heller Hintergrund
            const bgY = statusY - 2; // 2 Pixel vor der Oberkante des Textes
            const bgHeight = this.CHAR_HEIGHT - 2; // Text-Höhe minus 2 Pixel (endet 2 Pixel vor nächster Linie)
            ctx.fillRect(padL, bgY, CFG.TEXT_COLS * this.CHAR_WIDTH, bgHeight);
            
            // Statustext mit schwarzer Schrift auf hellem Hintergrund zeichnen
            ctx.fillStyle = '#000000'; // Schwarzer Text für inverse Darstellung
            const statusText = this.editorData.statusText;
            for (let col = 0; col < Math.min(statusText.length, CFG.TEXT_COLS); col++) {
                const char = statusText[col];
                const x = padL + col * this.CHAR_WIDTH;
                ctx.fillText(char, x, statusY);
            }
              // Textfarbe für nachfolgende Zeichen zurücksetzen
            ctx.fillStyle = CFG.BRIGHTNESS_LEVELS[15];
            
            // console.log('[EDITOR-STATUS] Drew status line (Telnet-style):', this.editorData.statusText);
        }
    },

    // Telnet message processing
    handleTelnetMessage: function(response) {
        if (response.content === 'start') {
            // Enter telnet mode
            this.telnetMode = true;
            this.telnetServerEcho = false; // Initialize as false until server negotiates
            // console.log('[TELNET-MODE] Telnet mode activated, server:', response.params?.serverName);
            this.telnetServerName = response.params?.serverName || 'Unknown Server';
            this.inputEnabled = true;
            this.runMode = false;            // Clear screen and show telnet header
            this.lines = [];
            this.inverseLines = [];
            this.lines.push(`Connected to ${this.telnetServerName}`);
            this.inverseLines.push([]);
            this.lines.push('Press Ctrl+X or ESC to exit telnet session');
            this.inverseLines.push([]);
            this.lines.push('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
            this.inverseLines.push([]);
            
            // Reset message processing state to prevent duplicates
            if (window.lastTelnetMessage) {
                window.lastTelnetMessage = { hash: '', time: 0 };
            }        } else if (response.content === 'end') {
            // Exit telnet mode
            // console.log('[TELNET-MODE] Telnet mode deactivated');
            this.telnetMode = false;
            this.telnetServerEcho = false; // Reset echo flag
            this.telnetEchoBuffer = []; // Clear echo buffer
            this.telnetServerName = '';
            
            // Reset message processing state
            if (window.lastTelnetMessage) {
                window.lastTelnetMessage = { hash: '', time: 0 };
            }
        } else if (response.content === 'echo_state') {
            // Echo state change from server
            if (response.params) {
                this.telnetServerEcho = response.params.serverEcho || false;
                const localEcho = response.params.localEcho || false;
                // console.log(`[TELNET-ECHO] Echo state changed: serverEcho=${this.telnetServerEcho}, localEcho=${localEcho}`);
                
                // Clear echo buffer when echo state changes to prevent mismatched echoes
                this.telnetEchoBuffer = [];
            }
        } else {
            // Regular telnet data - process immediately like Unix telnet
            if (this.telnetMode && response.content) {
                this.processTerminalData(response.content);
                // Force immediate redraw
                this.drawTerminal();
                return; // Don't call drawTerminal again at the end
            }
        }
        
        this.drawTerminal();
    },// Process incoming telnet data with ANSI support
    processTelnetData: function(data) {
        // Deprecated - now using direct processing        this.processTerminalData(data);
    },    // Real terminal emulation - process data character by character
    processTerminalData: function(data) {
        if (!data) return;
          // Initialize backspace throttling if not exists
        if (!this.backspaceThrottle) {
            this.backspaceThrottle = {
                lastBackspace: 0,
                throttleMs: 50 // Minimum time between backspace operations
            };
        }// TELNET mode: Only process data from server, ignore local echo
        // The telnet server handles echoing, so we don't need to suppress anything
        // Just process all incoming data directly
        
        // Check for Telnet IAC (Interpret As Command) sequences
        if (data.charCodeAt(0) === 255) { // IAC
            this.processTelnetIAC(data);
            return;
        }
            // Removed excessive debug logging for performance
        
        var i = 0;
        
        while (i < data.length) {
            const char = data[i];
            const charCode = char.charCodeAt(0);
            
            // Filter Telnet protocol characters (IAC commands)
            if (charCode === 255) { // IAC (Interpret As Command)
                // Skip IAC sequences - typically 3 bytes: IAC + command + option
                if (i + 2 < data.length) {
                    const command = data.charCodeAt(i + 1);
                    if (command >= 251 && command <= 254) { // WILL, WONT, DO, DONT
                        i += 3; // Skip 3-byte sequence
                        continue;
                    }
                }
                i += 2; // Skip 2-byte sequence or unknown
                continue;
            }            // Filter other control characters that shouldn't be displayed
            // Keep: CR(\r=13), LF(\n=10), TAB(\t=9), ESC(\x1b=27), BS(\x08=8), BEL(\x07=7)
            if (charCode < 32 && charCode !== 13 && charCode !== 10 && charCode !== 9 && charCode !== 27 && charCode !== 8 && charCode !== 7) {
                i++;
                continue;
            }
              // Check for ANSI escape sequence
            if (char === '\x1b' && i + 1 < data.length && data[i + 1] === '[') {
                // Find the end of the escape sequence
                let j = i + 2;
                while (j < data.length && !/[A-Za-z]/.test(data[j])) {
                    j++;
                }                if (j < data.length) {
                    // Extract and process the escape sequence
                    const sequence = data.substring(i, j + 1);
                    this.processANSISequence(sequence);
                    i = j + 1;
                    continue;
                }
            }
              // Handle regular characters
            if (char === '\r') {
                // Carriage return - move cursor to beginning of line
                this.cursorX = 0;            } else if (char === '\n') {
                // Line feed - move to next line
                this.cursorY++;
                this.cursorX = 0;
                this.cursorY = this.ensureLine(this.cursorY);
                this.validateCursor();} else if (char === '\x08') {
                // Backspace - move cursor back and delete character
                // Implement backspace throttling to prevent multiple rapid backspaces
                const now = Date.now();
                if (!this.lastBackspaceTime) this.lastBackspaceTime = 0;                // Only process backspace if enough time has passed (50ms threshold)
                if (now - this.lastBackspaceTime > 50) {                    if (this.cursorX > 0) {
                        this.cursorX--;
                        // Remove character at current position
                        this.cursorY = this.ensureLine(this.cursorY);
                        if (this.cursorY < this.lines.length && this.lines[this.cursorY].length > this.cursorX) {
                            this.lines[this.cursorY] = this.lines[this.cursorY].substring(0, this.cursorX) + 
                                                      this.lines[this.cursorY].substring(this.cursorX + 1);
                            // Also update inverse array
                            if (this.inverseLines[this.cursorY] && this.inverseLines[this.cursorY].length > this.cursorX) {
                                this.inverseLines[this.cursorY].splice(this.cursorX, 1);
                            }
                        }
                    }
                    this.lastBackspaceTime = now;} else {
                    // Throttle backspace to prevent spam
                }            } else if (char === '\x07') {
                // BEL (Bell) - ASCII 7, play beep sound
                if (window.RetroSound && typeof window.RetroSound.playBeep === 'function') {
                    window.RetroSound.playBeep();
                }
            } else if (charCode >= 32) {// Printable character - add to current position
                this.cursorY = this.ensureLine(this.cursorY);
                this.setCharAt(this.cursorY, this.cursorX, char);
                this.cursorX++;
                this.validateCursor();
            }
            
            i++;
        }        
          // Removed excessive debug logging for performance
          
        // Update display
        this.drawTerminal();
    },
    
    // Process Telnet IAC (Interpret As Command) sequences
    processTelnetIAC: function(data) {        // Reduced IAC logging for performance
        
        if (data.length >= 3) {
            var iac = data.charCodeAt(0);     // Should be 255
            var command = data.charCodeAt(1);
            var option = data.charCodeAt(2);
              // Reduced IAC logging
              // NAWS (Negotiate About Window Size) - Option 31
            if (option === 31) {
                if (command === 253) { // DO NAWS - Server asks if we support window size
                    // Reduced logging
                    this.sendTelnetResponse([255, 251, 31]); // WILL NAWS
                    this.sendWindowSize();
                } else if (command === 251) { // WILL NAWS - Server will use window size
                    // Reduced logging
                }
            }
            // ECHO - Option 1 (Critical for preventing double characters)
            else if (option === 1) {
                if (command === 251) { // WILL ECHO - Server will handle echoing
                    // console.log('[TELNET-ECHO] Server WILL ECHO - disabling local echo');
                    this.sendTelnetResponse([255, 253, 1]); // DO ECHO - We agree
                    this.telnetServerEcho = true; // Server handles echo
                } else if (command === 252) { // WONT ECHO - Server won't handle echoing
                    // console.log('[TELNET-ECHO] Server WONT ECHO - enabling local echo');
                    this.sendTelnetResponse([255, 254, 1]); // DONT ECHO - We disagree
                    this.telnetServerEcho = false; // We handle echo locally
                } else if (command === 253) { // DO ECHO - Server asks if we will echo
                    // console.log('[TELNET-ECHO] Server asks DO ECHO - we refuse');
                    this.sendTelnetResponse([255, 252, 1]); // WONT ECHO - We refuse
                }
            }
            // Terminal Type - Option 24
            else if (option === 24) {
                if (command === 253) { // DO TERMINAL-TYPE                    // Reduced logging
                    this.sendTelnetResponse([255, 251, 24]); // WILL TERMINAL-TYPE
                }
            }
        }
    },
    
    // Send Telnet response
    sendTelnetResponse: function(bytes) {
        var response = '';
        for (var i = 0; i < bytes.length; i++) {
            response += String.fromCharCode(bytes[i]);
        }        // Reduced logging
        
        const message = {
            type: 1, // MessageTypeText
            content: response,
            sessionId: window.sessionId
        };
        
        if (window.sendMessageWithSessionID) {
            window.sendMessageWithSessionID(message);
        }
    },
    
    // Send window size via NAWS
    sendWindowSize: function() {        // Reduced logging
        
        // IAC SB NAWS <cols-high> <cols-low> <rows-high> <rows-low> IAC SE
        var cols = CFG.TEXT_COLS;
        var rows = CFG.TEXT_ROWS;
        
        var response = [
            255, 250, 31,           // IAC SB NAWS
            Math.floor(cols / 256), cols % 256,  // columns (high byte, low byte)
            Math.floor(rows / 256), rows % 256,  // rows (high byte, low byte)
            255, 240                // IAC SE
        ];
        
        this.sendTelnetResponse(response);
    },    // Process a single ANSI escape sequence
    processANSISequence: function(sequence) {
        // Handle cursor visibility sequences first
        if (sequence.includes('?25h')) {
            // Show cursor (ESC[?25h)
            this.forcedHideCursor = false;
            return;
        }
        if (sequence.includes('?25l')) {
            // Hide cursor (ESC[?25l)
            this.forcedHideCursor = true;
            return;
        }
        
        // Handle text attribute sequences (SGR - Select Graphic Rendition)
        const sgrMatch = sequence.match(/\x1b\[([0-9;]*)m/);
        if (sgrMatch) {
            const params = sgrMatch[1] ? sgrMatch[1].split(';').map(p => parseInt(p) || 0) : [0];
            // For now, we only handle reset (0) and inverse mode (7/27)
            for (const param of params) {
                switch (param) {
                    case 0: // Reset all attributes
                        this.inverseMode = false;
                        break;
                    case 7: // Inverse/reverse video
                        this.inverseMode = true;
                        break;
                    case 27: // Not inverse/reverse video
                        this.inverseMode = false;
                        break;
                    // Other SGR codes could be added here as needed
                }
            }
            return;
        }
        
        const match = sequence.match(/\x1b\[([0-9;]*)([A-Za-z])/);
        if (!match) return;        const params = match[1] ? match[1].split(';').map(p => parseInt(p) || 0) : [0];
        const command = match[2];
          
        switch (command) {
            case 'H': // Cursor Home/Position
            case 'f': // Cursor Position
                if (params.length >= 2) {
                    this.cursorY = Math.max(0, (params[0] || 1) - 1); // Convert to 0-based
                    this.cursorX = Math.max(0, (params[1] || 1) - 1); // Convert to 0-based
                } else {
                    this.cursorY = 0; // Home position
                    this.cursorX = 0;
                }
                
                // In telnet mode, always limit cursor to visible area
                if (this.telnetMode) {
                    if (this.cursorY >= CFG.TEXT_ROWS) {                        // Reduced logging
                        this.cursorY = CFG.TEXT_ROWS - 1;
                    }
                    if (this.cursorX >= CFG.TEXT_COLS) {                        // Reduced logging
                        this.cursorX = CFG.TEXT_COLS - 1;
                    }
                }                  // Reduced logging
                this.cursorY = this.ensureLine(this.cursorY); // This will now handle telnet buffer limits
                break;
                  
            case 'J': // Clear Display
                const clearType = params[0] || 0;
                if (clearType === 2) {                    // Reduced logging
                    this.lines = [];
                    this.inverseLines = [];
                    this.cursorY = 0;
                    this.cursorX = 0;
                } else if (clearType === 0) {                    // Reduced logging
                    // Clear from cursor to end of screen
                    if (this.cursorY < this.lines.length) {
                        this.lines[this.cursorY] = this.lines[this.cursorY].substring(0, this.cursorX);
                        this.lines.splice(this.cursorY + 1);
                        this.inverseLines.splice(this.cursorY + 1);
                    }
                }
                break;            case 'K': // Clear Line
                const lineType = params[0] || 0;
                if (lineType === 2) {
                    // Clear entire line
                    if (this.cursorY < this.lines.length) {
                        this.lines[this.cursorY] = '';
                        this.inverseLines[this.cursorY] = [];
                    }
                } else if (lineType === 0) {
                    // Clear from cursor to end of line
                    if (this.cursorY < this.lines.length) {
                        this.lines[this.cursorY] = this.lines[this.cursorY].substring(0, this.cursorX);
                    }
                }
                break;
                
            case 'A': // Cursor Up
                this.cursorY = Math.max(0, this.cursorY - (params[0] || 1));
                break;            case 'B': // Cursor Down
                this.cursorY = this.cursorY + (params[0] || 1);
                this.cursorY = this.ensureLine(this.cursorY);
                break;case 'C': // Cursor Forward
                this.cursorX = this.cursorX + (params[0] || 1);
                break;
            case 'D': // Cursor Back
                this.cursorX = Math.max(0, this.cursorX - (params[0] || 1));
                break;
        }
    },
      
    // Simple cursor validation - ensure cursor stays within visible area
    validateCursor: function() {
        const maxLines = CFG.TEXT_ROWS - 1; // Reserve last line for status
        
        // Handle line wrapping when cursor exceeds line width
        if (this.cursorX >= CFG.TEXT_COLS) {
            this.cursorX = 0;
            this.cursorY++;
            
            // Handle scrolling if we exceed available lines
            if (this.cursorY >= maxLines) {
                this.cursorY = maxLines - 1;
                // Scroll lines up
                this.lines.shift();
                this.inverseLines.shift();
                this.lines.push('');
                this.inverseLines.push([]);
            }
        }
        
        // Clamp cursor Y to visible area
        if (this.cursorY >= maxLines) {
            this.cursorY = maxLines - 1;
        }
          
        // Ensure cursor is never negative
        this.cursorX = Math.max(0, this.cursorX);
        this.cursorY = Math.max(0, this.cursorY);
    },    // Simple terminal line management - RFC compliant  
    // We maintain exactly (TEXT_ROWS - 1) lines, last line is status
    ensureLine: function(row) {
        // In telnet mode: Only reserve one line for status (same as normal mode)
        const maxLines = CFG.TEXT_ROWS - 1;
          // Reduced logging
        
        // Add lines until we have enough
        while (this.lines.length <= row) {
            this.lines.push('');
            this.inverseLines.push([]);
        }
        
        // If we exceed max lines, scroll up (remove first line, shift others up)
        while (this.lines.length > maxLines) {
            if (this.debugMode) {
                // Reduced logging
            }
            this.lines.shift(); // Remove first line
            this.inverseLines.shift();
            
            // Adjust cursor position after scroll
            if (this.cursorY > 0) {
                this.cursorY--;
            }
            // Adjust the requested row after scroll
            if (row > 0) {
                row--;
            }
            if (this.debugMode) {
                // Reduced logging
            }
        }
          // Reduced logging
        return row; // Return possibly adjusted row index
    },
        
    // Set character at specific position
    setCharAt: function(row, col, char) {
        // Adjust row if it was modified by scrolling
        row = this.ensureLine(row);
        let line = this.lines[row];
          // Reduced logging
        
        // Extend line if needed
        while (line.length <= col) {
            line += ' ';
        }          // Replace character at position
        line = line.substring(0, col) + char + line.substring(col + 1);
        this.lines[row] = line;
        // Reduced logging
    },

    // Handle editor keyboard input
    handleEditorKeyDown: function(event) {
        if (!this.editorMode || !this.editorData) {
            console.warn("[EDITOR-INPUT] handleEditorKeyDown called without editorMode or editorData");
            return;
        }        // In read-only mode, only block text input, not navigation
        if (this.editorData.readOnly) {
            const navigationKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", 
                                   "Home", "End", "PageUp", "PageDown"];
            const isCtrlC = event.ctrlKey && event.key.toLowerCase() === "c";
            const isCtrlX = event.ctrlKey && event.key.toLowerCase() === "x";
            const isEscape = event.key === "Escape";
            
            if (navigationKeys.includes(event.key)) {
                // Navigation keys are allowed and sent to backend for proper scrolling
                // Continue with normal processing
            } else if (isCtrlC || isCtrlX || isEscape) {
                // Allow exit commands to be sent to backend
                // Continue with normal processing
            } else {
                // Block all other keys (text input, backspace, delete, etc.)
                event.preventDefault();
                // console.log("[EDITOR-INPUT] Text input blocked in read-only mode:", event.key);
                return;
            }        }
        
        // Prevent default browser actions for most keys when in editor mode
        // But in read-only mode, don't prevent navigation keys
        let shouldPreventDefault = true;
        if (this.editorData.readOnly) {
            const navigationKeys = ["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", 
                                   "Home", "End", "PageUp", "PageDown", "Escape"];
            const isCtrlC = event.ctrlKey && event.key.toLowerCase() === "c";
            const isCtrlX = event.ctrlKey && event.key.toLowerCase() === "x";
            
            if (navigationKeys.includes(event.key) || isCtrlC || isCtrlX) {
                shouldPreventDefault = false;
            }
        }
        
        if (shouldPreventDefault && (!event.ctrlKey || (event.ctrlKey && !['c','v'].includes(event.key.toLowerCase())))) {
            event.preventDefault();
        }
        
        // Debug output removed for production
        // Map key to editor command
        let command = "key_input"; // All editor commands use key_input
        let data = {};
          // Key mapping for cleaner code
        const keyMap = {
            "Escape": "Escape",
            "ArrowUp": "ArrowUp", 
            "ArrowDown": "ArrowDown",
            "ArrowLeft": "ArrowLeft",
            "ArrowRight": "ArrowRight",
            "Home": "Home",
            "End": "End", 
            "PageUp": "PageUp",
            "PageDown": "PageDown",
            "Backspace": "Backspace",
            "Delete": "Delete",
            "Enter": "Enter",
            "Tab": "Tab"
        };        // Handle special keys and control combinations
        if (event.ctrlKey && event.key.toLowerCase() === "s") {
            data = { key: "CTRL+S" };
        } else if (event.ctrlKey && event.key.toLowerCase() === "x") {
            data = { key: "CTRL+X" };
        } else if (event.ctrlKey && event.key.toLowerCase() === "c") {
            data = { key: "CTRL+C" };
        } else if (keyMap[event.key]) {
            data = { key: keyMap[event.key] };
        } else if (event.key.length === 1) {
            // Regular character input
            data = { key: "char", char: event.key };
        } else {
            // Unknown key - don't send anything
            command = null;
        }// Send command to backend if valid
        if (command && typeof window.sendEditorCommand === "function") {
            // console.log("[EDITOR-INPUT] Sending command to backend:", command, data);
            window.sendEditorCommand(command, data);
        } else if (command) {
            console.error("[EDITOR-INPUT] sendEditorCommand function not available");
        } else {
            // console.log("[EDITOR-INPUT] No command mapped for key:", event.key);
        }
    },

    // Additional methods for filename input handling
    startFilenameInput: function(response) {
        // console.log('[EDITOR-DEBUG] Starting filename input mode');
        
        // Small delay to prevent the original Ctrl+S event from being processed again
        setTimeout(() => {
            // Switch to filename input mode
            this.filenameInputMode = true;
            this.filenameInput = '';
            
            // Get prompt from response
            const prompt = response.params && response.params.prompt ? response.params.prompt : 'Save as: ';
            this.filenamePrompt = prompt;
            
            // Update status to show filename input
            this.editorStatus = prompt;
            
            // Re-draw to show the prompt
            if (this.editorMode && typeof this.drawEditor === 'function') {
                this.drawEditor();
            } else {
                this.drawTerminal();
            }
            
            // console.log('[EDITOR-DEBUG] Filename input mode started with prompt:', prompt);
        }, 50); // 50ms delay
    },    
    stopFilenameInput: function() {
        // console.log('[EDITOR-DEBUG] Stopping filename input mode');
        
        // Exit filename input mode
        this.filenameInputMode = false;
        this.filenameInput = '';
        this.filenamePrompt = '';
        
        // Reset status
        this.editorStatus = '';
        
        // Re-draw to update display        this.drawTerminal();
        
        // Debug output removed for production
    },

    // Chat mode initiation - automatically opens a Chat WebSocket connection
    initiateChatMode: function() {
        
        // Check if a chat connection already exists
        if (window.chatWs && window.chatWs.readyState === WebSocket.OPEN) {
            return;
        }
            // Get SessionID for chat connection
        const sessionId = window.sessionId || (window.sessionStorage && window.sessionStorage.getItem('sessionId'));
        if (!sessionId) {
            this.lines.push("Error: No valid session found for chat.");
            this.drawTerminal();
            return;
        }
        
        // Get JWT token for chat WebSocket authentication
        let authToken = null;
        if (window.authManager && window.authManager.token) {
            authToken = window.authManager.token;
        } else {
            // Fallback: try to get token from localStorage
            try {
                authToken = localStorage.getItem('tinyos_token');
            } catch (e) {
                console.warn('[CHAT] Could not access localStorage for token');
            }
        }
        
        if (!authToken) {
            this.lines.push("Error: No valid authentication token found for chat.");
            this.drawTerminal();
            return;
        }
          
        // SECURITY: Chat WebSocket URL with JWT token for authentication
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const chatUrl = `${protocol}//${window.location.host}/chat?token=${encodeURIComponent(authToken)}&session=${encodeURIComponent(sessionId)}`;
        
        // Session ID stored separately for secure transmission
        const chatSessionId = sessionId;
        
         try {
            // Create chat WebSocket
            window.chatWs = new WebSocket(chatUrl);
            
            // SECURITY: Transmit session ID securely via WebSocket message
            window.chatWs.onopen = function() {
                // Send session ID as first message for secure authentication
                const sessionAuthMsg = {
                    type: 'auth',
                    sessionId: chatSessionId
                };
                
                window.chatWs.send(JSON.stringify(sessionAuthMsg));
            };
            
            window.chatWs.onmessage = function(event) {
                try {
                    // Chat messages can contain multiple JSON objects, separated by line breaks
                    const messages = event.data.trim().split('\n').filter(line => line.trim() !== '');
                    
                    messages.forEach(function(messageData) {
                        try {
                            const chatResponse = JSON.parse(messageData);
                              
                            // Add chat messages to terminal
                            if (chatResponse.content) {
                                // Role-specific formatting
                                let prefix = "";
                                if (chatResponse.role === "ai") {
                                    prefix = "MCP: ";
                                } else if (chatResponse.role === "system") {
                                    prefix = "System: ";
                                }
                                
                                const wrappedLines = window.RetroConsole.wrapText(prefix + chatResponse.content, CFG.TEXT_COLS);
                                window.RetroConsole.lines.push(...wrappedLines);
                            }
                            
                            // Special handling for different chat response types (independent of content)
                            if (chatResponse.type === 2) { // BEEP
                                if (window.RetroSound && typeof window.RetroSound.playBeep === 'function') {
                                    window.RetroSound.playBeep();
                                }
                            } else if (chatResponse.type === 3) { // SPEAK
                                if (window.RetroSound && typeof window.RetroSound.speakText === 'function') {
                                    // In chat mode we use speakText without ID, as no SAY_DONE is required
                                    window.RetroSound.speakText(chatResponse.content);
                                }
                            }
                            
                            if (chatResponse.error) {
                                const wrappedError = window.RetroConsole.wrapText("Error: " + chatResponse.error, CFG.TEXT_COLS);
                                window.RetroConsole.lines.push(...wrappedError);
                            }
                            
                        } catch (e) {
                            // Ignore parse errors for individual messages
                        }
                    });
                    
                    // Update terminal only once after all messages
                    window.RetroConsole.drawTerminal();
                    
                } catch (e) {
                    window.RetroConsole.lines.push("Error processing chat response.");
                    window.RetroConsole.drawTerminal();
                }
            };
              
            window.chatWs.onclose = function(event) {
                window.RetroConsole.inputEnabled = true; // Re-enable input
                window.RetroConsole.lines.push("");
                window.RetroConsole.lines.push("Chat terminated.");
                window.RetroConsole.drawTerminal();
                window.chatWs = null;
            };
            
            window.chatWs.onerror = function(error) {
                window.RetroConsole.inputEnabled = true; // Re-enable input
                window.RetroConsole.lines.push("Error in chat connection.");
                window.RetroConsole.drawTerminal();
                window.chatWs = null;
            };
            
            // Add ping/pong handler for chat WebSocket (like normal terminal WebSocket)
            window.chatWs.addEventListener('ping', function(event) {
                // Pong is automatically sent by browser
            });

            window.chatWs.addEventListener('pong', function(event) {
                // Pong received
            });

            // Additional debug handlers for WebSocket events
            window.chatWs.addEventListener('open', function(event) {
                // Chat WebSocket opened
            });

            window.chatWs.addEventListener('close', function(event) {
                // Cleanup keepalive interval on close
                if (window.chatKeepAliveInterval) {
                    clearInterval(window.chatKeepAliveInterval);
                    window.chatKeepAliveInterval = null;
                }
            });

            // Keepalive mechanism for chat WebSocket
            window.chatKeepAliveInterval = setInterval(function() {
                if (window.chatWs && window.chatWs.readyState === WebSocket.OPEN) {
                    try {
                        window.chatWs.send(JSON.stringify({ type: 'keepalive' }));
                    } catch (e) {
                        // Ignore keepalive errors
                    }
                }
            }, 30000); // 30 seconds
              
        } catch (e) {
            this.lines.push("Error opening chat connection.");
            this.drawTerminal();
        }
    },

    // Initialize screen buffer for LOCATE functionality
    initScreenBuffer: function() {
        if (!this.screenBuffer) {
            this.screenBuffer = [];
            for (let y = 0; y < CFG.TEXT_ROWS; y++) {
                this.screenBuffer[y] = [];
                for (let x = 0; x < CFG.TEXT_COLS; x++) {
                    this.screenBuffer[y][x] = { char: ' ', inverse: false };
                }
            }
        }
    },
    
    // Set text at specific coordinates (for LOCATE)
    setTextAtPosition: function(x, y, text, inverse = false) {
        this.initScreenBuffer();
        
        if (y < 0 || y >= CFG.TEXT_ROWS) return;
        
        for (let i = 0; i < text.length && (x + i) < CFG.TEXT_COLS; i++) {
            if ((x + i) >= 0) {
                this.screenBuffer[y][x + i] = { 
                    char: text[i], 
                    inverse: inverse 
                };
            }
        }
    },

    // Handle bitmap messages for chess board and pieces
    handleBitmapMessage: function(response) {
        // console.log('[RetroConsole-BITMAP] Processing bitmap message:', response);
        // console.log('[RetroConsole-BITMAP] BitmapData length:', response.bitmapData ? response.bitmapData.length : 'undefined');
        // console.log('[RetroConsole-BITMAP] Position:', response.bitmapX, response.bitmapY);
        // console.log('[RetroConsole-BITMAP] Scale:', response.bitmapScale);
        // console.log('[RetroConsole-BITMAP] ID:', response.bitmapID);

        // Create bitmap element
        const img = new Image();
        img.onload = () => {
            // console.log('[RetroConsole-BITMAP] Image loaded successfully, dimensions:', img.width, 'x', img.height);
            
            // Create canvas for quantization to 16 brightness levels
            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');
            
            canvas.width = img.width;
            canvas.height = img.height;
            ctx.drawImage(img, 0, 0);
            // console.log('[RetroConsole-BITMAP] Canvas created, size:', canvas.width, 'x', canvas.height);
              // Quantize to 16 brightness levels for retro effect and map to green tones
            const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
            const data = imageData.data;
            
            // Get the brightness levels from config (16 green tones)
            const brightnessLevels = window.CONFIG && window.CONFIG.BRIGHTNESS_LEVELS ? 
                window.CONFIG.BRIGHTNESS_LEVELS : [
                    '#000000', '#001500', '#002500', '#003500', '#004500',
                    '#005500', '#006000', '#007000', '#008000', '#009000', 
                    '#00A000', '#00B000', '#00C000', '#00D000', '#00E000', '#5FFF5F'
                ];
            
            for (let i = 0; i < data.length; i += 4) {
                // Convert to grayscale and quantize to 16 levels
                const gray = (data[i] + data[i + 1] + data[i + 2]) / 3;
                const level = Math.min(15, Math.floor(gray / 16));
                
                // Map to green color from brightness levels
                const greenColor = brightnessLevels[level];
                const r = parseInt(greenColor.slice(1, 3), 16);
                const g = parseInt(greenColor.slice(3, 5), 16);
                const b = parseInt(greenColor.slice(5, 7), 16);
                
                data[i] = r;     // Red
                data[i + 1] = g; // Green  
                data[i + 2] = b; // Blue
                // Alpha channel (data[i + 3]) remains unchanged
            }
            
            ctx.putImageData(imageData, 0, 0);
            // console.log('[RetroConsole-BITMAP] Applied retro quantization effect with green mapping');
            
            // Apply scaling if specified
            const scale = response.bitmapScale || 1.0;
            if (scale !== 1.0) {
                const scaledCanvas = document.createElement('canvas');
                const scaledCtx = scaledCanvas.getContext('2d');
                
                scaledCanvas.width = canvas.width * scale;
                scaledCanvas.height = canvas.height * scale;
                
                scaledCtx.drawImage(canvas, 0, 0, scaledCanvas.width, scaledCanvas.height);
                canvas.width = scaledCanvas.width;
                canvas.height = scaledCanvas.height;
                ctx.drawImage(scaledCanvas, 0, 0);
            }
            
            // Apply rotation if specified
            const rotation = response.bitmapRotate || 0;
            if (rotation !== 0) {
                const rotatedCanvas = document.createElement('canvas');
                const rotatedCtx = rotatedCanvas.getContext('2d');
                
                rotatedCanvas.width = canvas.width;
                rotatedCanvas.height = canvas.height;
                
                rotatedCtx.translate(canvas.width / 2, canvas.height / 2);
                rotatedCtx.rotate((rotation * Math.PI) / 180);
                rotatedCtx.drawImage(canvas, -canvas.width / 2, -canvas.height / 2);
                
                canvas.width = rotatedCanvas.width;
                canvas.height = rotatedCanvas.height;
                ctx.clearRect(0, 0, canvas.width, canvas.height);
                ctx.drawImage(rotatedCanvas, 0, 0);
            }            // Draw directly to WebGL graphics system for proper CRT effects
            // Since chess commands work, the system must be ready
            const x = response.bitmapX || 0;
            const y = response.bitmapY || 0;
            
            // console.log('[RetroConsole-BITMAP] Drawing bitmap at position:', x, y);
            // Debug output removed for production
            // Check if WebGL graphics system is available
            // Use persistent2D canvas for chess bitmaps as they should persist between frames
            const graphicsContext = window.RetroGraphics?.persistent2DContext || window.persistent2DContext;
            const graphicsCanvas = window.RetroGraphics?.persistent2DCanvas || window.persistent2DCanvas;
            
            if (window.RetroGraphics && graphicsContext && graphicsCanvas) {
                try {
                    // Draw directly to the persistent 2D graphics context (not cleared every frame)
                    graphicsContext.drawImage(canvas, x, y);
                    
                    // Mark graphics as dirty so they get rendered in the next frame
                    window.RetroGraphics._graphics2DDirty = true;
                } catch (error) {
                    console.error('[RetroConsole-BITMAP] Error drawing bitmap to persistent 2D graphics:', error);
                }
            }
        };
        
        img.onerror = (error) => {
            console.error('[RetroConsole-BITMAP] Failed to load bitmap image:', error);
            console.error('[RetroConsole-BITMAP] Image src:', img.src);
            console.error('[RetroConsole-BITMAP] BitmapData preview:', response.bitmapData ? response.bitmapData.substring(0, 100) + '...' : 'undefined');
        };
        
        // Load bitmap from base64 data
        if (response.bitmapData) {
            // console.log('[RetroConsole-BITMAP] Setting image src with base64 data');
            img.src = 'data:image/png;base64,' + response.bitmapData;
        } else {
            console.error('[RetroConsole-BITMAP] No bitmapData provided in response');
        }
    },
    
    // Handle image messages for IMAGE commands
    handleImageMessage: function(response) {
        // Initialize imageManager inline if it doesn't exist
        if (!window.imageManager) {
            console.log('[RetroConsole-IMAGE] Initializing imageManager inline...');
            this.initInlineImageManager();
        }
        
        this.processImageMessage(response);
    },
    
    // Inline imageManager implementation
    initInlineImageManager: function() {
        let imageObjects = [];
        const maxImageHandle = 8;
        
        // Initialize dirty flag
        if (!window.RetroGraphics) {
            window.RetroGraphics = {};
        }
        if (typeof window.RetroGraphics._imagesDirty === 'undefined') {
            window.RetroGraphics._imagesDirty = false;
        }
        
        // Find image by handle
        const findImageByHandle = (handle) => {
            return imageObjects.find(obj => obj.handle === handle);
        };
        
        // Convert scale parameter to actual scale factor
        const getScaleFactor = (scaleParam) => {
            if (scaleParam === 0) return 1.0; // Original size
            if (scaleParam > 0) {
                return 1.0 + scaleParam;
            } else {
                return 1.0 / (1.0 - scaleParam);
            }
        };
        
        // Create imageManager object
        window.imageManager = {
            handleLoadImage: function(data) {
                const handle = data.id;
                const customData = data.customData;
                
                if (!customData || !customData.imageData) {
                    console.error('[IMAGE-INLINE] LOAD_IMAGE missing image data');
                    return false;
                }
                
                const imageData = customData.imageData;
                const width = customData.width || 0;
                const height = customData.height || 0;
                const filename = customData.filename || 'unknown';
                
                const img = new Image();
                img.onload = function() {
                    const imageObj = {
                        handle: handle,
                        image: img,
                        width: width,
                        height: height,
                        filename: filename,
                        visible: false,
                        x: 0,
                        y: 0,
                        scale: 0,
                        rotation: 0,
                        loaded: true
                    };
                    
                    // Remove existing image with same handle
                    imageObjects = imageObjects.filter(obj => obj.handle !== handle);
                    imageObjects.push(imageObj);
                    
                    console.log('[IMAGE-INLINE] Image loaded:', filename, 'handle:', handle);
                    window.RetroGraphics._imagesDirty = true;
                };
                
                img.onerror = function() {
                    console.error('[IMAGE-INLINE] Failed to load image:', filename);
                };
                
                img.src = 'data:image/png;base64,' + imageData;
                return true;
            },
            
            handleShowImage: function(data) {
                const handle = data.id;
                const position = data.position;
                const scale = data.scale || 0;
                
                const imgObj = findImageByHandle(handle);
                if (!imgObj) {
                    console.warn('[IMAGE-INLINE] SHOW_IMAGE: Image not found for handle:', handle);
                    return false;
                }
                
                imgObj.visible = true;
                imgObj.x = position.x || 0;
                imgObj.y = position.y || 0;
                imgObj.scale = scale;
                
                console.log('[IMAGE-INLINE] Showing image:', imgObj.filename, 'at', imgObj.x + ',' + imgObj.y, 'scale:', scale);
                window.RetroGraphics._imagesDirty = true;
                return true;
            },
            
            handleHideImage: function(data) {
                const handle = data.id;
                const imgObj = findImageByHandle(handle);
                if (!imgObj) {
                    console.warn('[IMAGE-INLINE] HIDE_IMAGE: Image not found for handle:', handle);
                    return false;
                }
                
                imgObj.visible = false;
                console.log('[IMAGE-INLINE] Hiding image:', imgObj.filename);
                window.RetroGraphics._imagesDirty = true;
                return true;
            },
            
            handleRotateImage: function(data) {
                const handle = data.id;
                const rotation = data.vecRotation;
                
                const imgObj = findImageByHandle(handle);
                if (!imgObj) {
                    console.warn('[IMAGE-INLINE] ROTATE_IMAGE: Image not found for handle:', handle);
                    return false;
                }
                
                if (rotation && typeof rotation.z === 'number') {
                    imgObj.rotation = rotation.z;
                    console.log('[IMAGE-INLINE] Rotating image:', imgObj.filename, 'to', (rotation.z * 180 / Math.PI).toFixed(1), 'degrees');
                }
                
                window.RetroGraphics._imagesDirty = true;
                return true;
            },
            
            renderImages: function(ctx, canvasWidth, canvasHeight) {
                if (!ctx) return;
                
                imageObjects.forEach(imgObj => {
                    if (!imgObj.visible || !imgObj.loaded) return;
                    
                    ctx.save();
                    const scaleFactor = getScaleFactor(imgObj.scale);
                    const displayWidth = imgObj.width * scaleFactor;
                    const displayHeight = imgObj.height * scaleFactor;
                    
                    ctx.translate(imgObj.x + displayWidth / 2, imgObj.y + displayHeight / 2);
                    ctx.rotate(imgObj.rotation);
                    ctx.scale(scaleFactor, scaleFactor);
                    ctx.drawImage(imgObj.image, -imgObj.width / 2, -imgObj.height / 2);
                    ctx.restore();
                });
            },
            
            initImageManager: function() {
                console.log('[IMAGE-INLINE] Image manager initialized inline');
            }
        };
        
        console.log('[RetroConsole-IMAGE] Inline imageManager created successfully');
    },
    
    processImageMessage: function(response) {
        
        if (!response.command) {
            console.warn('[RetroConsole-IMAGE] No command in image message');
            return;
        }
        
        switch (response.command) {
            case 'LOAD_IMAGE':
                window.imageManager.handleLoadImage(response);
                break;
            case 'SHOW_IMAGE':
                window.imageManager.handleShowImage(response);
                break;
            case 'HIDE_IMAGE':
                window.imageManager.handleHideImage(response);
                break;
            case 'ROTATE_IMAGE':
                window.imageManager.handleRotateImage(response);
                break;
            default:
                console.warn('[RetroConsole-IMAGE] Unknown image command:', response.command);
        }
    },
    
    // Handle particle messages for PARTICLE commands
    handleParticleMessage: function(response) {
        // Initialize particleManager inline if it doesn't exist
        if (!window.particleManager) {
            console.warn('[RetroConsole-PARTICLE] particleManager not available, creating fallback...');
            
            // Create minimal fallback particleManager
            window.particleManager = {
                handleCreateEmitter: function(data) {
                    console.log('[PARTICLE-FALLBACK] Create emitter:', data);
                },
                handleMoveEmitter: function(data) {
                    console.log('[PARTICLE-FALLBACK] Move emitter:', data);
                },
                handleShowEmitter: function(data) {
                    console.log('[PARTICLE-FALLBACK] Show emitter:', data);
                },
                handleHideEmitter: function(data) {
                    console.log('[PARTICLE-FALLBACK] Hide emitter:', data);
                },
                handleSetGravity: function(data) {
                    console.log('[PARTICLE-FALLBACK] Set gravity:', data);
                }
            };
        }
        
        if (!response.command) {
            console.warn('[RetroConsole-PARTICLE] No command in particle message');
            return;
        }
        
        switch (response.command) {
            case 'CREATE_EMITTER':
                window.particleManager.handleCreateEmitter(response);
                break;
            case 'MOVE_EMITTER':
                window.particleManager.handleMoveEmitter(response);
                break;
            case 'SHOW_EMITTER':
                window.particleManager.handleShowEmitter(response);
                break;
            case 'HIDE_EMITTER':
                window.particleManager.handleHideEmitter(response);
                break;
            case 'SET_GRAVITY':
                window.particleManager.handleSetGravity(response);
                break;
            default:
                console.warn('[RetroConsole-PARTICLE] Unknown particle command:', response.command);
        }
    },
    
    refreshTokenForSession: async function(sessionId) {
        // console.log('[RetroConsole-TOKEN] Refreshing token for session:', sessionId);
        
        // Check if global auth manager is available
        if (window.authManager || window.globalAuthManager) {
            const authManager = window.authManager || window.globalAuthManager;
            
            try {
                // Request new token from backend using the session ID
                // console.log('[RetroConsole-TOKEN] Requesting token for session:', sessionId);
                
                const loginResponse = await fetch(`${window.location.origin}/api/auth/login`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    credentials: 'include',
                    body: JSON.stringify({
                        sessionId: sessionId
                    })
                });
                
                const loginData = await loginResponse.json();
                
                if (loginData.success && loginData.token) {
                    // Update stored token and session ID
                    authManager.setStoredToken(loginData.token);
                    authManager.setStoredSessionId(loginData.sessionId);
                    
                    // console.log('[RetroConsole-TOKEN] Token successfully refreshed');
                    // console.log('[RetroConsole-TOKEN] New token:', loginData.token.substring(0, 20) + '...');
                    
                } else {
                    console.error('[RetroConsole-TOKEN] Failed to refresh token:', loginData.message);
                }
                
            } catch (error) {
                console.error('[RetroConsole-TOKEN] Error refreshing token:', error);
            }
        } else {
            console.warn('[RetroConsole-TOKEN] Auth manager not available for token refresh');
        }
    },
    
    // Command History Functions
    addToHistory: function(command) {
        // Don't add empty commands or commands that are identical to the last one
        if (!command.trim() || (this.commandHistory.length > 0 && this.commandHistory[this.commandHistory.length - 1] === command.trim())) {
            return;
        }
        
        // Add to history
        this.commandHistory.push(command.trim());
        
        // Limit history size
        if (this.commandHistory.length > this.maxHistorySize) {
            this.commandHistory.shift(); // Remove oldest entry
        }
        
        // Reset history index
        this.historyIndex = -1;
    },
    
    navigateHistory: function(direction) {
        if (this.commandHistory.length === 0) {
            return;
        }
        
        if (direction === 'up') {
            // Go backwards in history (newer to older)
            if (this.historyIndex === -1) {
                // First time accessing history - go to most recent
                this.historyIndex = this.commandHistory.length - 1;
            } else if (this.historyIndex > 0) {
                this.historyIndex--;
            }
        } else if (direction === 'down') {
            // Go forwards in history (older to newer)
            if (this.historyIndex !== -1) {
                this.historyIndex++;
                if (this.historyIndex >= this.commandHistory.length) {
                    // Past the end - clear input
                    this.historyIndex = -1;
                    this.input = "";
                    this.cursorPos = 0;
                    this.drawTerminal();
                    return;
                }
            }
        }
        
        // Load command from history
        if (this.historyIndex !== -1 && this.historyIndex < this.commandHistory.length) {
            this.input = this.commandHistory[this.historyIndex];
            this.cursorPos = this.input.length; // Move cursor to end
            this.drawTerminal();
        }
    }
});
