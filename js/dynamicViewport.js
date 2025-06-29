/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// dynamicViewport.js - Automatische Viewport-Anpassung für alle Bildschirmauflösungen
// Diese Datei muss als erstes geladen werden, bevor andere Module initialisiert werden

(function() {
    'use strict';    // Globaler Namespace für Viewport-Management
    window.DynamicViewport = {
        // Aktuelle Viewport-Informationen
        current: {
            actualWidth: 0,
            actualHeight: 0,
            canvasWidth: 640,   // BASIC-Koordinaten bleiben unverändert
            canvasHeight: 480,
            textCols: 80,
            textRows: 24,
            scaleX: 1,
            scaleY: 1,
            offsetX: 0,
            offsetY: 0,
            padding: { left: 0, top: 0, right: 0, bottom: 0 }
        },
        
        // Callbacks die aufgerufen werden wenn sich das Viewport ändert
        onResizeCallbacks: [],
        
        // Registriere Callback für Viewport-Änderungen
        onResize: function(callback) {
            if (typeof callback === 'function') {
                this.onResizeCallbacks.push(callback);
            }
        },
        
        // Berechne optimale Skalierung und Padding für den aktuellen Viewport
        calculateOptimalLayout: function() {
            const viewport = this.detectViewportSize();
            
            // Basis-Canvas-Größen aus window.CONFIG lesen
            // Fallback auf Standardwerte, falls CONFIG nicht verfügbar oder Werte fehlen (sollte nicht passieren)
            const baseCanvasW = (window.CONFIG && window.CONFIG.VIRTUAL_CRT_WIDTH) ? window.CONFIG.VIRTUAL_CRT_WIDTH : 760;
            const baseCanvasH = (window.CONFIG && window.CONFIG.VIRTUAL_CRT_HEIGHT) ? window.CONFIG.VIRTUAL_CRT_HEIGHT : 560;
            
            // Berechne verfügbaren Platz (mit Mindestablstand zum Rand)
            const minMargin = Math.min(viewport.width * 0.05, viewport.height * 0.05); // 5% Mindestrand
            const availableW = viewport.width - (minMargin * 2);
            const availableH = viewport.height - (minMargin * 2);
            
            // Berechne Skalierungsfaktoren
            const scaleX = availableW / baseCanvasW;
            const scaleY = availableH / baseCanvasH;
            
            // Verwende einheitliche Skalierung (kleinerer Wert) um Seitenverhältnis zu erhalten
            const uniformScale = Math.min(scaleX, scaleY);
            
            // Berechne tatsächliche Canvas-Größe nach Skalierung
            const scaledCanvasW = baseCanvasW * uniformScale;
            const scaledCanvasH = baseCanvasH * uniformScale;
            
            // Berechne Zentrierung (überschüssiger Platz wird gleichmäßig verteilt)
            const offsetX = (viewport.width - scaledCanvasW) / 2;
            const offsetY = (viewport.height - scaledCanvasH) / 2;
            
            // Behalte die ursprüngliche Berechnung für this.current.padding,
            // die für die internen Metriken von dynamicViewport verwendet wird (z.B. getTextMetrics).
            // Diese basiert auf scaledCanvasW/H und ist symmetrisch.
            const textPaddingPercent = 0.15; 
            const dvInternalTextPaddingX = scaledCanvasW * textPaddingPercent / 2;
            const dvInternalTextPaddingY = scaledCanvasH * textPaddingPercent / 2;
            
            // Aktualisiere aktuelle Werte in this.current
            this.current = {
                actualWidth: viewport.width,
                actualHeight: viewport.height,
                canvasWidth: baseCanvasW,  // Verwende die dynamisch geladenen Basiswerte
                canvasHeight: baseCanvasH, // Verwende die dynamisch geladenen Basiswerte
                textCols: (window.CONFIG && window.CONFIG.TEXT_COLS) ? window.CONFIG.TEXT_COLS : 80, // Hole TEXT_COLS aus CONFIG
                textRows: (window.CONFIG && window.CONFIG.TEXT_ROWS) ? window.CONFIG.TEXT_ROWS : 24, // Hole TEXT_ROWS aus CONFIG
                scaleX: uniformScale,
                scaleY: uniformScale,
                offsetX: offsetX,
                offsetY: offsetY,
                scaledCanvasW: scaledCanvasW,
                scaledCanvasH: scaledCanvasH,
                padding: { // Dies ist this.current.padding, symmetrisch für interne Zwecke
                    left: dvInternalTextPaddingX,
                    top: dvInternalTextPaddingY,
                    right: dvInternalTextPaddingX,
                    bottom: dvInternalTextPaddingY
                }
            };

            // Nun berechne die Paddings für window.CONFIG, die von retroconsole.js verwendet werden.
            // Diese basieren auf VIRTUAL_CRT_WIDTH/HEIGHT aus window.CONFIG.
            // Stelle sicher, dass virtualCrtWidth und virtualCrtHeight direkt die Werte aus CONFIG verwenden.
            const virtualCrtWidth = baseCanvasW; 
            const virtualCrtHeight = baseCanvasH;
            const textCols = this.current.textCols;

            // --- PADDING-BERECHNUNG --- 
            // Standard-Fallback-Werte, falls nichts in config.js definiert ist oder die Werte ungültig sind.
            const defaultPaddingPercent = 0.15; // Beibehaltung des prozentualen Fallbacks
            let cfgPaddingLeft, cfgPaddingTop, cfgPaddingRight, cfgPaddingBottom;

            // Prüfe, ob manuelle Korrekturwerte in window.CONFIG vorhanden und gültig sind.
            const manualLeft = window.CONFIG && typeof window.CONFIG.SCREEN_PADDING_LEFT === 'number' && window.CONFIG.SCREEN_PADDING_LEFT >= 0;
            const manualTop = window.CONFIG && typeof window.CONFIG.SCREEN_PADDING_TOP === 'number' && window.CONFIG.SCREEN_PADDING_TOP >= 0;
            const manualRight = window.CONFIG && typeof window.CONFIG.SCREEN_PADDING_RIGHT === 'number' && window.CONFIG.SCREEN_PADDING_RIGHT >= 0;
            const manualBottom = window.CONFIG && typeof window.CONFIG.SCREEN_PADDING_BOTTOM === 'number' && window.CONFIG.SCREEN_PADDING_BOTTOM >= 0;            if (manualLeft) {
                cfgPaddingLeft = window.CONFIG.SCREEN_PADDING_LEFT;
            } else {
                // Dynamische Berechnung für linkes Padding (eine Zeichenbreite), falls nicht manuell gesetzt
                // Diese Berechnung benötigt cfgPaddingRight, daher muss es vorher ggf. dynamisch berechnet werden.
                const tempCfgPaddingRight = (manualRight) ? window.CONFIG.SCREEN_PADDING_RIGHT : virtualCrtWidth * (defaultPaddingPercent / 2);
                if (textCols > 0) {
                    cfgPaddingLeft = (virtualCrtWidth - tempCfgPaddingRight) / (textCols + 1); 
                } else {
                    cfgPaddingLeft = virtualCrtWidth * (defaultPaddingPercent / 2);
                }
            }            if (manualTop) {
                cfgPaddingTop = window.CONFIG.SCREEN_PADDING_TOP;
            } else {
                cfgPaddingTop = virtualCrtHeight * (defaultPaddingPercent / 2);
            }            if (manualRight) {
                cfgPaddingRight = window.CONFIG.SCREEN_PADDING_RIGHT;
            } else {
                cfgPaddingRight = virtualCrtWidth * (defaultPaddingPercent / 2);
            }            if (manualBottom) {
                cfgPaddingBottom = window.CONFIG.SCREEN_PADDING_BOTTOM;
            } else {
                cfgPaddingBottom = virtualCrtHeight * (defaultPaddingPercent / 2);
            }
            
            // Update window.CONFIG mit den (potenziell manuell korrigierten oder dynamisch berechneten) Padding-Werten
            if (window.CONFIG) {
                window.CONFIG.SCREEN_PADDING_LEFT = cfgPaddingLeft;
                window.CONFIG.SCREEN_PADDING_TOP = cfgPaddingTop;
                window.CONFIG.SCREEN_PADDING_RIGHT = cfgPaddingRight;
                window.CONFIG.SCREEN_PADDING_BOTTOM = cfgPaddingBottom;

            } 
            

            
            return this.current;
        },
        
        // Erkenne echte Viewport-Größe
        detectViewportSize: function() {
            // Verschiedene Methoden zur Viewport-Erkennung
            const methods = [
                // Methode 1: Standard-Viewport
                { w: window.innerWidth, h: window.innerHeight, method: 'innerWindow' },
                // Methode 2: Dokumenten-Client-Bereich
                { w: document.documentElement.clientWidth, h: document.documentElement.clientHeight, method: 'documentClient' },
                // Methode 3: Bildschirm-verfügbare Größe
                { w: screen.availWidth, h: screen.availHeight, method: 'screenAvailable' }
            ];
            
            // Wähle die sinnvollste Methode (normalerweise innerWindow)
            let best = methods[0];
            
            // Validiere und nutze die erste verfügbare Methode mit vernünftigen Werten
            for (const method of methods) {
                if (method.w > 200 && method.h > 200 && method.w < 10000 && method.h < 10000) {
                    best = method;
                    break;
                }
            }
                        
            return { width: best.w, height: best.h };
        },
        
        // Konvertiere BASIC-Koordinaten zu echten Bildschirm-Koordinaten
        basicToScreen: function(basicX, basicY) {
            const c = this.current;
            const screenX = c.offsetX + (basicX / c.canvasWidth) * c.scaledCanvasW;
            const screenY = c.offsetY + (basicY / c.canvasHeight) * c.scaledCanvasH;
            return { x: screenX, y: screenY };
        },
        
        // Konvertiere Bildschirm-Koordinaten zu BASIC-Koordinaten
        screenToBasic: function(screenX, screenY) {
            const c = this.current;
            const basicX = ((screenX - c.offsetX) / c.scaledCanvasW) * c.canvasWidth;
            const basicY = ((screenY - c.offsetY) / c.scaledCanvasH) * c.canvasHeight;
            return { x: basicX, y: basicY };
        },
        
        // Berechne Zeichenmetriken für Text
        getTextMetrics: function() {
            const c = this.current;
            
            // Verfügbarer Textbereich (Canvas minus Padding)
            const textAreaW = c.scaledCanvasW - c.padding.left - c.padding.right;
            const textAreaH = c.scaledCanvasH - c.padding.top - c.padding.bottom;
            
            // Zeichendimensionen
            const charW = textAreaW / c.textCols;
            const charH = textAreaH / c.textRows;
            
            // Schriftgröße basierend auf Zeichenhöhe
            const fontSize = Math.max(8, Math.min(charH * 0.8, 24)); // Zwischen 8px und 24px
            
            return {
                charWidth: charW,
                charHeight: charH,
                fontSize: fontSize,
                textStartX: c.offsetX + c.padding.left,
                textStartY: c.offsetY + c.padding.top,
                textAreaWidth: textAreaW,
                textAreaHeight: textAreaH
            };
        },
        
        // Führe Neuberechnung durch und benachrichtige alle Module
        recalculate: function() {
            this.calculateOptimalLayout();
            
            // Benachrichtige alle registrierten Module
            for (const callback of this.onResizeCallbacks) {
                try {
                    callback(this.current);
                } catch (e) {
                    console.error('[DYNAMIC-VIEWPORT] Fehler in Resize-Callback:', e);
                }
            }
        },        // Initialisierung
        init: function() {
            
            // Initiale Berechnung
            this.calculateOptimalLayout();
            
            // Event-Listener für Größenänderungen
            window.addEventListener('resize', () => {
                setTimeout(() => this.recalculate(), 100);
            });
            
            // Listener für Orientierungsänderungen (Mobile)
            window.addEventListener('orientationchange', () => {
                setTimeout(() => this.recalculate(), 300);
            });
            
        }
    };    // Automatische Initialisierung wenn DOM bereit ist
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', () => window.DynamicViewport.init());
    } else {
        window.DynamicViewport.init();
    }
})();
