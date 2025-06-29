/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// retroterminal.js
// Einstiegspunkt und zentrale Logik für das Retro-Terminal

let graphicsInitialized = false;

// Global authentication manager instance
let globalAuthManager = null;

// Suppress ScriptProcessorNode deprecation warnings for SID player
// Diese Warnung kommt von der jsSID.js Bibliothek und ist nicht kritisch
(function() {
    const originalWarn = console.warn;
    console.warn = function(message) {
        if (typeof message === 'string' && 
            (message.includes('ScriptProcessorNode') || message.includes('createJavaScriptNode'))) {
            return; // Suppress SID player audio deprecation warnings
        }
        originalWarn.apply(console, arguments);
    };
})();

// Sicherstellen, dass DynamicViewport initialisiert ist
if (window.DynamicViewport && typeof window.DynamicViewport.init === 'function') {
    window.DynamicViewport.init();
} else {

}

// Funktion zur Initialisierung der nicht-grafischen Komponenten
function initializeNonGraphics() {
    // Warten, bis RetroConsole und seine Kernmethoden verfügbar sind
    const checkRetroConsoleReady = setInterval(() => {
        if (window.RetroConsole && 
            typeof window.RetroConsole.handleBackendMessage === 'function' &&
            typeof window.RetroConsole.drawTerminal === 'function' &&
            typeof window.RetroConsole.updateCharMetrics === 'function') {            clearInterval(checkRetroConsoleReady);

            // Sound initialisieren
            if (window.RetroSound) {
                window.RetroSound.initAudio();
                window.RetroSound.initSpeech();
                // Überprüfe, ob playSound verfügbar ist
                if (typeof window.RetroSound.playSound !== 'function') {

                    window.RetroSound.playSound = function(freq, duration) {
                        try {
                            if (!window.RetroSound.audioContext) {
                                window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();
                            }
                            const ctx = window.RetroSound.audioContext;
                            if (ctx.state === 'suspended') {
                                ctx.resume();
                            }
                            
                            const oscillator = ctx.createOscillator();
                            const gainNode = ctx.createGain();
                            oscillator.type = 'sine';
                            oscillator.frequency.setValueAtTime(freq || 440, ctx.currentTime);
                            gainNode.gain.setValueAtTime(0.2, ctx.currentTime);
                            oscillator.connect(gainNode);
                            gainNode.connect(ctx.destination);
                            oscillator.start();
                            oscillator.stop(ctx.currentTime + (duration ? duration / 1000 : 0.2));
                        } catch (e) {

                        }
                    };
                }
            } else {

            }            
            // Setup WebSocket with authentication
            (async () => {
                await setupWebSocket(); // WebSocket einrichten
            })();

            // Grafikinitialisierung starten, NACHDEM RetroGraphics vollständig initialisiert wurde            // Überprüfe, ob das Flag gesetzt ist oder warte auf das Event
            if (window.retroGraphicsFullyInitialized) {
                initializeGraphics();
            } else {
                document.addEventListener('retrographicsready', initializeGraphics, { once: true });
            }
        } 
    }, 100); // Alle 100ms prüfen
}

// Funktion zur Initialisierung der Grafikkomponenten
function initializeGraphics() {
    if (graphicsInitialized) {
        return;
    }
    
    // Flag SOFORT setzen, um Rekursion zu verhindern
    graphicsInitialized = true;
    
    // Sicherstellen, dass RetroGraphics und die benötigten Funktionen vorhanden sind
    if (!window.RetroGraphics || typeof window.RetroGraphics.initGraphicsPipeline !== 'function' || typeof window.RetroGraphics.animateCRT !== 'function') {
        return;
    }    try {
        if (window.RetroConsole && typeof window.RetroConsole.updateCharMetrics === 'function') {
            // Initialize RetroConsole's textCanvas first
            if (typeof window.RetroConsole.initTextCanvas === 'function') {
                window.RetroConsole.initTextCanvas();
            }
            window.RetroConsole.updateCharMetrics();
        }
        
        // Pass RetroConsole's textCanvas and textTexture to RetroGraphics
        const textCanvas = window.RetroConsole ? window.RetroConsole.textCanvas : null;
        const textTexture = window.RetroConsole ? window.RetroConsole.textTexture : null;
        
        if (textCanvas && textTexture) {
            window.RetroGraphics.initGraphicsPipeline(textCanvas, textTexture);
        } else {
            window.RetroGraphics.initGraphicsPipeline();
        }
        
        // Dynamisches Viewport für Grafiken initialisieren, falls vorhanden
        if (typeof window.RetroGraphics.initDynamicViewportForGraphics === 'function') {
            window.RetroGraphics.initDynamicViewportForGraphics();
        }
        
        window.RetroGraphics.animateCRT();        
        // Eingabebehandlung erst nach erfolgreicher Grafikinitialisierung starten
        setupInputHandling();        
        
        if (window.RetroConsole) {
            window.RetroConsole.inputEnabled = true; // Eingabe jetzt explizit aktivieren
            
            if (typeof window.RetroConsole.drawTerminal === 'function') {
                window.RetroConsole.drawTerminal();
            }
        }// Debug-Vektor-Rendering (optional, kann hier bleiben)
        setTimeout(() => {
            if (window.RetroGraphics && typeof window.RetroGraphics.debugVectorRendering === 'function') {
                window.RetroGraphics.debugVectorRendering();
                // Regelmäßige Überprüfung der Vector-Szene
                window.vectorDebugInterval = setInterval(() => {
                    if (window.RetroGraphics && window.RetroGraphics.getVectorSceneInfo) {
                        const sceneInfo = window.RetroGraphics.getVectorSceneInfo();
                        // Prüfen, ob die Vektor-Szene leer ist oder Probleme hat
                        if (!sceneInfo.initialized || sceneInfo.children === 0) {

                            // Erst versuchen, alle regulären Vektoren wiederherzustellen
                            if (window.RetroGraphics.resetAndRebuildVectorScene && 
                                window.RetroGraphics.resetAndRebuildVectorScene()) {
                            } else {
                                // Wenn keine regulären Vektoren vorhanden waren, Debug-Vektoren erstellen

                                window.RetroGraphics.debugVectorRendering();
                            }
                        }
                    }
                }, 3000); // Alle 3 Sekunden prüfen
            }
            // Event-Listener für Fenster-Fokus/Blur - kann WebGL-Rendering-Problemen vorbeugen
            window.addEventListener('focus', () => {
                if (window.RetroGraphics) {
                    // Priorisierte Wiederherstellung:
                    // 1. Erst versuchen, alle registrierten Vektoren wiederherzustellen
                    if (typeof window.RetroGraphics.resetAndRebuildVectorScene === 'function' &&
                        window.RetroGraphics.resetAndRebuildVectorScene()) {
                    } 
                    // 2. Sonst Debug-Vektoren anzeigen als Fallback
                    else if (typeof window.RetroGraphics.debugVectorRendering === 'function') {
                        window.RetroGraphics.debugVectorRendering();
                    }
                }
            });
        }, 2000); // 2 Sekunden Verzögerung

    } catch (error) {        // Versuchen, den Fehler im Terminal anzuzeigen, falls RetroConsole verfügbar ist
        if (window.RetroConsole && Array.isArray(window.RetroConsole.lines)) {
            window.RetroConsole.lines.push("Graphics initialization error!");
            if(error.message) window.RetroConsole.lines.push(error.message.substring(0, CFG.TEXT_COLS - 2));
            
            // Zurück zum ursprünglichen Verhalten
            if (typeof window.RetroConsole.drawTerminal === 'function') {
                window.RetroConsole.drawTerminal();
            }
        }
    }
}

