/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// Spielt einen einfachen Sound mit gegebener Frequenz und Dauer ab
function playSound(freq, duration) {
    try {
        // Verwende window.RetroSound statt RetroSound, um sicherzustellen, dass wir auf das globale Objekt zugreifen
        if (!window.RetroSound.audioContext) {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }        const ctx = window.RetroSound.audioContext;
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
}

// Spielt weißes oder anderes Rauschen für eine bestimmte Dauer ab
function playNoise(type, duration) {
    try {
        if (!window.RetroSound.audioContext) {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }        const ctx = window.RetroSound.audioContext;
        if (ctx.state === 'suspended') {
            ctx.resume();
        }

        // Buffer für weißes Rauschen erstellen
        const bufferSize = ctx.sampleRate * (duration / 1000 || 0.5); // Standard 0.5 Sekunden
        const noiseBuffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate);
        const output = noiseBuffer.getChannelData(0);

        // Je nach Typ verschiedenes Rauschen generieren
        switch (type) {
            case 'white':
            default:
                // Weißes Rauschen - alle Frequenzen gleich verteilt
                for (let i = 0; i < bufferSize; i++) {
                    output[i] = Math.random() * 2 - 1;
                }
                break;
            case 'pink':
                // Rosa Rauschen - tiefere Frequenzen stärker
                let b0 = 0, b1 = 0, b2 = 0, b3 = 0, b4 = 0, b5 = 0, b6 = 0;
                for (let i = 0; i < bufferSize; i++) {
                    const white = Math.random() * 2 - 1;
                    b0 = 0.99886 * b0 + white * 0.0555179;
                    b1 = 0.99332 * b1 + white * 0.0750759;
                    b2 = 0.96900 * b2 + white * 0.1538520;
                    b3 = 0.86650 * b3 + white * 0.3104856;
                    b4 = 0.55000 * b4 + white * 0.5329522;
                    b5 = -0.7616 * b5 - white * 0.0168980;
                    output[i] = b0 + b1 + b2 + b3 + b4 + b5 + b6 + white * 0.5362;
                    output[i] *= 0.11; // Lautstärke reduzieren
                    b6 = white * 0.115926;
                }
                break;
            case 'brown':
                // Braunes Rauschen - noch tiefere Frequenzen
                let lastOut = 0;
                for (let i = 0; i < bufferSize; i++) {
                    const white = Math.random() * 2 - 1;
                    output[i] = (lastOut + (0.02 * white)) / 1.02;
                    lastOut = output[i];
                    output[i] *= 3.5; // Verstärken
                }
                break;
        }

        // Buffer Source erstellen und abspielen
        const bufferSource = ctx.createBufferSource();
        const gainNode = ctx.createGain();
        
        bufferSource.buffer = noiseBuffer;        gainNode.gain.setValueAtTime(0.1, ctx.currentTime); // Leiser als normaler Sound
        
        bufferSource.connect(gainNode);
        gainNode.connect(ctx.destination);
        
        bufferSource.start();
        
    } catch (e) {

    }
}

// retrosound.js
// Sound- und Sprachfunktionalität für das Retro-Terminal

// RetroSound-Objekt global definieren
window.RetroSound = {
    // Audio-Player für Soundeffekte
    floppyAudio: null,
    floppyUnlocked: false,
    samSpeech: null,
    audioContext: null,
    
    // Sprach-Status und Event-Handling
    isSpeaking: false,
    lastSpeechText: "",
    processingQueue: false,
    _lastSpeechTime: 0,
    speechEndCallbacks: [],
    useBrowserSpeech: false,
    
    // Neue Eigenschaften für SAY-ID Tracking
    lastSpeechID: 0,                // Letzte gesprochene ID
    pendingSpeechRequests: {},      // Speichert ausstehende Sprachanfragen nach ID
    
    // Methoden werden später gebunden
    initSpeech: null,
    speakText: null,
    playBeep: null,
    playFloppySound: null,
    unlockAudio: null,
    initAudio: null,
    isSpeechActive: null,
    onSpeechEnd: null,
    clearSpeechEndCallbacks: null,
    forceBrowserSpeech: null,
      // Neue Methoden für ID-basierte Sprachausgabe
    speakTextWithID: null,
    sendSpeechDone: null,
    getWebSocket: null, // Hinzugefügte Methode
    playSound: playSound, // playSound hier binden
    playNoise: playNoise // Hinzugefügte Methode für NOISE-Befehl
};

