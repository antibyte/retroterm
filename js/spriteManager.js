/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// Shadow-Copy für Instanzvergleich (id -> JSON-String der Werte)
const lastRenderedSpriteInstances = new Map();
// spriteManager.js
// spriteManager.js - ES6-Modul
// Verwaltung und Rendering von 2D-Sprites für das Retro-Terminal
// In ES6 Modulen können wir externe globale Bibliotheken direkter über window verwenden

const CFG = window.CRT_CONFIG;

// Sprite-Definitionen (Basis, 32x32, 16 Helligkeiten pro Pixel, 0=transparent)
export const spriteDefinitions = new Map(); // id -> { pixelData: Uint8Array(1024) }
// Virtuelle Sprites (Kompositionen)
export const virtualSpriteCompositions = new Map(); // id -> { layout: '2x2'|'4x4', baseSpriteIds: number[] }
// Aktive Instanzen
export const spriteInstances = new Map(); // id -> { definitionId, x, y, rotation, visible, ... }
// Warteschlange für UPDATE_SPRITE-Nachrichten, die auf ihre Definition warten
let pendingUpdates = [];

// Globale Variable _spritesDirty im window.RetroGraphics Namespace erstellen, falls nicht vorhanden
if (!window.RetroGraphics) {
    window.RetroGraphics = {};
}
if (typeof window.RetroGraphics._spritesDirty === 'undefined') {
    window.RetroGraphics._spritesDirty = false;
}

// Handler für Backend-Nachrichten
export function handleDefineSprite(data) {
    window.RetroGraphics._spritesDirty = true;
    // { command: 'DEFINE_SPRITE', id, pixelData: [1024] } OR { command: 'DEFINE_SPRITE', id, spriteData: base64PNG }
    try {
        // Erweiterte Validierung mit Fallback-Optionen
        let pixelArray = null;
        let validationPassed = false;
          if (data && typeof data.id === 'number' && data.command === 'DEFINE_SPRITE') {
            // Check for chess sprites (base64 PNG spriteData)
            if (typeof data.spriteData === 'string' && data.spriteData.length > 0) {
                // Handle chess sprite - convert PNG to quantized pixels
                convertPNGToQuantizedPixels(data.spriteData, data.id);
                return; // Early return for chess sprites
            }
            
            // Traditional pixelData handling
            if (Array.isArray(data.pixelData) && data.pixelData.length === 1024) {
                pixelArray = data.pixelData;
                validationPassed = true;
            } else if (typeof data.pixelData === 'string') {
                // Fallback: String-Format parsen
                try {
                    const parts = data.pixelData.split(',').map(s => parseInt(s.trim(), 10));
                    if (parts.length === 1024 && parts.every(n => !isNaN(n) && n >= 0 && n <= 15)) {
                        pixelArray = parts;
                        validationPassed = true;
                    }
                } catch (e) {

                }
            }
            
            if (validationPassed && pixelArray) {
                // Prüfe, dass alle Elemente Zahlen sind
                let allNumbers = true;
                let uniqueValues = new Set();
                
                for(let i = 0; i < pixelArray.length; i++) {
                    if (typeof pixelArray[i] !== 'number' || isNaN(pixelArray[i])) {
                        allNumbers = false;

                        break;
                    }
                    if (pixelArray[i] !== 0) {
                        uniqueValues.add(pixelArray[i]);
                    }
                }

                if (allNumbers) {
                    const byteArray = Uint8Array.from(pixelArray);
                    spriteDefinitions.set(data.id, { 
                        pixelData: byteArray,
                        cachedImageData: null // Wird lazy erstellt für Performance
                    });
 
                    // Verarbeite ausstehende Updates für diese Definition
                    processPendingUpdates(data.id);
                } else {

                }
            } else {
                let errorMsg = '[SPRITE-FRONTEND-DEBUG] Ungültige DEFINE_SPRITE Daten. ABGELEHNT.';
                if (!data) {
                    errorMsg += ' Daten-Objekt ist null oder undefined.';
                } else {
                    if (typeof data.id !== 'number') errorMsg += ` Ungültige/fehlende ID (erwartet number): ${JSON.stringify(data.id)}.`;
                    if (data.command !== 'DEFINE_SPRITE') errorMsg += ` Ungültiger/fehlender Command (erwartet 'DEFINE_SPRITE'): ${JSON.stringify(data.command)}.`;
                    if (!data.pixelData && !data.spriteData) {
                        errorMsg += ' Weder pixelData noch spriteData vorhanden.';
                    } else if (data.pixelData && !Array.isArray(data.pixelData) && typeof data.pixelData !== 'string') {
                        errorMsg += ` pixelData ist weder Array noch String: ${typeof data.pixelData}.`;
                    } else if (Array.isArray(data.pixelData) && data.pixelData.length !== 1024) {
                        errorMsg += ` pixelData Array-Länge ist ${data.pixelData.length}, erwartet 1024.`;
                    }
                }

            }
        } else {

        }
    } catch (error) {

    }
}
export function handleDefineVirtualSprite(data) {
    window.RetroGraphics._spritesDirty = true;
    // { type: 'DEFINE_VIRTUAL_SPRITE', id, layout, baseSpriteIds }

    if (data && typeof data.id !== 'undefined' && data.layout && Array.isArray(data.baseSpriteIds)) {
        virtualSpriteCompositions.set(data.id, { layout: data.layout, baseSpriteIds: data.baseSpriteIds });

    } else {

    }
}
export function handleUpdateSprite(data) {
    if (!spriteDefinitions.has(data.definitionId)) {
        
        // Update in Warteschlange einreihen
        pendingUpdates.push({
            id: data.id,
            definitionId: data.definitionId,
            x: parseFloat(data.x) || 0,
            y: parseFloat(data.y) || 0,
            rotation: Number(data.rotation || 0),
            visible: data.visible !== false
        });
        
        return;
    }
    
    // Definition ist verfügbar - verarbeite das Update sofort
    if (data && typeof data.id === 'number') {
        let newX = parseFloat(data.x) || 0;
        let newY = parseFloat(data.y) || 0;

        const newInstanceData = {
            id: data.id,
            definitionId: data.definitionId,
            x: newX,
            y: newY,
            rotation: Number(data.rotation || 0),
            visible: data.visible !== false
        };

        spriteInstances.set(newInstanceData.id, newInstanceData);
        
        window.RetroGraphics._spritesDirty = true;
    }
}