// WebSocket-Verbindung einrichten
async function setupWebSocket() {    
    // Close any existing WebSocket connection to prevent duplicate message handling
    if (window.ws && window.ws.readyState !== WebSocket.CLOSED) {
        window.ws.close();
        window.ws = null;
    }
    
    // Initialize authentication manager if not done yet
    if (!globalAuthManager) {
        globalAuthManager = new AuthManager();
        await globalAuthManager.initialize();
    }
    
    // Get websocket URL with authentication
    const wsURL = await globalAuthManager.getWebSocketUrl();
    
    try {
        const ws = new WebSocket(wsURL);
        window.ws = ws; // Aktualisiere auch die globale Referenz

        ws.onopen = () => {
            // JWT authentication is handled via URL, session init is for compatibility
            const sessionInitMsg = { 
                type: 7, 
                content: "", 
                sessionId: globalAuthManager.sessionId || getSessionID() || ""
            };
            ws.send(JSON.stringify(sessionInitMsg));
            
            setTimeout(() => { 
                // Sicherstellen, dass sendTerminalConfig existiert
                if (typeof window.sendTerminalConfig === 'function') {
                    sendTerminalConfig();                } else {

                }
            }, 300);
            
            // Sicherstellen, dass RetroConsole und seine Eigenschaften/Methoden existieren
            if (window.RetroConsole) {
                window.RetroConsole.inputEnabled = true;
                if (Array.isArray(window.RetroConsole.lines)) {
                    window.RetroConsole.lines.push("Dialing mainframe...");
                } else {
                    window.RetroConsole.lines = ["Dialing mainframe..."]; // Fallback
                }
                if (typeof window.RetroConsole.drawTerminal === 'function') {
                    window.RetroConsole.drawTerminal();
                }
                // Start cursor blinking when input is enabled
                if (typeof window.RetroConsole.startCursorBlink === 'function') {
                    window.RetroConsole.startCursorBlink();
                }
            } else {

            }
        };        // WebSocket message handler
        ws.onmessage = function(event) {
            // Forward raw event to RetroConsole.handleBackendMessage for processing
            if (window.RetroConsole && typeof window.RetroConsole.handleBackendMessage === 'function') {
                window.RetroConsole.handleBackendMessage(event);
            } else {
                console.error('RetroConsole.handleBackendMessage not available');
            }
        };        ws.onclose = (event) => {
            console.log('[FRONTEND-DEBUG] WebSocket connection closed:', {
                timestamp: new Date().toISOString(),
                code: event.code,
                reason: event.reason,
                wasClean: event.wasClean
            });
            
            // Verhindere Recovery-Versuche, wenn bereits einer läuft
            if (window.sessionRecoveryInProgress) {
                return;
            }
            
            // Prüfe den Close-Code, um zwischen normalen Schließungen und Session-Fehlern zu unterscheiden  
            if (event.code === 1006 || event.code === 1011) {
                // Möglicher Session-Fehler - versuche Session zu erneuern
                attemptSessionRecovery();
            } else {
                RetroConsole.lines.push("Terminal connection lost, please reload.");
            }
        };

        ws.onerror = (error) => {
            console.log('[FRONTEND-DEBUG] WebSocket error occurred:', {
                timestamp: new Date().toISOString(),
                error: error
            });

            // Verhindere Recovery-Versuche, wenn bereits einer läuft
            if (window.sessionRecoveryInProgress) {
                return;
            }

            attemptSessionRecovery();
        };// Ping/Pong handler for WebSocket stability
        ws.addEventListener('ping', (event) => {
            // Browser should automatically respond
        });

        ws.addEventListener('pong', (event) => {
            // Pong received
        });// Keepalive mechanism: Send regular small messages
        const keepAliveInterval = setInterval(() => {
            if (ws.readyState === WebSocket.OPEN) {
                try {
                    ws.send(JSON.stringify({ type: 'keepalive' }));
                } catch (e) {

                }
            }
        }, 30000); // 30 seconds        // Cleanup on WebSocket close
        ws.addEventListener('close', () => {
            clearInterval(keepAliveInterval);
        });

        // Globale Referenz speichern
        window.backendSocket = ws;
        
    } catch (error) {

        if (window.RetroConsole && window.RetroConsole.lines) {
            window.RetroConsole.lines.push("Connection problems occurred.");
        }
    }
}

