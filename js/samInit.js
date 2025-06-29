/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// samInit.js - SAM Sprachausgabe vorinitialisieren
// Dieses Skript muss VOR retrosound.js geladen werden, um SAM vorzubereiten

// Dieses Skript stellt sicher, dass SAM global verfügbar ist, bevor die RestroSound-Funktionen es benötigen
(function() {
    // Globales SAM-Objekt für die S.A.M. Sprachausgabe
    window.SAM = {
        // Referenz auf den SamJs-Synthesizer
        _samInstance: null,
        
        // Initialisieren des SAM-Synthesizers
        init: function() {
            // Warten, bis SamJs geladen ist
            if (typeof window.SamJs !== 'undefined') {
                try {
                    this._samInstance = new window.SamJs({
                        pitch: 64,  // Mittlere Tonhöhe (0-255)
                        speed: 72,  // Mittlere Sprechgeschwindigkeit (0-255)
                        mouth: 128, // Mundformung (0-255)
                        throat: 128, // Kehlkopfposition (0-255)
                        singmode: false
                    });
                    return true;
                } catch (e) {

                    return false;
                }
            } else {

                return false;
            }
        },
        
        // Sprachausgabe-Funktion
        Speak: function(text) {            // SAM initialisieren, falls noch nicht geschehen
            if (!this._samInstance) {
                if (!this.init()) {

                    throw new Error("SAM konnte nicht initialisiert werden");
                }
            }
            
            try {
                return this._samInstance.speak(text);
            } catch (e) {

                throw e;
            }
        }
    };
    
    // Listener für dokumentenweite Ereignisse hinzufügen
    document.addEventListener('DOMContentLoaded', function() {
        // Versuche SamJs zu laden
        if (typeof window.SamJs === 'undefined') {
            const samScript = document.createElement('script');
            samScript.src = 'js/samjs.min.js';
            samScript.onload = function() {
                window.SAM.init();
            };
            samScript.onerror = function() {

            };
            document.head.appendChild(samScript);
        } else {
            window.SAM.init();
        }
    });
})();