// SAM oder Browser Speech API initialisieren (aber immer SAM bevorzugen)
function initSpeech() {
    // useBrowserSpeech wird ignoriert - wir wollen immer SAM verwenden
    RetroSound.useBrowserSpeech = false;
    
    try {
        // Prüfen, ob das SAM.js-Script geladen wurde
        if (typeof window.SamJs === 'undefined') {

            
            // Versuche SAM.js dynamisch nachzuladen
            const samScript = document.createElement('script');
            samScript.src = 'js/samjs.min.js';
            document.head.appendChild(samScript);
            
            // Rückmeldung, dass wir es versucht haben
            if (window.RetroConsole) {
                window.RetroConsole.appendLine("SAM.js wird nachgeladen...");
            }
            
            return false;
        }
        
        // Prüfen, ob SAM bereits initialisiert wurde
        if (window.RetroSound.samSpeech !== null) {
            return true;
        }
        
        // SAM initialisieren (Optionen nur beim Instanzieren setzen!)
        window.RetroSound.samSpeech = new window.SamJs({
            pitch: 64,  // Mittlere Tonhöhe (0-255)
            speed: 72,  // Mittlere Sprechgeschwindigkeit (0-255)
            mouth: 128, // Mundformung (0-255)
            throat: 128, // Kehlkopfposition (0-255)
            singmode: false
        });
        
        return true;
    } catch (e) {

        
        // Meldung im Terminal
        if (window.RetroConsole) {
            window.RetroConsole.appendLine("Fehler bei SAM-Initialisierung. Sprachausgabe eventuell nicht verfügbar.");
        }
        
        return false;
    }
}

// Experimentelle Funktion zur Verbesserung der SAM Audio-Erkennung
function initSamAudioDetection() {
    try {
        RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();

        return true;
    } catch (e) {

        return false;
    }
}

// Hilft beim Schätzen der Sprachdauer für SAM


// Sprachausgabe mit SAM oder Browser Speech API
function speakText(text) {

    if (!text || text.trim() === '') {

        return;
    }
    
    // Verhindern von Doppelverarbeitung des gleichen Textes kurz nacheinander
    if (text === RetroSound.lastSpeechText && Date.now() - RetroSound._lastSpeechTime < 1000) {

        return;
    }
    
    // Text und Zeitpunkt merken
    RetroSound.lastSpeechText = text;
    RetroSound._lastSpeechTime = Date.now();


    // Wenn bereits eine Sprachverarbeitung läuft
    if (RetroSound.processingQueue) {

        return;
    }

    // Sperre setzen, um gleichzeitige Verarbeitung zu verhindern
    RetroSound.processingQueue = true;

    
    // Abbrechen, falls noch eine Sprachausgabe läuft
    if (RetroSound.isSpeaking) {    if (window.speechSynthesis) {
            window.speechSynthesis.cancel();
        }
        
        RetroSound.isSpeaking = false;

        
        // Kurz warten, dann neue Sprachausgabe starten
        setTimeout(() => {

            RetroSound.processingQueue = false;
            startSpeech();
        }, 100);
        return;
    }


    startSpeech();

    function startSpeech() {
        RetroSound.isSpeaking = true;
        RetroSound.processingQueue = false;
        
        // SAM initialisieren, falls nötig
        if (!RetroSound.samSpeech) {
            const samOk = initSpeech();
            if (!samOk) {

                if (window.RetroConsole) {
                    window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe konnte nicht initialisiert werden.");
                }
                RetroSound.isSpeaking = false;
                handleSpeechEnd();
                return;
            }
        }
        
        if (RetroSound.samSpeech) {
            try {
                // Text bereinigen
                let cleanText = text
                    .replace(/[^a-zA-Z0-9\s.,!?-]/g, '')
                    .replace(/\s+/g, ' ')
                    .trim();
                
                if (!cleanText) {
                    RetroSound.isSpeaking = false; // Wichtig, um den Status zurückzusetzen
                    handleSpeechEnd(); // Callbacks ausführen, falls vorhanden
                    return; // Verhindert weiteren Fehler
                }
                
                // Zahlen als Wörter aussprechen
                if (/^\d+$/.test(cleanText)) {
                    const numberWords = {
                        '0': 'zero', '1': 'one', '2': 'two', '3': 'three', '4': 'four',
                        '5': 'five', '6': 'six', '7': 'seven', '8': 'eight', '9': 'nine'
                    };
                    
                    let spokenText = '';
                    for (let i = 0; i < cleanText.length; i++) {
                        spokenText += numberWords[cleanText[i]] + ' ';
                    }
                    cleanText = spokenText.trim();
                }

                const speakPromise = RetroSound.samSpeech.speak(cleanText);

                speakPromise
                    .then(() => {
                        RetroSound.isSpeaking = false;
                        // Rufen Sie handleSpeechEnd auf, um Callbacks zu verarbeiten
                        handleSpeechEnd();
                    })
                    .catch((err) => {

                        RetroSound.isSpeaking = false;
                        // Rufen Sie handleSpeechEnd auf, um Callbacks zu verarbeiten
                        handleSpeechEnd();
                        
                        // KEIN FALLBACK zur Browser-Sprachausgabe mehr!
                        if (window.RetroConsole) {
                            window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe fehlgeschlagen.");
                        }
                    });
                return;
            } catch (e) {

                RetroSound.isSpeaking = false;
                // Rufen Sie handleSpeechEnd auf, um Callbacks zu verarbeiten
                handleSpeechEnd();
                
                // KEIN FALLBACK zur Browser-Sprachausgabe mehr!
                if (window.RetroConsole) {
                    window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe fehlgeschlagen.");
                }
                return;
            }
        }
        
        // Wenn SAM nicht verfügbar ist, wird hier keine Browser-Sprachausgabe mehr verwendet

        if (window.RetroConsole) {
            window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe nicht verfügbar.");
        }
        RetroSound.isSpeaking = false;
        handleSpeechEnd();
    }        function useBrowserSpeechAPI() {
        try {
            if (typeof SpeechSynthesisUtterance !== 'undefined') {
                 
                // Sprachsynthese abbrechen, falls aktiv
                if (window.speechSynthesis && window.speechSynthesis.speaking) {
                    window.speechSynthesis.cancel();
                }
                
                const utterance = new SpeechSynthesisUtterance(text);
                utterance.rate = 0.85;
                utterance.lang = 'en-US';

                utterance.onstart = function() {};
                utterance.onend = function() {
                    RetroSound.isSpeaking = false;
                    // Rufen Sie handleSpeechEnd auf, um Callbacks zu verarbeiten
                    handleSpeechEnd();
                };
                utterance.onerror = function(e) {
                    RetroSound.isSpeaking = false;
                    // Rufen Sie handleSpeechEnd auf, um Callbacks zu verarbeiten
                    handleSpeechEnd();
                };

                window.speechSynthesis.speak(utterance);
                return;
            }
        } catch (e) {}
        
        // Wenn alles fehlschlägt
        RetroSound.isSpeaking = false; // Sicherstellen, dass der Status korrekt ist
        handleSpeechEnd();
    }
    
    function handleSpeechEnd() {

        // handleSpeechEnd kann jetzt auch mehrfach aufgerufen werden, aber Callbacks werden nur einmal ausgeführt
        if (!RetroSound.isSpeaking && RetroSound.speechEndCallbacks.length === 0) {

            // processingQueue hier auch zurücksetzen, falls es hängen geblieben ist
            if (RetroSound.processingQueue) {

                RetroSound.processingQueue = false;
            }
            return;
        }

        RetroSound.isSpeaking = false;
        RetroSound.processingQueue = false;

        const callbacks = [...RetroSound.speechEndCallbacks];
        RetroSound.speechEndCallbacks = []; // Callbacks sofort leeren, um Mehrfachausführung zu verhindern

        setTimeout(() => {
            callbacks.forEach((cb, index) => {

                try {
                    cb();
                } catch (e) {

                }
            });

        }, 10);
    }
}