// Funktion zum Abrufen von Sicherheits-Tokens (Platzhalter für lokale Entwicklung)
function initSecurity() {

    // In einer realen Umgebung würden hier echte Tokens abgerufen
    // (z.B. aus Meta-Tags, Cookies oder über einen API-Aufruf)
    return {
        csrfToken: 'dummy-csrf-token', // Platzhalter
        sessionToken: 'dummy-session-token' // Platzhalter
        // guestJWTToken könnte hier auch relevant sein, basierend auf der setupWebSocket-Logik
    };
}

// Eingabebehandlung einrichten
function setupInputHandling() {
    // Prevent multiple event listeners by checking if already set up
    if (window.inputHandlingSetup) {
        return;
    }
    window.inputHandlingSetup = true;
    
    // Key-Down Event Handler for INKEY$ support and editor
    window.addEventListener('keydown', function(event) {
        if (!window.RetroConsole || !graphicsInitialized) {
            return; 
        }
        
        // Pager-Modus: Spezielle Behandlung für einzelne Tastenanschläge
        if (window.RetroConsole.pagerMode) {
            handlePagerKeyDown(event);
            return;
        }        // Telnet mode: Special handling for all inputs
        if (window.RetroConsole && window.RetroConsole.telnetMode) {
            event.preventDefault(); // Stop default browser behavior
            event.stopPropagation(); // Stop event bubbling
            
            if (typeof handleTelnetKeyDown === 'function') {
                handleTelnetKeyDown(event);
            } else {
                console.error('handleTelnetKeyDown function not available');
            }
            return; // Important: Exit here to prevent normal processing
        }// Editor mode: Special handling via RetroConsole
        if (window.RetroConsole && window.RetroConsole.editorMode) {
            if (typeof window.RetroConsole.handleEditorKeyDown === 'function') {
                window.RetroConsole.handleEditorKeyDown(event);
                // Force redraw for cursor visibility
                if (typeof window.RetroConsole.drawEditor === 'function') {
                    window.RetroConsole.drawEditor();
                }
            } else {

            }
            return;
        }
        // STRG+C sollte IMMER funktionieren (auch bei deaktivierter Eingabe)
        if (event.ctrlKey && event.key.toLowerCase() === 'c') {
            event.preventDefault();  // Standardaktionen verhindern
            // Im Editor-Modus wird STRG+C von RetroConsole.handleEditorKeyDown behandelt
            // und sollte dort sendEditorCommand('key_input', 'CTRL+C') auslösen.
            // Daher hier keine spezielle Behandlung mehr für Editor-Modus nötig.
            
            // Im normalen Terminal-Modus: __BREAK__ Kommando als Nachricht senden
            const breakMessage = {
                type: 1,  // Typ 1 für Eingabe
                content: "__BREAK__"
            };
            
            // Im normalen Terminal-Modus (nicht RUN-Modus) ^C anzeigen
            // Im RUN-Modus wird das BREAK vom Backend gesendet
            if (!window.RetroConsole.runMode && window.RetroConsole.inputEnabled) {
                window.RetroConsole.lines.push(window.RetroConsole.promptSymbol + "^C");
            }
            
            // Message senden
            if (sendMessageWithSessionID(breakMessage)) {

            }
            
            // Input zurücksetzen
            window.RetroConsole.input = "";
            window.RetroConsole.cursorPos = 0;
            window.RetroConsole.drawTerminal();
            return;
        }        // INKEY$ Events should work even when normal input is disabled
        if (shouldSendKeyEvent(event.key)) {
            sendKeyEvent(16, event.key); // MessageTypeKeyDown = 16
        }
        
        // Normal input only when inputEnabled is active
        if (!window.RetroConsole.inputEnabled) {
            return;
        }
          if (['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End', 'Backspace', 'Delete', 'PageUp', 'PageDown'].includes(event.key)) {
            event.preventDefault();        }
        let ws = window.ws;
          switch (event.key) {
            case 'Enter':
                // Im Telnet-Modus wird Enter bereits von handleTelnetKeyDown verarbeitet
                if (window.RetroConsole && window.RetroConsole.telnetMode) {
                    return; // Keine weitere Verarbeitung im Telnet-Modus
                }
                
                // Im RUN-Modus keine normalen Terminal-Eingaben erlauben
                if (window.RetroConsole.runMode) {
                    return;
                }
                
                // Prüfe, ob Chat-Modus aktiv ist
                if (window.chatWs && window.chatWs.readyState === WebSocket.OPEN) {
                    // Chat-Modus: Sende Nachricht über Chat-WebSocket
                    const chatMessage = {
                        type: "text",
                        content: window.RetroConsole.input,
                        cols: CFG.TEXT_COLS,
                        rows: CFG.TEXT_ROWS
                    };                      // Zeige die eigene Nachricht im Terminal
                    const currentInput = "You: " + window.RetroConsole.input;
                    const wrappedLines = window.RetroConsole.wrapText(currentInput, CFG.TEXT_COLS);
                    window.RetroConsole.lines.push(...wrappedLines);
                    
                    // BUGFIX: Auch inverseLines Array synchron halten - Chat-Zeilen sind immer normal (nicht inverse)
                    for (let i = 0; i < wrappedLines.length; i++) {
                        window.RetroConsole.inverseLines.push(Array(wrappedLines[i].length).fill(false));
                    }
                    
                    // Nach dem Hinzufügen der neuen Zeilen: Prüfe, ob Scrolling erforderlich ist
                    if (typeof window.RetroConsole.checkAndScroll === 'function') {
                        window.RetroConsole.checkAndScroll();
                    }
                    
                    // Sende über Chat-WebSocket
                    window.chatWs.send(JSON.stringify(chatMessage));
                    
                    // Eingabe zurücksetzen
                    window.RetroConsole.input = "";
                    window.RetroConsole.cursorPos = 0;
                    window.RetroConsole.drawTerminal();
                      } else {
                    // Normal Terminal Mode: Send via standard WebSocket
                    const inputMessage = {
                        type: 1, 
                        content: window.RetroConsole.input
                    };
                    
                    // Display the current input line in the lines, even for empty input
                    // In password mode, mask the input with asterisks
                    let displayInput = window.RetroConsole.input;
                    if (window.RetroConsole.passwordMode) {
                        displayInput = '*'.repeat(window.RetroConsole.input.length);
                    }
                    const currentInput = window.RetroConsole.promptSymbol + displayInput;                      // For multi-line input: Split into lines according to terminal width
                    const maxLineLength = CFG.TEXT_COLS;
                    if (currentInput.length <= maxLineLength) {
                        // Input fits in one line
                        window.RetroConsole.lines.push(currentInput);
                        // BUGFIX: Also keep inverseLines array in sync - input lines are always normal (not inverse)
                        window.RetroConsole.inverseLines.push(Array(currentInput.length).fill(false));
                    } else {
                        // Split input across multiple lines
                        let remaining = currentInput;
                        while (remaining.length > 0) {
                            const lineText = remaining.substring(0, maxLineLength);
                            window.RetroConsole.lines.push(lineText);
                            // BUGFIX: Also keep inverseLines array in sync - input lines are always normal (not inverse)
                            window.RetroConsole.inverseLines.push(Array(lineText.length).fill(false));
                            remaining = remaining.substring(maxLineLength);
                        }
                    }
                    
                    // Nach dem Hinzufügen der neuen Zeilen: Prüfe, ob Scrolling erforderlich ist
                    if (typeof window.RetroConsole.checkAndScroll === 'function') {
                        window.RetroConsole.checkAndScroll();
                    }
                    
                    if (sendMessageWithSessionID(inputMessage)) {

                    }
                    
                    // Eingabe zurücksetzen NACH dem Hinzufügen zur Historie
                    window.RetroConsole.input = "";
                    window.RetroConsole.cursorPos = 0;
                    window.RetroConsole.drawTerminal();
                }
                break;            case 'Backspace':
                // Im Telnet-Modus wird Backspace bereits von handleTelnetKeyDown verarbeitet
                if (window.RetroConsole && window.RetroConsole.telnetMode) {
                    return; // Keine weitere Verarbeitung im Telnet-Modus
                }
                
                // Im RUN-Modus keine Terminal-Eingaben erlauben
                if (window.RetroConsole.runMode) {
                    return;
                }
                if (window.RetroConsole.cursorPos > 0) {
                    window.RetroConsole.input = window.RetroConsole.input.substring(0, window.RetroConsole.cursorPos - 1) +
                        window.RetroConsole.input.substring(window.RetroConsole.cursorPos);
                    window.RetroConsole.cursorPos--;
                    window.RetroConsole.drawTerminal();
                }
                break;
            case 'ArrowLeft':
                // Im RUN-Modus keine Terminal-Eingaben erlauben
                if (window.RetroConsole.runMode) {
                    return;
                }
                if (window.RetroConsole.cursorPos > 0) {
                    window.RetroConsole.cursorPos--;
                    window.RetroConsole.drawTerminal();
                }
                break;            case 'ArrowRight':
                // Im RUN-Modus keine Terminal-Eingaben erlauben
                if (window.RetroConsole.runMode) {
                    return;
                }
                if (window.RetroConsole.cursorPos < window.RetroConsole.input.length) {
                    window.RetroConsole.cursorPos++;
                    window.RetroConsole.drawTerminal();
                }
                break;            // Added default case for normal characters
            default:
                // This block is reached when inputEnabled, not runMode, not Telnet/Editor/Pager
                // and key is not Enter, Backspace, ArrowLeft/Right.
                if (event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey) {
                    // Dies ist für normale Zeicheneingabe.
                    event.preventDefault(); // Wichtig, um Standard-Browser-Aktion zu verhindern
                    const char = event.key;
                    window.RetroConsole.input = window.RetroConsole.input.substring(0, window.RetroConsole.cursorPos) +
                                               char +
                                               window.RetroConsole.input.substring(window.RetroConsole.cursorPos);
                    window.RetroConsole.cursorPos++;
                    window.RetroConsole.drawTerminal();
                }
                break;
        }
    }); // Ende des keydown Event-Listeners    // Key-Up Event Handler für INKEY$ Support
    window.addEventListener('keyup', function(event) {
        if (!window.RetroConsole || !graphicsInitialized) {
            return; 
        }
        
        // Skip in telnet mode - only keydown events are processed there
        if (window.RetroConsole && window.RetroConsole.telnetMode) {
            return;
        }
        
        // Skip in editor mode - editor handles its own key events
        if (window.RetroConsole && window.RetroConsole.editorMode) {
            return;
        }
        
        // Sende Key-Up Event für INKEY$ (auch bei deaktivierter normaler Eingabe)
        if (shouldSendKeyEvent(event.key)) {
            sendKeyEvent(17, event.key); // MessageTypeKeyUp = 17
        }
    });
}

