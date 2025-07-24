// renderLoop.js - Verbesserter Render-Loop für optimiertes 3D-Rendering

// Verbesserte Version des Render-Loops zur Überwachung und Wiederherstellung
// der Vektor-Szene bei problemen
function renderLoop(timestamp) {
    // Aktualisiere die Zeit für CRT-Effekte
    scanlineUniforms.time.value = timestamp * 0.001; // Sekunden
    
    // VEKTORSCENEN-ÜBERWACHUNG
    // Prüfen wir regelmäßig, ob die Vektorscene korrekt initialisiert ist und Objekte enthält
    if (!window.lastVectorSceneCheck || timestamp - window.lastVectorSceneCheck > 3000) {
        window.lastVectorSceneCheck = timestamp;
        
        // Scene-Info sammeln
        const sceneInfo = {
            initialized: !!window.vectorRendererReady,
            renderer: !!vectorRenderer,
            camera: !!vectorCamera,
            scene: !!vectorScene,
            children: vectorScene ? vectorScene.children.length : 0,
            registeredInstances: window.RetroGraphics._vectorObjects ? window.RetroGraphics._vectorObjects.size : 0
        };
        
        // Bei Problemen automatische Wiederherstellung (aber nicht für 2D vectorManager objects)
        // Check if we have 2D vector objects (floors, etc.) that shouldn't be reset
        const has2DVectorObjects = window.vectorManager && window.vectorManager.getVectorObjects && 
                                   window.vectorManager.getVectorObjects().length > 0;
        
        if ((!sceneInfo.initialized || sceneInfo.children === 0) && sceneInfo.registeredInstances > 0 && !has2DVectorObjects) {

            if (window.RetroGraphics && window.RetroGraphics.resetAndRebuildVectorScene) {
                window.RetroGraphics.resetAndRebuildVectorScene();
            }
        }
          // Debug-Ausgabe zur Szene nur bei kritischen Änderungen (entfernt um Console-Spam zu vermeiden)
        if (!window.lastSceneInfo || 
            window.lastSceneInfo.children !== sceneInfo.children || 
            window.lastSceneInfo.initialized !== sceneInfo.initialized) {

            window.lastSceneInfo = sceneInfo;
        }
    }
    
    // Führe das eigentliche Rendering durch
    render();
    
    // Nächsten Frame anfordern
    requestAnimationFrame(renderLoop);
}

// Exportieren für die Verwendung im Hauptmodul
export { renderLoop };