// Gibt zurück, ob aktuell eine Sprachausgabe läuft
function isSpeechActive() {
    return RetroSound.isSpeaking;
}

// Registriert einen Callback für das Ende der Sprachausgabe
function onSpeechEnd(callback) {

    if (typeof callback === 'function') {

        RetroSound.speechEndCallbacks.push(callback);
        // Wenn keine Sprachausgabe aktiv ist, führe handleSpeechEnd asynchron aus
        // um sicherzustellen, dass der Callback ausgeführt wird, wenn speakText nie gestartet wurde oder sofort fehlschlug.
        if (!RetroSound.isSpeaking) {

            setTimeout(() => {

                // Überprüfen, ob der Callback noch in der Liste ist, bevor handleSpeechEnd aufgerufen wird.
                // Dies ist wichtig, falls der Callback bereits durch einen anderen Mechanismus entfernt wurde.
                if (RetroSound.speechEndCallbacks.includes(callback)) {

                     RetroSound.handleSpeechEnd(); // Verwende die globale Version
                } else {

                }
            }, 10);
        } else {

        }
    } else {

    }
}

RetroSound.handleSpeechEnd = function() {

    if (!RetroSound.isSpeaking && RetroSound.speechEndCallbacks.length === 0) {

        if (RetroSound.processingQueue) {

            RetroSound.processingQueue = false;
        }
        return;
    }

    RetroSound.isSpeaking = false;
    RetroSound.processingQueue = false;

    const callbacks = [...RetroSound.speechEndCallbacks];
    RetroSound.speechEndCallbacks = [];

    setTimeout(() => {
        callbacks.forEach((cb, index) => {


        });

    }, 10);
}

// Löscht alle registrierten Callbacks
function clearSpeechEndCallbacks() {
    RetroSound.speechEndCallbacks = [];
}

// Erzwingt die Verwendung der Browser-Sprachausgabe
function forceBrowserSpeech(useFlag) {
    // Überschreiben - wir wollen immer SAM verwenden
    RetroSound.useBrowserSpeech = false;
    return false;
}

// Spielt einen einfachen Piepton ab
function playBeep() {

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
        oscillator.type = 'square';
        oscillator.frequency.setValueAtTime(800, ctx.currentTime);
        gainNode.gain.setValueAtTime(0.5, ctx.currentTime); // Lauter
        oscillator.connect(gainNode);
        gainNode.connect(ctx.destination);
        oscillator.start();
        oscillator.stop(ctx.currentTime + 0.2); // Länger

    } catch (e) {

    }
}

// Floppy-Sound abspielen
function playFloppySound() {
    if (window.RetroSound.floppyAudio && window.RetroSound.floppyUnlocked) {
        try {
            window.RetroSound.floppyAudio.currentTime = 0;
            window.RetroSound.floppyAudio.play().then(() => {
                // Sound erfolgreich gestartet
            }).catch(err => {

            });        } catch (e) {

        }
    } else {

        if (window.RetroSound.floppyAudio && !window.RetroSound.floppyUnlocked) {
            window.RetroSound.floppyUnlocked = true;
            playFloppySound(); // Rekursiver Aufruf
        }
    }
}