// Handle keyboard input during filename input mode
function handleFilenameInputKeyDown(event) {
    event.preventDefault(); // Prevent default browser behavior
    
    const key = event.key;
    const ctrlKey = event.ctrlKey;
    
    // Handle special keys
    if (ctrlKey) {
        switch (key.toLowerCase()) {
            case 's':
                // Submit filename (Ctrl+S) - allow even if empty, let backend handle validation
                sendEditorCommand('filename_submit', window.RetroConsole.filenameInput || '');
                return;
            case 'enter':
                // Submit filename (Ctrl+Enter)
                sendEditorCommand('filename_submit', window.RetroConsole.filenameInput || '');
                return;
        }
    }
    
    switch (key) {
        case 'Enter':
            // Submit filename
            sendEditorCommand('filename_submit', window.RetroConsole.filenameInput || '');
            return;
        case 'Escape':
            // Cancel filename input
            sendEditorCommand('filename_cancel', '');
            return;
        case 'Backspace':
            // Remove last character
            if (window.RetroConsole.filenameInput.length > 0) {
                window.RetroConsole.filenameInput = window.RetroConsole.filenameInput.slice(0, -1);
                updateFilenameDisplay();
            }
            return;
        default:
            // Add character if it's printable
            if (key.length === 1 && !event.ctrlKey && !event.altKey) {
                window.RetroConsole.filenameInput += key;
                updateFilenameDisplay();
            }
            return;
    }
}

