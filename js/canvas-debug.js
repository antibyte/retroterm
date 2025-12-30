// Canvas Clear Debug - Überwache ALLE clearRect Aufrufe
(function() {
    console.log('[CANVAS-DEBUG] Installing canvas clear monitoring...');
    
    // Speichere die originale clearRect Funktion
    const originalClearRect = CanvasRenderingContext2D.prototype.clearRect;
    
    // Überschreibe clearRect global
    CanvasRenderingContext2D.prototype.clearRect = function(x, y, w, h) {
        console.log(`[CANVAS-DEBUG] clearRect called on canvas ${this.canvas.width}x${this.canvas.height} - clearing area (${x}, ${y}) ${w}x${h}`);
        console.trace('[CANVAS-DEBUG] clearRect call stack');
        
        // Rufe die originale Funktion auf
        return originalClearRect.call(this, x, y, w, h);
    };
    
    console.log('[CANVAS-DEBUG] Canvas clear monitoring installed');
})();