// Audio-Kontext für Browser entsperren (nach User-Interaktion)
let audioContextInitialized = false;
let audioContextInitLogged = false;
function unlockAudio() {
    if (!window.RetroSound.audioContext) {
        try {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();

        } catch (e) {

        }
    } else if (!audioContextInitLogged) {

        audioContextInitLogged = true;
    }
}

// Initialisierungscode für den Audio-Kontext
// Diese Funktion sollte einmal beim Laden der Seite aufgerufen werden
window.RetroSound.initAudio = function() {
    if (!window.RetroSound.audioContext) {
        try {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();

            // Floppy-Audio initialisieren
            if (!window.RetroSound.floppyAudio) {                window.RetroSound.floppyAudio = new Audio('/floppy.mp3');
                window.RetroSound.floppyAudio.preload = 'auto';
            }
            
            // Auto-Resume nach Benutzerinteraktion
            document.addEventListener('click', function resumeAudio() {
                if (window.RetroSound.audioContext && window.RetroSound.audioContext.state === 'suspended') {
                    window.RetroSound.audioContext.resume().then(() => {

                        window.RetroSound.floppyUnlocked = true;
                    });
                } else {
                    // Falls AudioContext bereits läuft, sofort entsperren                    window.RetroSound.floppyUnlocked = true;
                }
            }, {once: true});
        } catch (e) {

        }
    }
};

// Audio bei Seitenladung initialisieren
window.RetroSound.initAudio();

// Funktion, um die WebSocket-Instanz sicher abzurufen
function getWebSocketInstance() {

    // Priorisiere window.backendSocket, dann window.ws
    if (window.backendSocket && window.backendSocket.readyState === WebSocket.OPEN) {

        return window.backendSocket;
    }
    if (window.ws && window.ws.readyState === WebSocket.OPEN) {

        return window.ws;
    }

    return null;
}

// Sprachausgabe mit ID-basierter Rückmeldung
function speakTextWithID(text, speechID) {
    if (!text || text.trim() === '') {
        // Wichtig: Auch hier muss ggf. eine Art "Fehler" oder "sofortiges Ende" signalisiert werden,
        // damit der Backend-Interpreter nicht blockiert.
        // Sende SAY_DONE, auch wenn nichts gesprochen wird, um den Interpreter freizugeben.
        window.RetroSound.sendSpeechDone(speechID);
        return;
    }
    
    // ID speichern
    window.RetroSound.lastSpeechID = speechID;
    window.RetroSound.pendingSpeechRequests[speechID] = true;
    
    // AUSSCHLIESSLICH SAM verwenden, nie auf Browser-Sprachausgabe zurückfallen
    
    // SAM initialisieren, falls nötig
    if (!window.RetroSound.samSpeech) {
        initSpeech();
    }
      if (window.RetroSound.samSpeech && typeof window.SamJs !== "undefined") {
        try {
            // Text für SAM vorbereiten
            let cleanText = text
                .replace(/[^a-zA-Z0-9\s.,!?-]/g, '')
                .replace(/\s+/g, ' ')
                .trim();
                
            if (!cleanText) {
                window.RetroSound.sendSpeechDone(speechID);
                return;
            }
            
            // Zahlen als Wörter aussprechen
            if (/^\d+$/.test(cleanText)) {
                const numberWords = {
                    '0': 'zero', '1': 'one', '2': 'two', '3': 'three', '4': 'four',
                    '5': 'five', '6': 'six', '7': 'seven', '8': 'eight', '9': 'nine'
                };
                
                let spokenText = '';
                for (let i = 0; i < cleanText.length; i++) {
                    spokenText += numberWords[cleanText[i]] + ' ';
                }
                cleanText = spokenText.trim();
            }
              // SAM-Instanz verwenden mit Promise

            const speakPromise = window.RetroSound.samSpeech.speak(cleanText);
              // Sicherstellen, dass es sich um ein Promise handelt
            if (speakPromise && typeof speakPromise.then === 'function') {
                speakPromise
                    .then(() => {
                        if (window.RetroSound.pendingSpeechRequests[speechID]) {
                            window.RetroSound.sendSpeechDone(speechID);
                            delete window.RetroSound.pendingSpeechRequests[speechID];
                        }
                    })
                    .catch((err) => {

                        if (window.RetroSound.pendingSpeechRequests[speechID]) {
                            window.RetroSound.sendSpeechDone(speechID);
                            delete window.RetroSound.pendingSpeechRequests[speechID];
                        }
                    });
            } else {
                // Bei SAM können wir nur ungefähr schätzen, wann die Ausgabe fertig ist
                const estimatedDuration = cleanText.length * 100; // ca. 100ms pro Zeichen
                setTimeout(() => {
                    if (window.RetroSound.pendingSpeechRequests[speechID]) {
                        window.RetroSound.sendSpeechDone(speechID);
                        delete window.RetroSound.pendingSpeechRequests[speechID];
                    }
                }, estimatedDuration);
            }
              } catch (e) {

            // KEIN Fallback zur Browser-Sprachausgabe, stattdessen Fehlermeldung
            if (window.RetroConsole) {
                window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe fehlgeschlagen.");
            }
            window.RetroSound.sendSpeechDone(speechID);
            delete window.RetroSound.pendingSpeechRequests[speechID];
        }
    } else {

        // Kein Fallback zur Browser-Sprachausgabe, stattdessen Fehlermeldung
        if (window.RetroConsole) {
            window.RetroConsole.appendLine("ERROR: SAM Sprachausgabe nicht verfügbar.");
        }
        
        // Script-Tag für SAM.js dynamisch nachladen als letzter Versuch
        const samScript = document.createElement('script');
        samScript.src = 'js/samjs.min.js';
        samScript.onload = function() {
            if (window.RetroConsole) {
                window.RetroConsole.appendLine("SAM.js nachgeladen. Versuchen Sie erneut zu sprechen.");
            }
            window.RetroSound.sendSpeechDone(speechID);
            delete window.RetroSound.pendingSpeechRequests[speechID];
        };
        samScript.onerror = function() {
            if (window.RetroConsole) {
                window.RetroConsole.appendLine("FEHLER: SAM.js konnte nicht geladen werden!");
            }
            window.RetroSound.sendSpeechDone(speechID);
            delete window.RetroSound.pendingSpeechRequests[speechID];
        };
        document.head.appendChild(samScript);
    }
}