// Update the filename display in the status line
function updateFilenameDisplay() {
    if (window.RetroConsole.filenameInputMode) {
        window.RetroConsole.editorStatus = window.RetroConsole.filenamePrompt + window.RetroConsole.filenameInput;
        // In editor mode, call drawEditor instead of drawTerminal
        if (window.RetroConsole.editorMode && typeof window.RetroConsole.drawEditor === 'function') {
            window.RetroConsole.drawEditor();
        } else {
            window.RetroConsole.drawTerminal();
        }
    }
}

// Editor-Tastatur-Behandlung - Entfernt!
// Diese Funktion wurde entfernt, um Konflikte mit RetroConsole.handleEditorKeyDown zu vermeiden.
// Stattdessen wird die Implementierung in RetroConsole.handleEditorKeyDown verwendet.

// Editor-Kommando an Backend senden
function sendEditorCommand(command, data) {
    if (!window.ws || window.ws.readyState !== WebSocket.OPEN) {

        return;
    }
      const message = {
        type: 20, // MessageTypeEditor
        editorCommand: command,
        editorData: typeof data === 'object' ? JSON.stringify(data) : (data || ''), 
        sessionId: window.sessionId // Sicherstellen, dass sessionId hier korrekt ist
    };
    
    try {
        window.ws.send(JSON.stringify(message));
    } catch (error) {

    }
}

// Make sendEditorCommand globally available
window.sendEditorCommand = sendEditorCommand;

// Hilfsfunktion zum sicheren Zugriff auf RetroConsole
function safeRetroConsoleAccess(action) {
    if (window.RetroConsole) {
        return action();
    } else {

        // Nach kurzer Zeit erneut versuchen
        setTimeout(() => safeRetroConsoleAccess(action), 100);
        return null;
    }
}    // Hilfsfunktion, um die Session-ID konsistent abzurufen
function getSessionID() {    // Versuche SessionID aus sessionStorage oder fallback auf globale Variable
    const sessionID = window.sessionStorage ? window.sessionStorage.getItem('sessionId') : window.sessionId;
    return sessionID || ""; // Stelle sicher, dass wir nie null oder undefined zurückgeben, sondern einen leeren String
}

// Hilfsfunktion zum konsistenten Senden von Nachrichten mit Session-ID
function sendMessageWithSessionID(messageObj) {
    const ws = window.ws;

    if (ws && ws.readyState === WebSocket.OPEN) {
        // Add session ID if not present
        if (!messageObj.sessionId) {
            messageObj.sessionId = getSessionID();
        }        
        const jsonMessage = JSON.stringify(messageObj);
        
        // FRONTEND DEBUG: Log all outgoing messages
        console.log('[FRONTEND-DEBUG] Sending WebSocket message:', {
            timestamp: new Date().toISOString(),
            messageType: messageObj.type,
            contentPreview: messageObj.content ? String(messageObj.content).substring(0, 50) : 'null',
            sessionId: messageObj.sessionId,
            telnetMode: window.RetroConsole ? window.RetroConsole.telnetMode : false
        });
        
          // Prevent rapid duplicate messages in telnet mode (relaxed timing)
        if (window.RetroConsole && window.RetroConsole.telnetMode && messageObj.type === 1) {
            const now = Date.now();
            const messageHash = messageObj.content + messageObj.sessionId;
            
            if (!window.lastTelnetMessage) {
                window.lastTelnetMessage = { hash: '', time: 0 };
            }
            
            // Prevent identical messages within 10ms (very short window)
            if (window.lastTelnetMessage.hash === messageHash && 
                now - window.lastTelnetMessage.time < 10) {
                console.log('[TELNET-DUPE] Prevented duplicate message:', messageObj.content);
                return false;
            }
            
            window.lastTelnetMessage = { hash: messageHash, time: now };
        }
        
        ws.send(jsonMessage);
        return true;
    } else {
        console.error('[FRONTEND-DEBUG] WebSocket not available or not open for sending message. ReadyState:', ws ? ws.readyState : 'null');
        return false;
    }
}

// Funktion zum Senden der Terminal-Konfiguration an das Backend
function sendTerminalConfig() {
    safeRetroConsoleAccess(() => {
        if (!window.CRT_CONFIG) {
            return;
        }        // Konfiguration an das Backend senden - direkt aus CRT_CONFIG nehmen
        // WICHTIG: height = TEXT_ROWS - 1, da die letzte Zeile für die Statuszeile reserviert ist
        const config = {
            type: 8, // CONFIG
            width: window.CRT_CONFIG.TEXT_COLS,
            height: window.CRT_CONFIG.TEXT_ROWS - 1, // Eine Zeile für Statuszeile reservieren
            sessionID: getSessionID() // SessionID bei der Konfiguration mitsenden
        };        sendMessageWithSessionID(config);
    });
}