// Convert PNG chess sprite to quantized pixel data
function convertPNGToQuantizedPixels(base64PNG, spriteId) {
    try {
        const img = new Image();
        img.onload = function() {
            // Create temporary canvas to extract pixel data
            const canvas = document.createElement('canvas');
            canvas.width = 32;  // Chess sprites are 32x32
            canvas.height = 32;
            const ctx = canvas.getContext('2d');
            
            // Draw image to canvas (scaled to 32x32 if needed)
            ctx.drawImage(img, 0, 0, 32, 32);
            
            // Get pixel data
            const imageData = ctx.getImageData(0, 0, 32, 32);
            const pixels = imageData.data;
            
            // Convert to quantized brightness levels (16 levels)
            const quantizedPixels = new Uint8Array(1024);
            
            for (let i = 0; i < 1024; i++) {
                const pixelIndex = i * 4;
                const r = pixels[pixelIndex];
                const g = pixels[pixelIndex + 1];
                const b = pixels[pixelIndex + 2];
                const a = pixels[pixelIndex + 3];
                
                // If alpha is low, make transparent
                if (a < 128) {
                    quantizedPixels[i] = 0;
                } else {
                    // Convert RGB to brightness and quantize to 16 levels
                    const brightness = (r * 0.299 + g * 0.587 + b * 0.114) / 255;
                    const quantized = Math.round(brightness * 15);
                    quantizedPixels[i] = Math.max(1, quantized); // Ensure non-zero for opaque pixels
                }
            }
              // Store the sprite definition
            spriteDefinitions.set(spriteId, { 
                pixelData: quantizedPixels,
                cachedImageData: null
            });
            
            // Process any pending updates for this sprite
            processPendingUpdates(spriteId);
        };
        
        img.onerror = function() {

        };
        
        // Load the base64 PNG
        img.src = `data:image/png;base64,${base64PNG}`;
        
    } catch (error) {

    }
}

// Funktion zum Verarbeiten ausstehender Updates nach neuer Definition
function processPendingUpdates(definitionId) {
    const processedUpdates = [];
    
    for (let i = pendingUpdates.length - 1; i >= 0; i--) {
        const update = pendingUpdates[i];        if (update.definitionId === definitionId) {
            // Update jetzt verarbeiten
            spriteInstances.set(update.id, {
                definitionId: update.definitionId,
                x: update.x,
                y: update.y,
                rotation: update.rotation || 0,
                visible: update.visible !== false
            });
            
            // Aus der Warteschlange entfernen
            pendingUpdates.splice(i, 1);
            processedUpdates.push(update.id);
        }
    }
    
    if (processedUpdates.length > 0) {
        window.RetroGraphics._spritesDirty = true;
    }
}