// Informiert das Backend, dass die Sprachausgabe beendet ist
function sendSpeechDone(speechID) {

    const ws = getWebSocketInstance(); // WebSocket-Instanz abrufen

    if (ws && ws.readyState === WebSocket.OPEN) {
        const message = {
            type: 6, // SAY_DONE
            speechId: speechID,
            // SessionID wird serverseitig aus der Verbindung extrahiert
        };


        ws.send(JSON.stringify(message));
    } else {
        const wsStatus = ws ? ws.readyState : 'nicht vorhanden';

    }
}

// Funktion zur Erzeugung von Rausch-Effekten mit Web Audio API
function playNoise(pitch, attack, decay) {

    
    // Stellen Sie sicher, dass der Audio-Context initialisiert ist
    if (!window.AudioContext && !window.webkitAudioContext) {

        return;
    }
    
    // AudioContext aus der übergeordneten Scope verwenden, falls vorhanden, oder neu erstellen
    if (!window.RetroSound.audioContext) {
        window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();

    }
    
    // Lokale Referenz für einfacheren Zugriff
    const audioContext = window.RetroSound.audioContext;
    
    // Audio-Context fortsetzen, falls unterbrochen
    if (audioContext.state === 'suspended') {
        audioContext.resume();

    }
      // Der audioContext ist jetzt eine lokale Referenz zur globalen Instanz

    // Puffer für weißes Rauschen erstellen
    const bufferSize = audioContext.sampleRate * 2; // 2 Sekunden Puffer

    const noiseBuffer = audioContext.createBuffer(1, bufferSize, audioContext.sampleRate);
    
    // Weißes Rauschen generieren
    const outputData = noiseBuffer.getChannelData(0);
    for (let i = 0; i < bufferSize; i++) {
        outputData[i] = Math.random() * 2 - 1;
    }
    
    // Noise-Quelle erstellen
    const noiseSource = audioContext.createBufferSource();
    noiseSource.buffer = noiseBuffer;
    
    // Filter erstellen, um die Tonhöhe zu beeinflussen
    const filter = audioContext.createBiquadFilter();
    filter.type = 'bandpass';
    
    // Pitch-Wert in Frequenz umwandeln (0-255 -> 50-5000Hz logarithmisch)
    const minFreq = 50;
    const maxFreq = 5000;
    const frequencyValue = minFreq * Math.pow(maxFreq/minFreq, pitch/255);
    filter.frequency.value = frequencyValue;
    filter.Q.value = 1.0; // Resonanz für deutlicheren Effekt
    
    // Gain-Node für die Lautstärkekontrolle
    const gainNode = audioContext.createGain();
    gainNode.gain.value = 0;
    
    // Attack- und Decay-Zeiten berechnen (0-255 -> 0-1500ms)
    const attackTime = attack * 6; // max 1.5 Sekunden
    const decayTime = decay * 6;   // max 1.5 Sekunden
    
    // Aktuellen Zeitpunkt im Audio-Context abrufen
    const currentTime = audioContext.currentTime;
    
    // Envelope anwenden (Attack und Decay)
    gainNode.gain.setValueAtTime(0, currentTime);
    gainNode.gain.linearRampToValueAtTime(0.7, currentTime + attackTime/1000); // Maximal 70% Lautstärke
    gainNode.gain.linearRampToValueAtTime(0, currentTime + attackTime/1000 + decayTime/1000);
    
    // Nodes verbinden
    noiseSource.connect(filter);
    filter.connect(gainNode);
    gainNode.connect(audioContext.destination);
      // Sound abspielen
    try {
        noiseSource.start();
        noiseSource.stop(currentTime + attackTime/1000 + decayTime/1000 + 0.1); // Zusätzliche Zeit für sauberes Ausblenden

    } catch (e) {

    }
};

// Einfache playNoise-Funktion für Backend-Kompatibilität (type, duration)
function playNoiseSimple(type, duration) {

    
    // Standardwerte für die komplexere playNoise-Funktion
    const pitch = 128;  // Mittlere Tonhöhe
    const attack = 10;  // Schneller Attack
    const decay = Math.min(255, duration / 10); // Decay basierend auf Dauer
    
    try {
        // Rufe die bestehende playNoise-Funktion mit Standardwerten auf
        playNoise(pitch, attack, decay);
    } catch (e) {

    }
}