// Funktion zum Anfordern eines Gast-Tokens vom Backend
function requestGuestToken() {
    if (!guestTokenRequested) {
        guestTokenRequested = true; // Setze das Flag, um weitere Anfragen zu verhindern
        
        const tokenRequest = {
            content: "guest-token",
            type: 7 // SESSION Typ für Token-Anfrage
        };
        
        if (sendMessageWithSessionID(tokenRequest)) {

        }
    }
}

// Funktion zum Speichern des JWT-Tokens
function saveToken(token) {
    jwtToken = token;
    try {
        // In manchen Browsern kann localStorage deaktiviert sein
        localStorage.setItem('jwtToken', token);
    } catch (e) {

        // Alternativ Session-Cookie oder ähnliches verwenden
    }
}

// Funktion zum Laden des JWT-Tokens
function loadToken() {
    jwtToken = localStorage.getItem('jwtToken');
}

// Funktion zum Löschen des JWT-Tokens
function deleteToken() {
    jwtToken = null;
    localStorage.removeItem('jwtToken');
}

// Funktion für automatische Session-Wiederherstellung
function attemptSessionRecovery() {
  
    // Verhindere mehrfache gleichzeitige Recovery-Versuche
    if (window.sessionRecoveryInProgress) {
        return;
    }
    
    window.sessionRecoveryInProgress = true;
    
    // Warte kurz vor dem Wiederverbindungsversuch
    setTimeout(() => {
        try {
            // Schließe die bestehende WebSocket-Verbindung falls sie noch offen ist
            if (window.ws && window.ws.readyState === WebSocket.OPEN) {
                window.ws.close();
            }            // Lösche die alte Session-ID und JWT Token
            if (window.sessionStorage) {
                window.sessionStorage.removeItem('sessionId');
            }
            window.sessionId = null;
            
            // Reset auth manager to force re-authentication
            globalAuthManager = null;
            
              // Erstelle neue WebSocket-Verbindung
            (async () => {
                await setupWebSocket();
            })();
            
            // Markiere Recovery als abgeschlossen nach kurzer Verzögerung
            setTimeout(() => {
                window.sessionRecoveryInProgress = false;
            }, 2000);
            
        } catch (error) {
            window.sessionRecoveryInProgress = false;
            
            // Falls automatische Wiederherstellung fehlschlägt, zeige Benutzer-Nachricht
            if (window.RetroConsole && window.RetroConsole.lines) {
                window.RetroConsole.lines.push("Session expired. Please reload the page.");
                if (typeof window.RetroConsole.drawTerminal === 'function') {
                    window.RetroConsole.drawTerminal();
                }
            }
        }    }, 1000); // 1 Sekunde Verzögerung vor Wiederverbindung
}

// INKEY$ Support Funktionen
function shouldSendKeyEvent(key) {
    // Sende Events für alle relevanten Tasten (Pfeiltasten, Editor-Tasten, Escape)
    if (key === 'Escape' || 
        key === 'ArrowUp' || 
        key === 'ArrowDown' || 
        key === 'ArrowLeft' || 
        key === 'ArrowRight' ||
        key === 'Home' ||
        key === 'End' ||
        key === 'PageUp' ||
        key === 'PageDown') {
        return true;
    }
    
    // Sende auch normale Zeichen (a-z, A-Z, 0-9, Leerzeichen)
    if (key.length === 1) {
        return /[a-zA-Z0-9 ]/.test(key);
    }
    
    return false;
}

function sendKeyEvent(messageType, key) {
    if (!window.ws || window.ws.readyState !== WebSocket.OPEN) {
        return false;
    }
    
    const keyMessage = {
        type: messageType, // 16 für KeyDown, 17 für KeyUp
        key: key
    };
    
    try {
        window.ws.send(JSON.stringify(keyMessage));
        return true;
    } catch (error) {

        return false;
    }
}

// Handle keydown in pager mode - send single characters immediately without Enter
function handlePagerKeyDown(event) {
    // Prevent default behavior
    event.preventDefault();
    
    // Only handle specific keys for pager
    const key = event.key.toLowerCase();
    if (key === 'm' || key === 'q' || key === 'enter') {
        // Send immediately without waiting for Enter
        let command = key;
        if (key === 'enter') command = 'm'; // Enter acts as 'more'        // Send via WebSocket with session ID
        const message = {
            type: 1, // Normal text input (like Enter key)
            content: command
        };
        sendMessageWithSessionID(message);
    }
    // Ignore other keys in pager mode
}