// Optimierte Sprite-Rendering-Funktion mit ImageData-Caching
export function renderSprites(ctx, width, height) {
    // Nur rendern wenn sich etwas geändert hat
    if (!window.RetroGraphics._spritesDirty) {
        return; // Keine Änderungen, kein Rendering nötig
    }
    
    // Canvas für Sprites löschen (nur die sichtbaren Bereiche)
    ctx.clearRect(0, 0, width, height);
    
    // Sammle alle zu rendernden Sprites in Batches nach definitionId
    const renderBatches = new Map();
    
    for (const [id, inst] of spriteInstances) {
        if (!inst.visible) continue;
        
        let def = spriteDefinitions.get(inst.definitionId);
        if (!def || !def.pixelData || def.pixelData.length === 0) continue;

        if (!renderBatches.has(inst.definitionId)) {
            renderBatches.set(inst.definitionId, []);
        }
        renderBatches.get(inst.definitionId).push(inst);
    }

    // Rendere jede Batch von Sprites mit der gleichen Definition
    for (const [definitionId, instances] of renderBatches) {
        const def = spriteDefinitions.get(definitionId);
        
        // Cache für ImageData pro Definition
        if (!def.cachedImageData) {
            def.cachedImageData = createImageDataFromPixels(def.pixelData, CFG.SPRITE_SIZE, CFG.SPRITE_SIZE);
        }
        
        // Erstelle Sprite-Canvas nur einmal pro Definition pro Frame
        const tempCanvas = getTempCanvas(CFG.SPRITE_SIZE, CFG.SPRITE_SIZE);
        const tempCtx = tempCanvas.getContext('2d');
        tempCtx.putImageData(def.cachedImageData, 0, 0);
        
        // Rendere alle Instanzen dieser Definition
        for (const inst of instances) {
            ctx.save();
            
            // Koordinaten-Skalierung anwenden
            if (window.RetroGraphics && window.RetroGraphics.getScaledCoordinates) {
                const scaled = window.RetroGraphics.getScaledCoordinates(inst.x, inst.y);
                ctx.translate(scaled.x, scaled.y);
            } else {
                ctx.translate(inst.x, inst.y);
            }
            
            ctx.rotate((inst.rotation || 0) * Math.PI / 180);
            
            // Zeichne das gecachte Sprite-Bild
            ctx.drawImage(tempCanvas, -CFG.SPRITE_SIZE / 2, -CFG.SPRITE_SIZE / 2);
            
            ctx.restore();
        }
        
        // Gib temporären Canvas zurück an den Pool
        releaseTempCanvas(tempCanvas);
    }
    
    // Nach dem Rendern aller Sprites das Dirty-Flag zurücksetzen
    window.RetroGraphics._spritesDirty = false;
}

// Erstelle ImageData aus Pixel-Array (gecacht für bessere Performance)
function createImageDataFromPixels(pixelData, w, h) {
    const canvas = document.createElement('canvas');
    canvas.width = w;
    canvas.height = h;
    const ctx = canvas.getContext('2d');
    const imgData = ctx.createImageData(w, h);
    
    for (let i = 0; i < w * h; ++i) {
        const bright = pixelData[i];
        if (bright === 0) {
            imgData.data[i * 4 + 3] = 0; // Transparent
        } else {
            const colorHex = CFG.BRIGHTNESS_LEVELS[bright] || '#5FFF5F';
            imgData.data[i * 4 + 0] = parseInt(colorHex.substring(1, 3), 16);
            imgData.data[i * 4 + 1] = parseInt(colorHex.substring(3, 5), 16);
            imgData.data[i * 4 + 2] = parseInt(colorHex.substring(5, 7), 16);
            imgData.data[i * 4 + 3] = 255;
        }
    }
    
    return imgData;
}

// Pool von temporären Canvas für bessere Performance
const _tempCanvasPool = [];
const _tempCanvasPoolUsed = new Set();

function getTempCanvas(width, height) {
    // Versuche einen verfügbaren Canvas aus dem Pool zu finden
    for (let i = 0; i < _tempCanvasPool.length; i++) {
        const canvas = _tempCanvasPool[i];
        if (!_tempCanvasPoolUsed.has(canvas) && canvas.width === width && canvas.height === height) {
            _tempCanvasPoolUsed.add(canvas);
            return canvas;
        }
    }
    
    // Erstelle einen neuen Canvas wenn keiner verfügbar ist
    const newCanvas = document.createElement('canvas');
    newCanvas.width = width;
    newCanvas.height = height;
    _tempCanvasPool.push(newCanvas);
    _tempCanvasPoolUsed.add(newCanvas);
    return newCanvas;
}

function releaseTempCanvas(canvas) {
    _tempCanvasPoolUsed.delete(canvas);
}

// Die alte drawSpritePixelsToTempContext Funktion wurde durch createImageDataFromPixels ersetzt
// für bessere Performance durch Caching

// Die ursprüngliche drawSpritePixels-Funktion wird nicht mehr direkt in renderSprites verwendet.
// function drawSpritePixels(ctx, pixelData, w, h) { ... }

// Funktion zum Löschen aller Sprite-Daten
export function clearAllSpriteData() {
    window.RetroGraphics._spritesDirty = true;
    spriteDefinitions.clear();
    virtualSpriteCompositions.clear();
    spriteInstances.clear();

}

export function clearActiveSpriteInstances() {
    window.RetroGraphics._spritesDirty = true;
    spriteInstances.clear();

}

// Globale Verfügbarkeit sicherstellen
window.spriteManager = {
    spriteDefinitions,
    virtualSpriteCompositions,
    spriteInstances,
    renderSprites,
    handleDefineSprite,
    handleDefineVirtualSprite,
    handleUpdateSprite,
    clearAllSpriteData,
    clearActiveSpriteInstances
};

// Event auslösen, um anderen Modulen mitzuteilen, dass spriteManager verfügbar ist
const spriteManagerReadyEvent = new CustomEvent('spritemanagerready', {    detail: { timestamp: Date.now() } 
});
document.dispatchEvent(spriteManagerReadyEvent);