// Komplexe playNoise-Funktion für NOISE-Befehl (pitch, attack, decay)
function playNoiseComplex(pitch, attack, decay) {
    try {
        if (!window.RetroSound.audioContext) {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }
        const ctx = window.RetroSound.audioContext;
        
        if (ctx.state === 'suspended') {
            ctx.resume().then(() => {
                playNoiseComplexInternal(ctx, pitch, attack, decay);
            }).catch(err => {

            });
        } else {
            playNoiseComplexInternal(ctx, pitch, attack, decay);
        }
        
    } catch (e) {

    }
}

function playNoiseComplexInternal(ctx, pitch, attack, decay) {
    try {
        // Parameter validieren und normalisieren
        const normalizedPitch = Math.max(1, Math.min(255, pitch || 128)); // 1-255
        const normalizedAttack = Math.max(1, Math.min(255, attack || 10)); // 1-255
        const normalizedDecay = Math.max(1, Math.min(255, decay || 50)); // 1-255
        
        // Berechne Dauer basierend auf attack + decay (in Millisekunden)
        const duration = (normalizedAttack + normalizedDecay) * 10; // Faktor 10 für hörbare Dauer
        
        // Buffer für Noise erstellen
        const bufferSize = ctx.sampleRate * (duration / 1000);
        const noiseBuffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate);
        const output = noiseBuffer.getChannelData(0);

        // Noise basierend auf Pitch generieren
        const baseFreq = 55 + (normalizedPitch / 255) * 1000; // 55Hz bis 1055Hz
        
        for (let i = 0; i < bufferSize; i++) {
            // Erzeuge gefiltertes Rauschen basierend auf Pitch
            const white = Math.random() * 2 - 1;
            const time = i / ctx.sampleRate;
            
            // Attack und Decay Envelope
            let envelope = 1.0;
            const attackTime = (normalizedAttack / 255) * (duration / 2000); // Attack Phase
            const decayTime = (normalizedDecay / 255) * (duration / 1000); // Decay Phase
            
            if (time < attackTime) {
                // Attack Phase - linear ansteigend
                envelope = time / attackTime;
            } else {
                // Decay Phase - exponentiell abfallend
                const decayProgress = (time - attackTime) / decayTime;
                envelope = Math.exp(-decayProgress * 3); // Exponentieller Abfall
            }
            
            // Frequenz-basierte Filterung
            const freqModulation = Math.sin(time * baseFreq * 2 * Math.PI) * 0.3;
            output[i] = (white + freqModulation) * envelope * 0.3; // Lautstärke begrenzen
        }

        // Buffer Source erstellen und abspielen
        const bufferSource = ctx.createBufferSource();
        const gainNode = ctx.createGain();
        
        bufferSource.buffer = noiseBuffer;
        gainNode.gain.setValueAtTime(0.2, ctx.currentTime); // Mittlere Lautstärke
          bufferSource.connect(gainNode);
        gainNode.connect(ctx.destination);
        
        bufferSource.start();
        
    } catch (e) {

    }
}

// Methoden an das globale RetroSound-Objekt binden
window.RetroSound.initSpeech = initSpeech;
window.RetroSound.speakText = speakText;
window.RetroSound.playBeep = playBeep;
window.RetroSound.playFloppySound = playFloppySound;
window.RetroSound.unlockAudio = unlockAudio;
window.RetroSound.initAudio = window.RetroSound.initAudio; // Verwende bereits definierte Funktion
window.RetroSound.playSound = playSound;
window.RetroSound.isSpeechActive = isSpeechActive;
window.RetroSound.onSpeechEnd = onSpeechEnd;
window.RetroSound.clearSpeechEndCallbacks = clearSpeechEndCallbacks;
window.RetroSound.forceBrowserSpeech = forceBrowserSpeech;
window.RetroSound.speakTextWithID = speakTextWithID;
window.RetroSound.sendSpeechDone = sendSpeechDone;
window.RetroSound.getWebSocket = getWebSocketInstance; // Binden der neuen Hilfsfunktion
window.RetroSound.playNoise = playNoiseSimple; // Verwende die einfache Version für Backend-Kompatibilität
window.RetroSound.playNoiseComplex = playNoiseComplex; // Verwende die richtige komplexe Version
window.RetroSound.playNoiseSimple = playNoiseSimple;

// SID Music Player Funktionen binden
window.RetroSound.openSidMusic = openSidMusic;
window.RetroSound.playSidMusic = playSidMusic;
window.RetroSound.stopSidMusic = stopSidMusic;
window.RetroSound.pauseSidMusic = pauseSidMusic;

// Globaler Event-Listener für Audio-Kontext-Resume
document.addEventListener('click', function() {
    if (window.RetroSound && window.RetroSound.audioContext && 
        window.RetroSound.audioContext.state === 'suspended') {
        window.RetroSound.audioContext.resume().then(() => {

        }).catch(err => {

        });
    }
});

document.addEventListener('keydown', function() {
    if (window.RetroSound && window.RetroSound.audioContext && 
        window.RetroSound.audioContext.state === 'suspended') {
        window.RetroSound.audioContext.resume().then(() => {

        }).catch(err => {

        });
    }
});