// Handle keydown in telnet mode - send characters immediately and check for exit commands
function handleTelnetKeyDown(event) {
    try {
        // Initialize escape sequence state if not exists
        if (!window.telnetEscapeState) {
            window.telnetEscapeState = {
                expectingCommand: false,
                commandBuffer: "",
                lastTilde: false
            };
        }
        
        // Check for tilde escape sequence (~.)
        if (event.key === '~' && !window.telnetEscapeState.expectingCommand) {
            event.preventDefault();
            window.telnetEscapeState.lastTilde = true;
            // Don't send tilde to telnet server yet, wait for next character
            return;
        }
        
        // If we just received a tilde, check next character
        if (window.telnetEscapeState.lastTilde) {
            window.telnetEscapeState.lastTilde = false;
            
            if (event.key === '.') {
                // Tilde-dot sequence: enter local command mode
                event.preventDefault();
                window.telnetEscapeState.expectingCommand = true;
                window.telnetEscapeState.commandBuffer = "";
                
                // Show local command prompt
                if (window.RetroConsole) {
                    window.RetroConsole.lines.push("~. (Local TinyOS command - type 'help' or 'exit' to return to telnet)");
                    window.RetroConsole.inverseLines.push([]);
                    window.RetroConsole.lines.push("TinyOS> ");
                    window.RetroConsole.inverseLines.push([]);
                    window.RetroConsole.drawTerminal();
                }
                return;
            } else {
                // Not an escape sequence, send the pending tilde and continue with current key
                const tildeMessage = {
                    type: 1,
                    content: '~',
                    sessionId: window.sessionId
                };
                sendMessageWithSessionID(tildeMessage);
                // Continue processing the current key below
            }
        }
        
        // If we're in local command mode, handle commands locally
        if (window.telnetEscapeState.expectingCommand) {
            event.preventDefault();
            
            if (event.key === 'Enter') {
                // Execute local command
                const command = window.telnetEscapeState.commandBuffer.trim();
                window.telnetEscapeState.expectingCommand = false;
                window.telnetEscapeState.commandBuffer = "";
                
                if (command === 'exit' || command === 'return') {
                    // Return to telnet mode
                    if (window.RetroConsole) {
                        window.RetroConsole.lines.push("Returning to telnet session...");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.drawTerminal();
                    }
                    return;
                } else if (command === 'help') {
                    // Show help
                    if (window.RetroConsole) {
                        window.RetroConsole.lines.push("Local TinyOS commands available:");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.lines.push("  debug telnet  - Show telnet session status");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.lines.push("  exit/return  - Return to telnet session");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.lines.push("  help         - Show this help");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.lines.push("TinyOS> ");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.drawTerminal();
                    }
                    window.telnetEscapeState.expectingCommand = true;
                    return;                } else {
                    // Send command to TinyOS backend (not telnet server)
                    const localMessage = {
                        type: 0, // MessageTypeText - will be processed as normal TinyOS command
                        content: command,
                        sessionId: window.sessionId,
                        localCommand: true // Special flag to indicate this is a local command, not telnet input
                    };
                    sendMessageWithSessionID(localMessage);
                    
                    // Return to telnet mode after command
                    if (window.RetroConsole) {
                        window.RetroConsole.lines.push("Command sent to TinyOS. Returning to telnet session...");
                        window.RetroConsole.inverseLines.push([]);
                        window.RetroConsole.drawTerminal();
                    }
                    return;
                }
            } else if (event.key === 'Backspace') {
                // Handle backspace in command buffer
                if (window.telnetEscapeState.commandBuffer.length > 0) {
                    window.telnetEscapeState.commandBuffer = window.telnetEscapeState.commandBuffer.slice(0, -1);
                    
                    // Update display
                    if (window.RetroConsole && window.RetroConsole.lines.length > 0) {
                        let lastLine = window.RetroConsole.lines[window.RetroConsole.lines.length - 1];
                        if (lastLine.startsWith("TinyOS> ")) {
                            window.RetroConsole.lines[window.RetroConsole.lines.length - 1] = "TinyOS> " + window.telnetEscapeState.commandBuffer;
                            window.RetroConsole.drawTerminal();
                        }
                    }
                }
                return;
            } else if (event.key === 'Escape') {
                // Cancel local command mode
                window.telnetEscapeState.expectingCommand = false;
                window.telnetEscapeState.commandBuffer = "";
                
                if (window.RetroConsole) {
                    window.RetroConsole.lines.push("Local command cancelled. Returning to telnet session...");
                    window.RetroConsole.inverseLines.push([]);
                    window.RetroConsole.drawTerminal();
                }
                return;
            } else if (event.key.length === 1) {
                // Add character to command buffer
                window.telnetEscapeState.commandBuffer += event.key;
                
                // Update display
                if (window.RetroConsole && window.RetroConsole.lines.length > 0) {
                    let lastLine = window.RetroConsole.lines[window.RetroConsole.lines.length - 1];
                    if (lastLine.startsWith("TinyOS> ")) {
                        window.RetroConsole.lines[window.RetroConsole.lines.length - 1] = "TinyOS> " + window.telnetEscapeState.commandBuffer;
                        window.RetroConsole.drawTerminal();
                    }
                }
                return;
            }
            return; // Ignore other keys in command mode
        }
        
        // Check for exit commands
        if ((event.ctrlKey && event.key.toLowerCase() === 'x') || event.key === 'Escape') {
        event.preventDefault();
        
        // Send exit command
        let exitChar;
        if (event.ctrlKey && event.key.toLowerCase() === 'x') {
            exitChar = '\x18'; // Ctrl+X
        } else if (event.key === 'Escape') {
            exitChar = '\x1b'; // ESC
        }
        
        const message = {
            type: 1, // MessageTypeText
            content: exitChar,
            sessionId: window.sessionId
        };
        
        sendMessageWithSessionID(message);
        return;
    }
    
    // Handle other Control key combinations first
    if (event.ctrlKey && !event.altKey) {
        let ctrlChar = null;
        const key = event.key.toLowerCase();
          // Common control characters
        if (key === 'c') {
            ctrlChar = '\x03'; // Ctrl+C (SIGINT)
        } else if (key === 'd') {
            ctrlChar = '\x04'; // Ctrl+D (EOF)
        } else if (key === 'z') {
            ctrlChar = '\x1a'; // Ctrl+Z (SUSP)
        } else if (key === 'l') {
            ctrlChar = '\x0c'; // Ctrl+L (clear screen)
        } else if (key === 'u') {
            ctrlChar = '\x15'; // Ctrl+U (kill line)        } else if (key === 'k') {
            ctrlChar = '\x0b'; // Ctrl+K (kill to end)
        } else if (key === 'a') {
            ctrlChar = '\x01'; // Ctrl+A (beginning of line)
        } else if (key === 'e') {
            ctrlChar = '\x05'; // Ctrl+E (end of line)
        } else if (key === 'w') {
            ctrlChar = '\x17'; // Ctrl+W (kill word)
        } else if (key === 'r') {
            ctrlChar = '\x12'; // Ctrl+R (reverse search)
        } else if (key === 'n') {
            ctrlChar = '\x0e'; // Ctrl+N (next line)
        } else if (key === 'p') {
            ctrlChar = '\x10'; // Ctrl+P (previous line)
        } else if (key === 'f') {
            ctrlChar = '\x06'; // Ctrl+F (forward char)
        } else if (key === 'b') {
            ctrlChar = '\x02'; // Ctrl+B (backward char)
        } else if (key === 'h') {
            ctrlChar = '\x08'; // Ctrl+H (backspace)
        } else if (key === 't') {
            ctrlChar = '\x14'; // Ctrl+T (transpose chars)
        } else if (key === 'v') {
            ctrlChar = '\x16'; // Ctrl+V (literal next)
        } else if (key === 'y') {
            ctrlChar = '\x19'; // Ctrl+Y (yank)
        }
        
        if (ctrlChar !== null) {
            event.preventDefault();
            const message = {
                type: 1, // MessageTypeText
                content: ctrlChar,
                sessionId: window.sessionId
            };
            sendMessageWithSessionID(message);
            return;        }
    }
    
    // For other keys, send them to the telnet server
    let telnetInput = null;
      if (event.key === 'Enter') {
        telnetInput = '\r\n'; // RFC 854 specifies CR LF for new line
    } else if (event.key === 'Backspace') {
        telnetInput = '\x08'; // Backspace
    } else if (event.key === 'Tab') {
        telnetInput = '\t'; // Tab
    } else if (event.key === 'Delete') {
        telnetInput = '\x7f'; // Delete
    } else if (event.key === 'ArrowUp') {
        telnetInput = '\x1b[A'; // Up arrow
    } else if (event.key === 'ArrowDown') {
        telnetInput = '\x1b[B'; // Down arrow
    } else if (event.key === 'ArrowRight') {
        telnetInput = '\x1b[C'; // Right arrow
    } else if (event.key === 'ArrowLeft') {
        telnetInput = '\x1b[D'; // Left arrow
    } else if (event.key === 'Home') {
        telnetInput = '\x1b[H'; // Home
    } else if (event.key === 'End') {
        telnetInput = '\x1b[F'; // End
    } else if (event.key === 'PageUp') {
        telnetInput = '\x1b[5~'; // Page Up
    } else if (event.key === 'PageDown') {
        telnetInput = '\x1b[6~'; // Page Down
    } else if (event.key === 'Insert') {
        telnetInput = '\x1b[2~'; // Insert
    } else if (event.key.startsWith('F') && event.key.length <= 3) {
        // Function keys F1-F12
        const fNumber = parseInt(event.key.substring(1));
        if (fNumber >= 1 && fNumber <= 12) {            const fKeyCodes = {
                1: '\x1bOP', 2: '\x1bOQ', 3: '\x1bOR', 4: '\x1bOS',
                5: '\x1b[15~', 6: '\x1b[17~', 7: '\x1b[18~', 8: '\x1b[19~',
                9: '\x1b[20~', 10: '\x1b[21~', 11: '\x1b[23~', 12: '\x1b[24~'
            };
            telnetInput = fKeyCodes[fNumber] || null;
        }    } else if (event.key.length === 1) {
        // Regular printable character
        telnetInput = event.key;
    }
    
    if (telnetInput !== null) {
        event.preventDefault();        // TELNET: Send input to server
        // Handle local echo if server doesn't echo
        const shouldLocalEcho = window.RetroConsole && !window.RetroConsole.telnetServerEcho;
        
        if (shouldLocalEcho && telnetInput.length === 1 && telnetInput >= ' ' && telnetInput <= '~') {
            // Only echo printable ASCII characters locally
            // Control characters and special sequences should not be echoed locally
            window.RetroConsole.processTerminalData(telnetInput);
            window.RetroConsole.drawTerminal();
        }
        
        const message = {
            type: 1, // MessageTypeText
            content: telnetInput,
            sessionId: window.sessionId
        };
          // Send immediately without delay to maintain proper synchronization
        sendMessageWithSessionID(message);
    } else {
        // Key not supported for telnet input
    }
    } catch (error) {

    }
}

// Globale Funktionen für Editor-Interaktion (werden von RetroConsole genutzt)
// Die Funktion sendEditorCommand wurde bereits weiter oben definiert und wird hier entfernt, um den SyntaxError zu beheben。
/*
function sendEditorCommand(command, params) {
    if (window.ws && window.ws.readyState === WebSocket.OPEN) {
        const message = {
            type: 20, // MessageTypeEditor
            editorCommand: command,
            params: params,
            sessionId: getSessionID() // Session-ID hinzufügen
        };        try {
            const jsonMessage = JSON.stringify(message);
            window.ws.send(jsonMessage);
        } catch (e) {

        }
    } else {

    }
}
*/

// Die Funktionen showEditorView und showTerminalView werden entfernt,
// da die Ansicht jetzt durch RetroConsole.editorMode und RetroConsole.drawTerminal() gesteuert wird.

// Initialisierung des globalen Editor-Objekts wird entfernt.
// window.editorInstance = null; // Nicht mehr benötigt

document.addEventListener('DOMContentLoaded', () => {
    // Initialisierung der nicht-grafischen Komponenten
    initializeNonGraphics();

    // Event Listener für benutzerdefinierte Events (z.B. von RetroGraphics)
    document.addEventListener('retrographicsready', () => {
        if (!graphicsInitialized) {
            initializeGraphics();
        }
    });
});

// Globale Variable für den WebSocket (bleibt wie gehabt)
window.getSessionID = getSessionID;
window.sendTerminalConfig = sendTerminalConfig;