// SID Music Player Implementation
// Diese Implementation bietet grundlegende SID-Musikwiedergabe

// Globale Variablen für SID-Player
window.RetroSound.sidPlayer = {
    currentSid: null,
    isPlaying: false,
    isPaused: false,
    audioSource: null,
    gainNode: null
};

// SID-Datei öffnen/laden
function openSidMusic(filename) {
    try {

        
        // Verwende sidPlayer direkt
        if (window.sidPlayer && typeof window.sidPlayer.loadSID === 'function') {
            const sidPath = `/api/file?path=${encodeURIComponent(filename)}`;
            window.sidPlayer.loadSID(sidPath, filename);

        } else {

            // Fallback zu alter Implementierung
            openSidMusicLegacy(filename);
        }
            
    } catch (error) {

        // Fallback zu alter Implementierung
        openSidMusicLegacy(filename);
    }
}

// Legacy SID-Datei öffnen/laden (Fallback)
function openSidMusicLegacy(filename) {
    try {
        
        // Initialisiere Audio-Kontext falls nötig
        if (!window.RetroSound.audioContext) {
            window.RetroSound.audioContext = new (window.AudioContext || window.webkitAudioContext)();
        }
        
        const ctx = window.RetroSound.audioContext;
        if (ctx.state === 'suspended') {
            ctx.resume();
        }

        // Stoppe aktuelle Musik falls vorhanden
        stopSidMusic();

        // Lade SID-Datei vom Server
        const sidPath = `/api/file?path=${encodeURIComponent(filename)}`;
        
        fetch(sidPath)
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }
                return response.arrayBuffer();
            })
            .then(arrayBuffer => {
                // Vereinfachte SID-Verarbeitung - konvertiere zu Audio
                // Da echte SID-Emulation komplex ist, simulieren wir hier den Sound
                return processSidFile(arrayBuffer);
            })
            .then(audioBuffer => {
                window.RetroSound.sidPlayer.currentSid = audioBuffer;

            })
            .catch(error => {

                // Fallback: Generiere Platzhalter-Audio
                generateFallbackSidAudio(filename);
            });
            
    } catch (error) {

        generateFallbackSidAudio(filename);
    }
}

// Vereinfachte SID-Datei Verarbeitung
function processSidFile(arrayBuffer) {
    return new Promise((resolve, reject) => {
        try {
            const ctx = window.RetroSound.audioContext;
            
            // Da echte SID-Emulation sehr komplex ist, erstellen wir eine 
            // vereinfachte Interpretation basierend auf den SID-Daten
            const sidData = new Uint8Array(arrayBuffer);
            
            // Einfache Analyse der SID-Header-Informationen
            const magicBytes = String.fromCharCode(...sidData.slice(0, 4));
            
            if (magicBytes === 'PSID' || magicBytes === 'RSID') {
                // Gültige SID-Datei erkannt

                
                // Generiere Audio basierend auf SID-Daten
                const audioBuffer = generateSidAudio(sidData, ctx);
                resolve(audioBuffer);
            } else {
                // Keine gültige SID-Datei, generiere Fallback

                const audioBuffer = generateFallbackAudio(ctx);
                resolve(audioBuffer);
            }
        } catch (error) {
            reject(error);
        }
    });
}

// Generiere Audio basierend auf SID-Daten (vereinfacht)
function generateSidAudio(sidData, audioContext) {
    const duration = 30; // 30 Sekunden Beispiel-Musik
    const sampleRate = audioContext.sampleRate;
    const length = sampleRate * duration;
    
    const audioBuffer = audioContext.createBuffer(1, length, sampleRate);
    const channelData = audioBuffer.getChannelData(0);
    
    // Einfache Tonfolge basierend auf SID-Daten
    let frequency = 440; // Grundfrequenz
    let phase = 0;
    
    // Verwende Bytes aus der SID-Datei für Frequenzmodulation
    for (let i = 0; i < length; i++) {
        const time = i / sampleRate;
        const dataIndex = Math.floor((i / length) * (sidData.length - 0x7C)) + 0x7C; // Skip Header
        
        if (dataIndex < sidData.length) {
            // Moduliere Frequenz basierend auf SID-Daten
            const dataValue = sidData[dataIndex];
            frequency = 220 + (dataValue * 2); // Frequenzbereich 220-730 Hz
        }
        
        // Erzeuge Rechteckwelle (typisch für SID)
        const squareWave = Math.sin(phase) > 0 ? 0.3 : -0.3;
        
        // Addiere harmonische für reicheren Sound
        const harmonic1 = Math.sin(phase * 2) * 0.1;
        const harmonic2 = Math.sin(phase * 3) * 0.05;
        
        channelData[i] = squareWave + harmonic1 + harmonic2;
        
        // Aktualisiere Phase
        phase += (2 * Math.PI * frequency) / sampleRate;
        if (phase >= 2 * Math.PI) phase -= 2 * Math.PI;
        
        // Variiere Frequenz über Zeit für musikalischen Effekt
        if (i % (sampleRate * 0.5) === 0) { // Alle 0.5 Sekunden
            frequency *= 1.05; // Leichte Frequenzsteigerung
            if (frequency > 800) frequency = 220; // Reset bei zu hoch
        }
    }
    
    return audioBuffer;
}

// Fallback-Audio wenn SID-Verarbeitung fehlschlägt
function generateFallbackAudio(audioContext) {
    const duration = 10; // 10 Sekunden
    const sampleRate = audioContext.sampleRate;
    const length = sampleRate * duration;
    
    const audioBuffer = audioContext.createBuffer(1, length, sampleRate);
    const channelData = audioBuffer.getChannelData(0);
    
    // Einfache C64-artige Melodie
    const notes = [261.63, 293.66, 329.63, 349.23, 392.00, 440.00, 493.88, 523.25]; // C-Dur Tonleiter
    let noteIndex = 0;
    let phase = 0;
    
    for (let i = 0; i < length; i++) {
        const frequency = notes[noteIndex];
        
        // Rechteckwelle
        const squareWave = Math.sin(phase) > 0 ? 0.2 : -0.2;
        channelData[i] = squareWave;
        
        phase += (2 * Math.PI * frequency) / sampleRate;
        if (phase >= 2 * Math.PI) phase -= 2 * Math.PI;
        
        // Wechsle Note alle 1.25 Sekunden
        if (i % Math.floor(sampleRate * 1.25) === 0) {
            noteIndex = (noteIndex + 1) % notes.length;
        }
    }
    
    return audioBuffer;
}

// Fallback für Ladefehler
function generateFallbackSidAudio(filename) {
    try {
        const ctx = window.RetroSound.audioContext;
        const audioBuffer = generateFallbackAudio(ctx);
        window.RetroSound.sidPlayer.currentSid = audioBuffer;

    } catch (error) {

    }
}

// SID-Musik abspielen
function playSidMusic() {
    try {
        // Verwende sidPlayer direkt
        if (window.sidPlayer && typeof window.sidPlayer.play === 'function') {
            window.sidPlayer.play();

            return;
        }
        
        // Fallback zu legacy Implementierung
        if (!window.RetroSound.sidPlayer.currentSid) {

            return;
        }

        const ctx = window.RetroSound.audioContext;
        if (ctx.state === 'suspended') {
            ctx.resume();
        }

        // Stoppe aktuell laufende Musik
        if (window.RetroSound.sidPlayer.audioSource) {
            window.RetroSound.sidPlayer.audioSource.stop();
        }

        // Erstelle neue Audio-Source
        const source = ctx.createBufferSource();
        source.buffer = window.RetroSound.sidPlayer.currentSid;
        
        // Erstelle Gain-Node für Lautstärkekontrolle
        const gainNode = ctx.createGain();
        gainNode.gain.setValueAtTime(0.5, ctx.currentTime); // 50% Lautstärke
        
        // Verbinde Audio-Graph
        source.connect(gainNode);
        gainNode.connect(ctx.destination);
        
        // Setze Loop für kontinuierliche Wiedergabe
        source.loop = true;
        
        // Starte Wiedergabe
        source.start();
        
        // Speichere Referenzen
        window.RetroSound.sidPlayer.audioSource = source;
        window.RetroSound.sidPlayer.gainNode = gainNode;
        window.RetroSound.sidPlayer.isPlaying = true;
        window.RetroSound.sidPlayer.isPaused = false;
        

        
    } catch (error) {

    }
}

// SID-Musik stoppen
function stopSidMusic() {
    try {
        // Verwende sidPlayer direkt
        if (window.sidPlayer && typeof window.sidPlayer.stop === 'function') {
            window.sidPlayer.stop();

            return;
        }
        
        // Legacy Implementierung
        if (window.RetroSound.sidPlayer.audioSource) {
            window.RetroSound.sidPlayer.audioSource.stop();
            window.RetroSound.sidPlayer.audioSource = null;
        }
        
        window.RetroSound.sidPlayer.gainNode = null;
        window.RetroSound.sidPlayer.isPlaying = false;
        window.RetroSound.sidPlayer.isPaused = false;
        

        
    } catch (error) {

    }
}

// SID-Musik pausieren
function pauseSidMusic() {
    try {
        // Verwende sidPlayer direkt
        if (window.sidPlayer && typeof window.sidPlayer.pause === 'function') {
            window.sidPlayer.pause();

            return;
        }
        
        // Legacy Implementierung
        if (window.RetroSound.sidPlayer.isPlaying && !window.RetroSound.sidPlayer.isPaused) {
            if (window.RetroSound.sidPlayer.gainNode) {
                // Fade out für sanftes Pausieren
                const ctx = window.RetroSound.audioContext;
                window.RetroSound.sidPlayer.gainNode.gain.linearRampToValueAtTime(0, ctx.currentTime + 0.1);
                
                setTimeout(() => {
                    if (window.RetroSound.sidPlayer.audioSource) {
                        window.RetroSound.sidPlayer.audioSource.stop();
                        window.RetroSound.sidPlayer.audioSource = null;
                    }
                    window.RetroSound.sidPlayer.isPaused = true;
                    window.RetroSound.sidPlayer.isPlaying = false;

                }, 100);
            }
        } else if (window.RetroSound.sidPlayer.isPaused) {
            // Resume from pause
            playSidMusic();

        }        
    } catch (error) {

    }
}