/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// retrographics.js - CRT-Rendering mit WebGL/Three.js (als normales Script geladen)

// Globaler Namespace für RetroGraphics-Funktionalität
window.RetroGraphics = window.RetroGraphics || {};

// Funktion zum Aktualisieren der Sprite-Handler wenn spriteManager verfügbar wird
function updateSpriteHandlers() {
    if (window.spriteManager) {
        const { handleDefineSprite, handleDefineVirtualSprite, handleUpdateSprite, clearAllSpriteData } = window.spriteManager;
        
        window.RetroGraphics.handleDefineSprite = handleDefineSprite;
        window.RetroGraphics.handleUpdateSprite = handleUpdateSprite;
        window.RetroGraphics.handleDefineVirtualSprite = handleDefineVirtualSprite;        window.RetroGraphics.clearAllSpriteData = clearAllSpriteData;
    } else {
    }
}

// Versuche sofort die Handler zu aktualisieren (falls spriteManager bereits geladen ist)
updateSpriteHandlers();

// Event-Listener für später geladenen spriteManager
document.addEventListener('spritemanagerready', () => {
    updateSpriteHandlers();
});

// Verwende global verfügbare Module (keine ES6 Imports)
const { spriteDefinitions, virtualSpriteCompositions, spriteInstances, renderSprites } = window.spriteManager || {};
const vectorManager = window.vectorManager || {};

// Initialisierung des Vector-Speichers
window.RetroGraphics._vectorObjects = new Map();

// Initialisierung des Sprite-Dirty-Flags für Performance-Optimierung
window.RetroGraphics._spritesDirty = false;

// Initialisierung des 2D-Graphics-Dirty-Flags
window.RetroGraphics._graphics2DDirty = false;

// Render Targets für Nachleucht-Effekt
let readBuffer, writeBuffer;


// Globale Variablen für Post-Processing Szene und Kamera
let postProcessingScene, orthoCamera;
// NEU: Separate Szene für den finalen Render-Pass (Blitting)
let blitScene, blitMaterial;

let mainOutputCanvas;

const vertexShader = `
    varying vec2 vUv;
    void main() {
        vUv = uv;
        gl_Position = projectionMatrix * modelViewMatrix * vec4(position, 1.0);
    }
`;

// Fragment Shader
const fragmentShader = `
    uniform float time;
    uniform vec2 resolution;
    uniform vec2 graphicsResolution;
    uniform float u_paddingLeft;
    uniform float u_paddingTop;
    uniform float u_paddingRight;
    uniform float u_paddingBottom;

    uniform sampler2D mainTexture;          // 2D-Grafik-Canvas
    uniform sampler2D textTextureSampler;   // Text-Canvas
    uniform sampler2D prevFrameTexture;     // Vorheriger Frame für Afterglow

    // Effekt-Uniforms
    uniform bool scanlinesEnabled;
    uniform float scanlineIntensity;
    uniform float scanlineFrequency;

    uniform bool barrelDistortionEnabled;
    uniform float barrelDistortionStrength;
    uniform float u_barrelOverscan;    
    uniform bool noiseEnabled;
    uniform float noiseIntensity;
    uniform float noiseSpeed;
    uniform float u_time;

    uniform bool vignetteEnabled;
    uniform float vignetteStrength;
    uniform float vignetteRadius;    
    uniform bool glareEnabled;
    uniform float glareIntensity;
    uniform float glareSize;
    uniform vec2 glarePosition;
    uniform float glareFalloff;
    uniform float glareBrightnessThreshold;

    uniform bool afterglowEnabled;
    uniform float afterglowPersistence;

    // NEU: Hintergrundleuchten
    uniform bool backgroundGlowEnabled;
    uniform vec3 backgroundGlowColor;
    uniform bool circularFalloffEnabled;
    uniform float circularFalloffRadius;    // Wie weit das Leuchten reicht
    uniform float circularFalloffIntensity; // Wie stark der Abfall zur Kante hin ist

    // NEU: Flimmer-Effekt
    uniform bool flickerEnabled;
    uniform float flickerIntensity;

    // NEU: Chromatische Aberration
    uniform bool chromaticAberrationEnabled;
    uniform float chromaticAberrationStrength;

    // NEU: Shadow Mask
    uniform bool shadowMaskEnabled;
    uniform float shadowMaskIntensity;
    uniform float shadowMaskScale;

    // NEU: Screen Jitter
    uniform bool screenJitterEnabled;
    uniform float screenJitterAmount;
    uniform float screenJitterSpeed;

    // Hum Bar
    uniform bool humBarEnabled;
    uniform float humBarIntensity;
    uniform float humBarSpeed;
    uniform float humBarHeight;
    uniform bool humBarHeightVariationEnabled;
    uniform float humBarHeightVariationAmount;
    uniform float humBarHeightVariationSpeed;
    uniform float humBarFalloffStrength;

    varying vec2 vUv;

    // Funktion für Scanlines
    float getScanline(vec2 uv, float frequency, float intensity) {
        return pow(sin(uv.y * frequency) * 0.5 + 0.5, intensity);
    }    // Funktion für Rauschen/Noise
    float random(vec2 st) {
        return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123);
    }

    float noise(vec2 uv, float time) {
        vec2 i = floor(uv);
        vec2 f = fract(uv);
        
        // Sample random values at the corners of the current cell
        float a = random(i + vec2(0.0, 0.0) + time);
        float b = random(i + vec2(1.0, 0.0) + time);
        float c = random(i + vec2(0.0, 1.0) + time);
        float d = random(i + vec2(1.0, 1.0) + time);
        
        // Smooth interpolation
        vec2 u = f * f * (3.0 - 2.0 * f);
        
        return mix(a, b, u.x) + (c - a) * u.y * (1.0 - u.x) + (d - b) * u.x * u.y;
    }

    // Funktion für Barrel Distortion (Tonnenverzeichnung)
    vec2 barrelDistort(vec2 uv, float strength) {
        vec2 center = vec2(0.5, 0.5);
        vec2 texCoord = uv - center;

        // This is the key: calculate the maximum distortion at the corner of the screen
        // and pre-scale the texture coordinates to counteract it. This brings the
        // distorted corners back into the viewport without blurring.
        float r_corner_sq = 0.5; // (0.5*0.5 + 0.5*0.5)
        float corner_distortion = 1.0 + strength * r_corner_sq;

        // Apply the correction and the distortion
        float r = length(texCoord);
        float distortionFactor = 1.0 + strength * r * r;

        if (distortionFactor == 0.0) { return uv; } // Avoid division by zero

        vec2 distortedUv = center + (texCoord * corner_distortion) / distortionFactor;
        return distortedUv;
    }

    void main() {
        vec2 uv = vUv;

        // Berechne die Hintergrundleuchtfarbe für den *gesamten* Bildschirm,
        // basierend auf den unverzerrten UVs, um einen gleichmäßigen Effekt zu erzielen.
        vec3 finalGlowColor = vec3(0.0);
        if (backgroundGlowEnabled) {
            finalGlowColor = backgroundGlowColor;
            if (circularFalloffEnabled) {
                // Berechne den Abstand von der Mitte des Bildschirms (unverzerrte UVs)
                float dist = distance(uv, vec2(0.5));
                // Gauss-Funktion für einen natürlichen, glockenförmigen Abfall von der Mitte
                float falloff = exp(-pow(dist / circularFalloffRadius, 2.0) * circularFalloffIntensity);
                finalGlowColor *= falloff;
            }
        }

        // SCREEN JITTER (wird zuerst angewendet, da es die UVs für alles Folgende beeinflusst)
        if (screenJitterEnabled) {
            float jitterX = (random(vec2(u_time * screenJitterSpeed, 0.0)) - 0.5) * screenJitterAmount;
            float jitterY = (random(vec2(0.0, u_time * screenJitterSpeed)) - 0.5) * screenJitterAmount;
            uv += vec2(jitterX, jitterY);
        }

        vec2 final_uv = uv;

        // Barrel Distortion anwenden, falls aktiviert
        if (barrelDistortionEnabled) {
            final_uv = barrelDistort(uv, barrelDistortionStrength);
        }

        // Prüfen, ob die verzerrten UVs noch im gültigen Bereich [0,1] sind
        if (final_uv.x < 0.0 || final_uv.x > 1.0 || final_uv.y < 0.0 || final_uv.y > 1.0) {
            gl_FragColor = vec4(finalGlowColor, 1.0); // Zeige das berechnete Leuchten statt einer festen Farbe
            return;
        }
        
        // Pass 1: Kombiniert Text, Grafik und wendet Effekte an
        vec2 screen_uv_for_textures = vec2(final_uv.x, final_uv.y);

        // CHROMATIC ABERRATION (wird hier angewendet, um die Textur-Samples zu verschieben)
        vec2 offset = vec2(0.0);
        if (chromaticAberrationEnabled) {
            offset = (final_uv - 0.5) * chromaticAberrationStrength * 0.01;
        }

        // Text-Pixel abrufen
        // Flip Y-Koordinate für Canvas-Texturen
        vec4 textPixel;
        textPixel.r = texture2D(textTextureSampler, vec2(screen_uv_for_textures.x - offset.x, 1.0 - screen_uv_for_textures.y)).r;
        textPixel.g = texture2D(textTextureSampler, vec2(screen_uv_for_textures.x, 1.0 - screen_uv_for_textures.y)).g;
        textPixel.b = texture2D(textTextureSampler, vec2(screen_uv_for_textures.x + offset.x, 1.0 - screen_uv_for_textures.y)).b;
        textPixel.a = texture2D(textTextureSampler, vec2(screen_uv_for_textures.x, 1.0 - screen_uv_for_textures.y)).a;

        vec2 graphics_uv = vec2(-1.0, -1.0);

        // Prüfe, ob der aktuelle Pixel im gerenderten Bereich liegt
        if (screen_uv_for_textures.x * resolution.x >= u_paddingLeft && 
            screen_uv_for_textures.x * resolution.x <= resolution.x - u_paddingRight &&
            screen_uv_for_textures.y * resolution.y >= u_paddingTop && 
            screen_uv_for_textures.y * resolution.y <= resolution.y - u_paddingBottom) {
            
            float relative_screen_x = screen_uv_for_textures.x * resolution.x - u_paddingLeft;
            float relative_screen_y = screen_uv_for_textures.y * resolution.y - u_paddingTop;

            float effective_gfx_width_on_screen = resolution.x - u_paddingLeft - u_paddingRight;
            float effective_gfx_height_on_screen = resolution.y - u_paddingTop - u_paddingBottom;

            if (effective_gfx_width_on_screen > 0.0 && effective_gfx_height_on_screen > 0.0) {
                graphics_uv.x = relative_screen_x / effective_gfx_width_on_screen;
                graphics_uv.y = relative_screen_y / effective_gfx_height_on_screen;
            }
        }
        
        vec4 graphicsPixel = vec4(0.0, 0.0, 0.0, 0.0); 
        if (graphics_uv.x >= 0.0 && graphics_uv.x <= 1.0 &&
            graphics_uv.y >= 0.0 && graphics_uv.y <= 1.0) {
            graphicsPixel = texture2D(mainTexture, vec2(graphics_uv.x, 1.0 - graphics_uv.y));
        }

        // Kombiniere Text und Grafik
        vec4 combinedColor;
        bool isTextVisible = textPixel.r > 0.01 || textPixel.g > 0.01 || textPixel.b > 0.01;

        if (isTextVisible) { 
            combinedColor = textPixel;
        } else if (graphicsPixel.a > 0.01) {
            combinedColor = graphicsPixel;
        } else {
            combinedColor = vec4(0.0, 0.0, 0.0, 1.0);
        }

        // Scanlines-Effekt
        if (scanlinesEnabled) {
            float scanlineEffect = getScanline(final_uv, scanlineFrequency, scanlineIntensity);
            combinedColor.rgb *= scanlineEffect;
        }
        
        // SHADOW MASK (simuliert die Phosphor-Punkte)
        if (shadowMaskEnabled) {
            float mask = 1.0 - shadowMaskIntensity;
            float verticalMask = sin( (final_uv.y + 0.5) * resolution.y * shadowMaskScale * 0.5 );
            float horizontalMask = sin( (final_uv.x + 0.5) * resolution.x * shadowMaskScale );
            combinedColor.rgb *= mix(vec3(mask), vec3(1.0), (verticalMask + horizontalMask) * 0.5);
        }

        // Vignette-Effekt
        if (vignetteEnabled) {
            vec2 center = vec2(0.5, 0.5);
            float dist = distance(final_uv, center);
            float vignette = smoothstep(vignetteRadius, vignetteRadius - vignetteStrength, dist);
            combinedColor.rgb *= vignette;
        }

        // HINTERGRUNDLEUCHTEN (Phosphor-Effekt)
        // Das Leuchten ist eine Basisfarbe. max() stellt sicher, dass hellere Pixel nicht dunkler werden.
        if (backgroundGlowEnabled) { // Wir verwenden die bereits berechnete finalGlowColor
            combinedColor.rgb = max(combinedColor.rgb, finalGlowColor);
        }

        // Glare-Effekt - wird ZUERST berechnet (physikalische Reflexion des Glases)
        vec3 glareColor = vec3(0.0);
        float glareMask = 0.0;
        if (glareEnabled) {
            float glareDist = distance(final_uv, glarePosition);
            float glare = pow(1.0 - smoothstep(0.0, glareSize, glareDist), glareFalloff) * glareIntensity;
            
            // Berechne die Helligkeit des ursprünglichen Pixels (vor allen Effekten)
            float brightness = dot(combinedColor.rgb, vec3(0.299, 0.587, 0.114)); // Luminanz-Gewichtung
            
            // Glare-Maske: Stärker auf dunklen Bereichen, schwächer auf hellen (konfigurierbarer Schwellwert)
            glareMask = (1.0 - smoothstep(0.0, glareBrightnessThreshold, brightness)) * glare;
            
            // Glare als separate weiße Reflexion
            glareColor = vec3(glareMask);
        }
        
        // CRT-Rauschen-Effekt - wird NACH Glare berechnet, aber Glare-Bereiche bleiben rauschfrei
        if (noiseEnabled) {
            float time = u_time * noiseSpeed * 0.1; // Skaliere die Zeit für langsamere Animation
            float noiseValue = noise(final_uv * 200.0, time); // Hochfrequentes Rauschen
            
            // Berechne die Helligkeit des aktuellen Pixels
            float brightness = dot(combinedColor.rgb, vec3(0.299, 0.587, 0.114));
            
            // Subtiles Rauschen, das zwischen -noiseIntensity und +noiseIntensity variiert
            float noiseOffset = (noiseValue - 0.5) * noiseIntensity * 0.01;
            
            // Auf dunklen Bereichen (schwarze Pixel) verwende grünes Rauschen
            // Auf hellen Bereichen verwende normales weißes Rauschen
            vec3 darkGreen = vec3(0.0, 0.4, 0.0); // Dunkles Grün ähnlich #00aa00
            vec3 noiseColor = mix(darkGreen, vec3(1.0), brightness * 5.0); // Mix zwischen grün und weiß basierend auf Helligkeit
            
            // Rauschen nur dort anwenden, wo kein Glare ist (1.0 - glareMask reduziert Rauschen bei Glare)
            float noiseReduction = 1.0 - min(glareMask, 1.0);
            combinedColor.rgb += noiseColor * noiseOffset * noiseReduction;
        }

        // FLIMMER-EFFEKT (Helligkeitsschwankung des Phosphors)
        if (flickerEnabled) {
            // Erzeugt einen Wert zwischen -flickerIntensity und +flickerIntensity
            float flicker = (random(vec2(u_time, u_time * 0.5)) * 2.0 - 1.0) * flickerIntensity;
            combinedColor.rgb += flicker;
        }

        // HUM BAR (Netzbrummen-Streifen)
        if (humBarEnabled) {
            float barPosition = fract(u_time * humBarSpeed); // Position des Balkens, die über die Zeit scrollt
            float distFromBar = abs(vUv.y - barPosition); // Abstand des Pixels von der Balkenmitte

            float currentBarHeight = humBarHeight;
            if (humBarHeightVariationEnabled) {
                // Erzeugt einen Multiplikator, der um 1.0 schwankt (z.B. 0.5 bis 1.5 bei amount=0.5)
                float heightMultiplier = 1.0 + (random(vec2(u_time * humBarHeightVariationSpeed, 42.0)) * 2.0 - 1.0) * humBarHeightVariationAmount;
                currentBarHeight *= heightMultiplier;
            }

            float barHeight = currentBarHeight * 0.1; // Höhe des Balkens

            // Stärke des Effekts basierend auf der Distanz (weiche Kanten mit Gauss-Funktion für natürlicheren Abfall)
            float humStrength = exp(-pow(distFromBar / barHeight, 2.0) * humBarFalloffStrength);

            if (humStrength > 0.0) {
                // Erzeuge ein Rauschen, das wie horizontale Störlinien aussieht.
                // Die vUv.y-Koordinate wird stark gestreckt, um Linien zu erzeugen.
                float noise = random( vec2(vUv.x * 2.0, vUv.y * 300.0) + u_time * 20.0 );

                // Die Intensität des Rauschens wird durch die Position im Balken (humStrength) moduliert.
                combinedColor.rgb += vec3(0.2, 1.0, 0.2) * noise * humStrength * humBarIntensity;
            }
        }

        // AFTERGLOW-EFFEKT (NACHLEUCHTEN)
        // Dieser Effekt wird als einer der letzten angewendet. Er mischt den komplett
        // gerenderten aktuellen Frame mit dem gedämpften vorherigen Frame.
        // max() stellt sicher, dass neue Inhalte immer in voller Helligkeit erscheinen,
        // während alte Inhalte (Nachleuchten) nur dann sichtbar sind, wenn sie heller
        // als der aktuelle Inhalt sind.
        if (afterglowEnabled) {
            // WICHTIG: Der vorherige Frame (prevFrameTexture) ist bereits verzerrt.
            // Wir müssen ihn mit den *unverzerrten* UVs (vUv) abtasten, um den "Tunnel"-Effekt zu vermeiden.
            vec4 decayedPrevFrame = texture2D(prevFrameTexture, vUv) * afterglowPersistence;
            combinedColor = max(combinedColor, decayedPrevFrame);
        }

        // FINALE KOMPOSITION
        // Der Glare (Reflexion auf dem Glas) wird ZULETZT hinzugefügt,
        // damit er von den anderen Effekten (Rauschen, Flimmern etc.) unberührt bleibt.
        vec3 finalColor = max(combinedColor.rgb, glareColor);

        gl_FragColor = vec4(clamp(finalColor, 0.0, 1.0), combinedColor.a);
    }
`;

// Globale Variablen
let textCanvas, textTexture, textContext;
let graphicsSpriteTexture = null;

// Canvas für persistente 2D-Grafiken
let persistentGraphicsCanvas = null;
let persistentGraphicsContext = null;

// Separate canvas for persistent 2D graphics
let persistent2DCanvas = null;
let persistent2DContext = null;

let glowCanvas, glowTexture, glowCtx;

let renderer;
let crtMesh, crtMaterial, crtUniforms;

// Animation-bezogene globale Variablen
let animationFrameId = null;
let animationRunning = false;

const CFG = window.CRT_CONFIG;

function initGraphicsPipeline(textCanvasSource, textTextureSource) {
    // Ensure DOM is ready for canvas creation
    if (!document.body) {

        return;
    }

    // 1. Setup main output canvas (terminalCanvas)
    const existingCanvas = document.getElementById('terminalCanvas');
    if (existingCanvas) {
        mainOutputCanvas = existingCanvas;
        mainOutputCanvas.width = CFG.VIRTUAL_CRT_WIDTH;
        mainOutputCanvas.height = CFG.VIRTUAL_CRT_HEIGHT;
    } else {
        mainOutputCanvas = document.createElement('canvas');
        mainOutputCanvas.id = 'terminalCanvas';
        mainOutputCanvas.width = CFG.VIRTUAL_CRT_WIDTH;
        mainOutputCanvas.height = CFG.VIRTUAL_CRT_HEIGHT;
        const container = document.querySelector('.crt-container') || document.body;
        container.appendChild(mainOutputCanvas);
    }

    // Initialize Three.js renderer with the main output canvas
    try {
        renderer = new THREE.WebGLRenderer({ canvas: mainOutputCanvas });
        renderer.setSize(mainOutputCanvas.width, mainOutputCanvas.height);
    } catch (error) {

        mainOutputCanvas.getContext('2d').fillText("WebGL Error. Cannot initialize graphics.", 10, 50);
        return;
    }

    // Render Targets für Afterglow-Effekt initialisieren
    // Diese werden für den Nachleucht-Effekt benötigt (Ping-Pong-Rendering)
    const renderTargetOptions = {
        minFilter: THREE.LinearFilter,
        magFilter: THREE.LinearFilter, // LinearFilter für weichere Übergänge beim Nachleuchten
        format: THREE.RGBAFormat,
        type: THREE.FloatType // FloatType für bessere Präzision bei Nachleuchteffekten
    };
    readBuffer = new THREE.WebGLRenderTarget(mainOutputCanvas.width, mainOutputCanvas.height, renderTargetOptions);
    readBuffer.texture.flipY = false;
    writeBuffer = new THREE.WebGLRenderTarget(mainOutputCanvas.width, mainOutputCanvas.height, renderTargetOptions);
    writeBuffer.texture.flipY = false;

    // 2. Setup Text Rendering Part
    if (textCanvasSource && textTextureSource) {
        textCanvas = textCanvasSource;
        textTexture = textTextureSource;


    } else {
        // Fallback: Create own text canvas and texture
        textCanvas = document.createElement('canvas');
        textCanvas.width = CFG.VIRTUAL_CRT_WIDTH;
        textCanvas.height = CFG.VIRTUAL_CRT_HEIGHT;
        textContext = textCanvas.getContext('2d');
        textContext.imageSmoothingEnabled = false;
        textTexture = new THREE.CanvasTexture(textCanvas);
        textTexture.minFilter = THREE.NearestFilter;
        textTexture.magFilter = THREE.NearestFilter;
        textTexture.flipY = false;
    }

    // 3. Setup Persistent Graphics Area
    persistentGraphicsCanvas = document.createElement('canvas');
    persistentGraphicsCanvas.width = CFG.GRAPHICS_WIDTH;
    persistentGraphicsCanvas.height = CFG.GRAPHICS_HEIGHT;
    persistentGraphicsContext = persistentGraphicsCanvas.getContext('2d');
    if (persistentGraphicsContext) {
        persistentGraphicsContext.imageSmoothingEnabled = CFG.GRAPHICS_ANTIALIASING !== undefined ? CFG.GRAPHICS_ANTIALIASING : false;
    } else {

        return;
    }
    
    graphicsSpriteTexture = new THREE.CanvasTexture(persistentGraphicsCanvas);
    graphicsSpriteTexture.minFilter = THREE.NearestFilter;
    graphicsSpriteTexture.magFilter = THREE.NearestFilter;
    graphicsSpriteTexture.flipY = false;

    // 3.5. Setup separate canvas for persistent 2D graphics
    persistent2DCanvas = document.createElement('canvas');
    persistent2DCanvas.width = CFG.GRAPHICS_WIDTH;
    persistent2DCanvas.height = CFG.GRAPHICS_HEIGHT;
    persistent2DContext = persistent2DCanvas.getContext('2d');
    if (persistent2DContext) {
        persistent2DContext.imageSmoothingEnabled = CFG.GRAPHICS_ANTIALIASING !== undefined ? CFG.GRAPHICS_ANTIALIASING : false;
    } else {

        return;
    }    // Initialize vectorManager if available
    if (window.vectorManager && typeof window.vectorManager.initVectorManager === 'function') {
        window.vectorManager.initVectorManager(persistentGraphicsCanvas, persistentGraphicsContext);
    } else {

    }
    
    // Initialize imageManager if available
    if (window.imageManager && typeof window.imageManager.initImageManager === 'function') {
        window.imageManager.initImageManager();
    }
    
    // Initialize particleManager if available
    if (window.particleManager && typeof window.particleManager.initParticleManager === 'function') {
        window.particleManager.initParticleManager();
    }
    
    // 5. Setup Post-Processing Scene
    postProcessingScene = new THREE.Scene();
    orthoCamera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);

    const planeGeometry = new THREE.PlaneGeometry(2, 2);

    // NEU: Szene und Material für den finalen Blit-Pass
    blitScene = new THREE.Scene();
    blitMaterial = new THREE.ShaderMaterial({
        uniforms: {
            displayTexture: { value: null }
        },
        vertexShader: vertexShader, // Kann den gleichen einfachen Vertex-Shader verwenden
        fragmentShader: `
            uniform sampler2D displayTexture;
            varying vec2 vUv;
            void main() {
                gl_FragColor = texture2D(displayTexture, vUv);
            }
        `,
        transparent: false,
        depthTest: false,
        depthWrite: false
    });
    const blitQuad = new THREE.Mesh(planeGeometry, blitMaterial);
    blitScene.add(blitQuad);
    
    // Uniforms for the CRT shader
    crtUniforms = {
        textTextureSampler: { value: textTexture },
        mainTexture: { value: graphicsSpriteTexture },
        
        time: { value: 0.0 },
        resolution: { value: new THREE.Vector2(mainOutputCanvas.width, mainOutputCanvas.height) },
        graphicsResolution: { value: new THREE.Vector2(CFG.GRAPHICS_WIDTH || 640, CFG.GRAPHICS_HEIGHT || 480) },
        
        // CRT effect parameters from new config structure
        scanlinesEnabled: { value: CFG.CRT_EFFECTS.SCANLINES_ENABLED },
        scanlineIntensity: { value: CFG.CRT_EFFECTS.SCANLINES_INTENSITY },
        scanlineFrequency: { value: CFG.CRT_EFFECTS.SCANLINES_FREQUENCY },
        
        barrelDistortionEnabled: { value: CFG.CRT_EFFECTS.BARREL_DISTORTION_ENABLED },
        barrelDistortionStrength: { value: CFG.CRT_EFFECTS.BARREL_DISTORTION_STRENGTH },
        u_barrelOverscan: { value: CFG.CRT_EFFECTS.BARREL_OVERSCAN },
        noiseEnabled: { value: CFG.CRT_EFFECTS.NOISE_ENABLED },
        noiseIntensity: { value: CFG.CRT_EFFECTS.NOISE_INTENSITY },
        noiseSpeed: { value: CFG.CRT_EFFECTS.NOISE_SPEED },
        u_time: { value: 0.0 },
        
        vignetteEnabled: { value: CFG.CRT_EFFECTS.VIGNETTE_ENABLED },
        vignetteStrength: { value: CFG.CRT_EFFECTS.VIGNETTE_STRENGTH },
        vignetteRadius: { value: CFG.CRT_EFFECTS.VIGNETTE_RADIUS },
        glareEnabled: { value: CFG.CRT_EFFECTS.GLARE_ENABLED },
        glareIntensity: { value: CFG.CRT_EFFECTS.GLARE_INTENSITY },
        glareSize: { value: CFG.CRT_EFFECTS.GLARE_SIZE },
        glarePosition: { value: new THREE.Vector2(
            CFG.CRT_EFFECTS.GLARE_POSITION_X,
            CFG.CRT_EFFECTS.GLARE_POSITION_Y
        )},
        glareFalloff: { value: CFG.CRT_EFFECTS.GLARE_FALLOFF },
        glareBrightnessThreshold: { value: CFG.CRT_EFFECTS.GLARE_BRIGHTNESS_THRESHOLD },
        
        afterglowEnabled: { value: CFG.CRT_EFFECTS.AFTERGLOW_ENABLED },
        afterglowPersistence: { value: CFG.CRT_EFFECTS.AFTERGLOW_PERSISTENCE },

        // Hintergrundleuchten
        backgroundGlowEnabled: { value: CFG.CRT_EFFECTS.BACKGROUND_GLOW.ENABLED },
        backgroundGlowColor: { value: new THREE.Color(CFG.CRT_EFFECTS.BACKGROUND_GLOW.COLOR) },
        circularFalloffEnabled: { value: CFG.CRT_EFFECTS.BACKGROUND_GLOW.CIRCULAR_FALLOFF_ENABLED },
        circularFalloffRadius: { value: CFG.CRT_EFFECTS.BACKGROUND_GLOW.CIRCULAR_FALLOFF_RADIUS },
        circularFalloffIntensity: { value: CFG.CRT_EFFECTS.BACKGROUND_GLOW.CIRCULAR_FALLOFF_INTENSITY },
        
        // Flimmer-Effekt
        flickerEnabled: { value: CFG.CRT_EFFECTS.FLICKER_ENABLED },
        flickerIntensity: { value: CFG.CRT_EFFECTS.FLICKER_INTENSITY },

        // Chromatische Aberration
        chromaticAberrationEnabled: { value: CFG.CRT_EFFECTS.CHROMATIC_ABERRATION_ENABLED },
        chromaticAberrationStrength: { value: CFG.CRT_EFFECTS.CHROMATIC_ABERRATION_STRENGTH },

        // Shadow Mask
        shadowMaskEnabled: { value: CFG.CRT_EFFECTS.SHADOW_MASK_ENABLED },
        shadowMaskIntensity: { value: CFG.CRT_EFFECTS.SHADOW_MASK_INTENSITY },
        shadowMaskScale: { value: CFG.CRT_EFFECTS.SHADOW_MASK_SCALE },

        // Screen Jitter
        screenJitterEnabled: { value: CFG.CRT_EFFECTS.SCREEN_JITTER_ENABLED },
        screenJitterAmount: { value: CFG.CRT_EFFECTS.SCREEN_JITTER_AMOUNT },
        screenJitterSpeed: { value: CFG.CRT_EFFECTS.SCREEN_JITTER_SPEED },

        // Hum Bar
        humBarEnabled: { value: CFG.CRT_EFFECTS.HUM_BAR_ENABLED },
        humBarIntensity: { value: CFG.CRT_EFFECTS.HUM_BAR_INTENSITY },
        humBarSpeed: { value: CFG.CRT_EFFECTS.HUM_BAR_SPEED },
        humBarHeight: { value: CFG.CRT_EFFECTS.HUM_BAR_HEIGHT },
        humBarHeightVariationEnabled: { value: CFG.CRT_EFFECTS.HUM_BAR_HEIGHT_VARIATION_ENABLED },
        humBarHeightVariationAmount: { value: CFG.CRT_EFFECTS.HUM_BAR_HEIGHT_VARIATION_AMOUNT },
        humBarHeightVariationSpeed: { value: CFG.CRT_EFFECTS.HUM_BAR_HEIGHT_VARIATION_SPEED },
        humBarFalloffStrength: { value: CFG.CRT_EFFECTS.HUM_BAR_FALLOFF_STRENGTH },

        prevFrameTexture: { value: null },
        // Padding uniforms
        u_paddingLeft: { value: CFG.SCREEN_PADDING_LEFT || 0 },
        u_paddingTop: { value: CFG.SCREEN_PADDING_TOP || 0 },
        u_paddingRight: { value: CFG.SCREEN_PADDING_RIGHT || 0 },
        u_paddingBottom: { value: CFG.SCREEN_PADDING_BOTTOM || 0 }
    };

    // CRT Shader Material
    crtMaterial = new THREE.ShaderMaterial({
        uniforms: crtUniforms,
        vertexShader: vertexShader,
        fragmentShader: fragmentShader,
        transparent: false
    });

    const crtQuad = new THREE.Mesh(planeGeometry, crtMaterial);
    postProcessingScene.add(crtQuad);

    // Final check
    if (renderer && postProcessingScene && orthoCamera) {

    } else {

        return;
    }
    
    // Start animation loop
    if (animationFrameId === null) {
        try {
            renderer.render(postProcessingScene, orthoCamera);
        } catch (e) {

        }
    }
    animationRunning = true;
    animateCRT();    // Canvas-Referenzen global verfügbar machen für CLS-Befehl und Bitmap-Rendering
    window.persistentGraphicsContext = persistentGraphicsContext;
    window.persistentGraphicsCanvas = persistentGraphicsCanvas;
    window.persistent2DContext = persistent2DContext;
    window.persistent2DCanvas = persistent2DCanvas;

    // Canvas-Referenzen auch über RetroGraphics Namespace verfügbar machen
    window.RetroGraphics.persistentGraphicsContext = persistentGraphicsContext;
    window.RetroGraphics.persistentGraphicsCanvas = persistentGraphicsCanvas;
    window.RetroGraphics.persistent2DContext = persistent2DContext;
    window.RetroGraphics.persistent2DCanvas = persistent2DCanvas;

    // Dispatch event indicating RetroGraphics is ready
    const event = new CustomEvent('retrographicsready', { detail: { success: true } });
    document.dispatchEvent(event);
    window.retroGraphicsFullyInitialized = true;

}

function animateCRT() {
    if (!animationRunning) {
        return;
    }

    animationFrameId = requestAnimationFrame(animateCRT);

    // Update time for shader effects
    if (crtUniforms && crtUniforms.time) {
        crtUniforms.time.value = performance.now() * 0.001;
        crtUniforms.u_time.value = performance.now() * 0.001;
    }

    // Update Text Texture
    if (textTexture && window.RetroConsole && window.RetroConsole.isTextCanvasDirty && window.RetroConsole.isTextCanvasDirty()) {
        textTexture.needsUpdate = true;
        window.RetroConsole.markTextCanvasClean();
    }    // 2. Update Graphics (Sprites and Vectors) - only when needed
    if (persistentGraphicsContext && persistentGraphicsCanvas) {
        // Only render sprites if there are changes
        const spritesNeedUpdate = window.RetroGraphics && window.RetroGraphics._spritesDirty;
        const vectorsNeedUpdate = window.RetroGraphics && window.RetroGraphics._vectorsDirty;
        const graphics2DNeedUpdate = window.RetroGraphics && window.RetroGraphics._graphics2DDirty;
        const imagesNeedUpdate = window.RetroGraphics && window.RetroGraphics._imagesDirty;
        const particlesNeedUpdate = window.RetroGraphics && window.RetroGraphics._particlesDirty;
        
        if (spritesNeedUpdate || vectorsNeedUpdate || graphics2DNeedUpdate || imagesNeedUpdate || particlesNeedUpdate) {
            // Clear the graphics canvas for this frame
            persistentGraphicsContext.clearRect(0, 0, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
              // Render sprites if spriteManager is available and sprites need update
            if (spritesNeedUpdate && window.spriteManager && typeof window.spriteManager.renderSprites === 'function') {
                window.spriteManager.renderSprites(persistentGraphicsContext);
                // Das Dirty-Flag wird bereits in renderSprites() zurückgesetzt
            }
              // Render vectors if vectorManager is available and vectors need update
            if (vectorsNeedUpdate && window.vectorManager && typeof window.vectorManager.renderVectors === 'function') {
                window.vectorManager.renderVectors(persistentGraphicsContext, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
                // Reset vector dirty flag after rendering (but keep it dirty if we have persistent objects like floors)
                const hasFloorObjects = window.vectorManager.getVectorObjects().some(obj => obj && obj.type === 'floor');
                if (!hasFloorObjects) {
                    window.RetroGraphics._vectorsDirty = false;
                }
                // If we have floor objects, keep the dirty flag to ensure continuous rendering without flicker
            }
            
            // Always render all visible images if imageManager is available
            // (since canvas was cleared, we need to redraw everything)
            if (window.imageManager && typeof window.imageManager.renderImages === 'function') {
                window.imageManager.renderImages(persistentGraphicsContext, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
            }
            
            // Reset image dirty flag after rendering
            if (imagesNeedUpdate) {
                window.RetroGraphics._imagesDirty = false;
            }
            
            // Always render particles if particleManager is available
            // (particles have their own continuous update/animation loop)
            if (window.particleManager && typeof window.particleManager.renderParticles === 'function') {
                window.particleManager.renderParticles(persistentGraphicsContext, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
            }
            
            // Don't reset particle dirty flag - particles need continuous animation
            // The particleManager will manage its own dirty state based on active particles
            
            // Copy persistent2D graphics to main graphics canvas if needed
            if (persistent2DCanvas && persistent2DContext) {
                persistentGraphicsContext.drawImage(persistent2DCanvas, 0, 0);
                // Keep graphics2D dirty if persistent2D canvas has content
                window.RetroGraphics._graphics2DDirty = true;
            }
            
            // Reset 2D graphics dirty flag only if no persistent content
            if (graphics2DNeedUpdate && !(persistent2DCanvas && persistent2DContext)) {
                window.RetroGraphics._graphics2DDirty = false;
            }
        }
        
        // Always update the graphics texture if any graphics content changed
        if ((spritesNeedUpdate || vectorsNeedUpdate || graphics2DNeedUpdate || imagesNeedUpdate || particlesNeedUpdate) && graphicsSpriteTexture) {
            graphicsSpriteTexture.needsUpdate = true;
        }
    }
   // Update time for shader effects
    if (crtUniforms && crtUniforms.u_time) {
        crtUniforms.u_time.value = performance.now() * 0.001;
    }
    // 3. Render the Post-Processing Scene (applies CRT shader)
    if (renderer && postProcessingScene && orthoCamera) {
        try {
            if (CFG.CRT_EFFECTS.AFTERGLOW_ENABLED && readBuffer && writeBuffer) {
                // --- Afterglow Render-Pfad ---
    
                // Pass 1: Rendere die Szene mit allen Effekten in den `writeBuffer`.
                // Der Shader liest dabei aus dem readBuffer (letzter Frame).
                crtUniforms.prevFrameTexture.value = readBuffer.texture;
                renderer.setRenderTarget(writeBuffer);
                renderer.render(postProcessingScene, orthoCamera);
    
                // Pass 2: Rendere das Ergebnis aus dem `writeBuffer` auf den Bildschirm.
                // Hierfür wird die separate Blit-Szene mit dem einfachen Shader verwendet.
                blitMaterial.uniforms.displayTexture.value = writeBuffer.texture;
                renderer.setRenderTarget(null);
                renderer.render(blitScene, orthoCamera);
    
                // Tausche die Puffer für den nächsten Frame
                [readBuffer, writeBuffer] = [writeBuffer, readBuffer];
    
            } else { // --- Standard-Render-Pfad (ohne Afterglow) ---
                renderer.setRenderTarget(null); // Direkt auf den Bildschirm rendern
                renderer.render(postProcessingScene, orthoCamera);
            }

        } catch (error) {

        }
    } else {

    }
}

// Helper function to convert brightness value to hex color
function brightnessToColor(brightness) {
    if (typeof brightness !== 'number' || brightness < 0 || brightness > 15) {
        return '#5FFF5F'; // Default terminal green
    }
    
    if (window.CONFIG && window.CONFIG.BRIGHTNESS_LEVELS && window.CONFIG.BRIGHTNESS_LEVELS[brightness]) {
        return window.CONFIG.BRIGHTNESS_LEVELS[brightness];
    }
    
    // Fallback if config not available
    const brightnessLevels = [
        '#000000', '#001500', '#002500', '#003500', '#004500',
        '#005500', '#006000', '#007000', '#008000', '#009000', 
        '#00A000', '#00B000', '#00C000', '#00D000', '#00E000', '#5FFF5F'
    ];
    return brightnessLevels[brightness] || '#5FFF5F';
}

// Helper function to get color from graphics command data
function getColorFromData(data) {
    // If brightness is specified, use it
    if (typeof data.brightness === 'number') {
        return brightnessToColor(data.brightness);
    }
    
    // If color is specified as string, use it
    if (data.color && typeof data.color === 'string') {
        return data.color;
    }
    
    // Default terminal green
    return '#5FFF5F';
}

// Utility functions
function updateTextTexture(newTexture) {
    if (newTexture && crtMaterial && crtMaterial.uniforms && crtMaterial.uniforms.textTextureSampler) {
        crtMaterial.uniforms.textTextureSampler.value = newTexture;
        return true;
    }
    return false;
}

function forceTextTextureUpdate() {
    if (textTexture && textTexture.image) {
        textTexture.needsUpdate = true;
        return true;
    } else {

        return false;
    }
}

function recreateMaterial() {
    if (!crtMaterial || !crtMesh) {

        return false;
    }
    return true;
}

// 2D-Grafik-Handler
function handlePlot(data) {
    if (!persistent2DCanvas || !persistent2DContext) {
        console.warn("[RETROGRAPHICS] Canvas oder Context nicht verfügbar für PLOT");
        return;
    }
    
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
    ctx.fillStyle = color;
    ctx.fillRect(Math.floor(data.x), Math.floor(data.y), 1, 1);
    
    // Setze Dirty-Flag für 2D-Grafiken
    window.RetroGraphics._graphics2DDirty = true;
}

function handleLine(data) {
    if (!persistent2DCanvas || !persistent2DContext) {
        console.warn("[RETROGRAPHICS] Canvas oder Context nicht verfügbar für LINE");
        return;
    }
    
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
    ctx.strokeStyle = color;
    ctx.beginPath();
    ctx.moveTo(Math.floor(data.x1), Math.floor(data.y1));
    ctx.lineTo(Math.floor(data.x2), Math.floor(data.y2));
    ctx.stroke();
    
    // Setze Dirty-Flag für 2D-Grafiken
    window.RetroGraphics._graphics2DDirty = true;
}

function handleCircle(data) {
    if (!persistent2DCanvas || !persistent2DContext) {
        console.warn("[RETROGRAPHICS] Canvas oder Context nicht verfügbar für CIRCLE");
        return;
    }
    
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
    
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(Math.floor(data.x), Math.floor(data.y), Math.floor(data.radius), 0, 2 * Math.PI);
    
    if (data.fill) {
        ctx.fill();
    } else {
        ctx.stroke();
    }
    
    // Setze Dirty-Flag für 2D-Grafiken
    window.RetroGraphics._graphics2DDirty = true;
}

function handleRect(data) {

    
    if (!persistent2DCanvas || !persistent2DContext) {
        console.error("[RECT-DEBUG] Canvas/Context nicht verfügbar:", {
            canvas: !!persistent2DCanvas,
            context: !!persistent2DContext
        });
        return;
    }
    

    
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
  
      ctx.strokeStyle = color;
    ctx.fillStyle = color;
    
    if (data.fill) {
        ctx.fillRect(Math.floor(data.x), Math.floor(data.y), Math.floor(data.width), Math.floor(data.height));

    } else {
        ctx.strokeRect(Math.floor(data.x), Math.floor(data.y), Math.floor(data.width), Math.floor(data.height));

    }
    
    // Setze Dirty-Flag für 2D-Grafiken
    window.RetroGraphics._graphics2DDirty = true;
}

function handleFill(data) {
    if (!persistent2DCanvas || !persistent2DContext) {
        console.warn("[RETROGRAPHICS] Canvas oder Context nicht verfügbar für FILL");
        return;
    }
    
    const ctx = persistent2DContext;
    let color = (data && data.color && typeof data.color === 'string') ? data.color : '#000000';
    
    ctx.fillStyle = color;
    ctx.fillRect(0, 0, persistent2DCanvas.width, persistent2DCanvas.height);
    
    // Setze Dirty-Flag für 2D-Grafiken
    window.RetroGraphics._graphics2DDirty = true;
}

function handleClearScreen() {
    // Clear the persistent 2D graphics canvas
    if (persistent2DContext && persistent2DCanvas) {
        persistent2DContext.clearRect(0, 0, persistent2DCanvas.width, persistent2DCanvas.height);
    }
    
    // Also clear the main graphics canvas
    if (persistentGraphicsContext && persistentGraphicsCanvas) {
        persistentGraphicsContext.clearRect(0, 0, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
    }
    
    // Clear vectors
    if (window.vectorManager && typeof window.vectorManager.clearAllVectorObjects3D === 'function') {
        window.vectorManager.clearAllVectorObjects3D();
        
        // Nach dem Löschen der Vektoren explizit neu rendern
        if (typeof window.vectorManager.renderVectors === 'function') {
            window.vectorManager.renderVectors(persistentGraphicsContext, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
        }
    }
      // Clear sprites - nur Instanzen löschen, Definitionen behalten
    if (window.spriteManager && typeof window.spriteManager.clearActiveSpriteInstances === 'function') {
        window.spriteManager.clearActiveSpriteInstances();
    }
    
    // Clear particles - alle aktiven Emitter stoppen
    if (window.particleManager && typeof window.particleManager.clearAllParticles === 'function') {
        window.particleManager.clearAllParticles();
    }
    
    // Setze Dirty-Flag für sofortige Anzeige des Clear-Vorgangs
    window.RetroGraphics._graphics2DDirty = true;
}

// Vektor-Grafik-Handler
function handleUpdateVector(data) {
    // Debug-Ausgaben nur bei aktiviertem Debug-Flag
    if (window.CRT_CONFIG && window.CRT_CONFIG.DEBUG_VECTOR_MANAGER) {

    }
    
    if (window.vectorManager && typeof window.vectorManager.handleUpdateVector3D === 'function') {
        const result = window.vectorManager.handleUpdateVector3D(data);
        if (window.CRT_CONFIG && window.CRT_CONFIG.DEBUG_VECTOR_MANAGER) {

        }
        return result;
    } else {

        return false;
    }
}

function clearAllVectors() {
    if (window.CRT_CONFIG && window.CRT_CONFIG.DEBUG_VECTOR_MANAGER) {

    }
    
    if (window.vectorManager && typeof window.vectorManager.clearAllVectorObjects3D === 'function') {
        window.vectorManager.clearAllVectorObjects3D();
        if (window.CRT_CONFIG && window.CRT_CONFIG.DEBUG_VECTOR_MANAGER) {

        }
    } else {

    }
}

// Moduswechsel-Handler
function handleModeChange(modeData) {
    if (modeData && modeData.mode === 'TEXT') {

    } else if (modeData && modeData.mode === 'GRAPHICS') {

    }
}

// Debug-Funktion hinzufügen
function debugGraphicsState() {
    // Debug output removed for production
}

// Erweiterte Debug-Funktion
function debugCanvasLayers() {
    // Debug output removed for production
    
    // Prüfe unser 2D-Canvas spezifisch
    if (persistent2DCanvas) {
        // Debug output removed for production
        
        // Test: Male ein großes rotes Rechteck zur Sichtbarkeitsprüfung
        const ctx = persistent2DContext;
        ctx.fillStyle = '#FF0000';
        ctx.fillRect(0, 0, 200, 200);
        // Debug output removed for production
    }
}

window.debugCanvasLayers = debugCanvasLayers;

// Debug für 2D-zu-Graphics-Canvas Transfer
function debugCanvasTransfer() {
    // Debug output removed for production
    
    if (persistent2DCanvas && persistentGraphicsCanvas) {
        // Teste den Transfer manuell
        persistentGraphicsContext.clearRect(0, 0, persistentGraphicsCanvas.width, persistentGraphicsCanvas.height);
        persistentGraphicsContext.drawImage(persistent2DCanvas, 0, 0);
          // Teste auch die Texture-Update
        if (graphicsSpriteTexture) {
            graphicsSpriteTexture.needsUpdate = true;
            // Debug output removed for production
        }
        
        // Debug output removed for production
        
        // Trigger rendering
        if (window.RetroGraphics && window.RetroGraphics.animateCRT) {
            // Debug output removed for production
            // Ein einzelner Frame wird gerendert
        }
    } else {
        console.error("- Canvas nicht verfügbar:", {
            persistent2D: !!persistent2DCanvas,
            persistentGraphics: !!persistentGraphicsCanvas
        });
    }
}

window.debugCanvasTransfer = debugCanvasTransfer;

// Debug-Funktion global verfügbar machen
window.debugGraphicsState = debugGraphicsState;

// Globale Funktionen über window.RetroGraphics verfügbar machen
window.RetroGraphics.initGraphicsPipeline = initGraphicsPipeline;
window.RetroGraphics.animateCRT = animateCRT;
window.RetroGraphics.forceTextTextureUpdate = forceTextTextureUpdate;
window.RetroGraphics.recreateMaterial = recreateMaterial;
window.RetroGraphics.updateTextTexture = updateTextTexture;

// Grafik-Handler
window.RetroGraphics.handlePlot = handlePlot;
window.RetroGraphics.handleLine = handleLine;
window.RetroGraphics.handleRect = handleRect;
window.RetroGraphics.handleCircle = handleCircle;
window.RetroGraphics.handleFill = handleFill;
window.RetroGraphics.handleClearScreen = handleClearScreen;

// Vektor-Handler
window.RetroGraphics.handleUpdateVector = handleUpdateVector;
window.RetroGraphics.clearAllVectors = clearAllVectors;

// Moduswechsel-Handler
window.RetroGraphics.handleModeChange = handleModeChange;

// Signalisiere, dass RetroGraphics initialisiert und bereit ist
window.retroGraphicsFullyInitialized = true;
const retroGraphicsReadyEvent = new CustomEvent('retrographicsready', { detail: { timestamp: Date.now() } });
document.dispatchEvent(retroGraphicsReadyEvent);

// MCP Evil Effect - Dramatically increases noise intensity for 1 second, then fades back over 5 seconds
function triggerEvilEffect() {
    if (!crtMaterial || !crtMaterial.uniforms || !crtMaterial.uniforms.noiseIntensity) {
        console.warn("[EVIL-EFFECT] CRT material or noise intensity uniform not available");
        return;
    }    // Store original noise intensity
    const originalIntensity = window.CONFIG ? window.CONFIG.CRT_EFFECTS.NOISE_INTENSITY : 4.0;
    const maxIntensity = 50.0; // Maximum evil intensity - very dramatic!
    
    // Debug output removed for production
    
    // Immediately set to maximum intensity
    crtMaterial.uniforms.noiseIntensity.value = maxIntensity;
    
    // After 1 second, start fading back to original over 5 seconds
    setTimeout(() => {
        const fadeDuration = 5000; // 5 seconds in milliseconds
        const startTime = Date.now();
        
        function fadeStep() {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / fadeDuration, 1.0);
            
            // Ease-out function for smooth transition
            const easedProgress = 1 - Math.pow(1 - progress, 3);
            
            // Interpolate from max back to original
            const currentIntensity = maxIntensity + (originalIntensity - maxIntensity) * easedProgress;
            crtMaterial.uniforms.noiseIntensity.value = currentIntensity;
            
            if (progress < 1.0) {
                requestAnimationFrame(fadeStep);
            } else {                // Ensure we end exactly at the original value
                crtMaterial.uniforms.noiseIntensity.value = originalIntensity;
                // Debug output removed for production
            }
        }
        
        fadeStep();
    }, 1000); // Wait 1 second before starting fade
}

// Export the evil effect function
window.RetroGraphics.triggerEvilEffect = triggerEvilEffect;

// Temporäre Lösung: Direkt auf Terminal-Canvas zeichnen
function debugDirectDraw() {
    // Debug output removed for production
    
    const terminalCanvas = document.getElementById('terminalCanvas');
    if (terminalCanvas) {
        const ctx = terminalCanvas.getContext('2d');
          // Zeichne direkt ein grünes Rechteck
        ctx.fillStyle = '#00FF00';
        ctx.fillRect(10, 10, 100, 100);
        
        // Debug output removed for production
          return true;
    } else {
        console.error("- Terminal-Canvas nicht gefunden!");
        return false;
    }
}

window.debugDirectDraw = debugDirectDraw;

// 2D Physics Integration System
// Store 2D graphics objects that can be moved by physics
window.RetroGraphics._physicsObjects = new Map(); // id -> { type, originalData, currentX, currentY }

// Function to register a 2D graphics object for physics updates
window.RetroGraphics.registerPhysicsObject = function(id, type, data) {
    this._physicsObjects.set(id, {
        type: type,
        originalData: { ...data },
        currentX: data.x,
        currentY: data.y,
        rotation: 0
    });
    console.log(`[RETROGRAPHICS] Registered physics object ${id} of type ${type} at (${data.x}, ${data.y})`);
    console.log(`[RETROGRAPHICS] Total physics objects: ${this._physicsObjects.size}`);
};

// Function to update a 2D graphics object position from physics
window.RetroGraphics.updatePhysicsObject = function(id, newX, newY, rotation = 0) {
    const obj = this._physicsObjects.get(id);
    if (!obj) {
        console.warn(`[RETROGRAPHICS] Physics object ${id} not found for update`);
        return;
    }
    
    // Update position
    obj.currentX = newX;
    obj.currentY = newY;
    obj.rotation = rotation;
    
    // Trigger redraw of all physics objects
    this.redrawPhysicsObjects();
};

// Function to redraw all physics-controlled 2D objects
window.RetroGraphics.redrawPhysicsObjects = function() {
    if (!persistent2DCanvas || !persistent2DContext) {
        console.warn("[RETROGRAPHICS] Canvas not available for physics redraw");
        return;
    }
    
    console.log(`[RETROGRAPHICS] Redrawing ${this._physicsObjects.size} physics objects`);
    
    // Clear only the dynamic objects area (or full canvas for simplicity)
    const ctx = persistent2DContext;
    
    // For now, we'll clear the entire canvas and redraw everything
    // This is simple but not optimal - could be improved with layers
    ctx.clearRect(0, 0, persistent2DCanvas.width, persistent2DCanvas.height);
    
    // Redraw all physics objects at their current positions
    for (const [id, obj] of this._physicsObjects) {
        const data = {
            ...obj.originalData,
            x: obj.currentX,
            y: obj.currentY
        };
        
        console.log(`[RETROGRAPHICS] Redrawing ${obj.type} ${id} at (${obj.currentX}, ${obj.currentY})`);
        
        if (obj.type === 'CIRCLE') {
            this.drawCircleAtPosition(data);
        } else if (obj.type === 'RECT') {
            this.drawRectAtPosition(data);
        }
        // Add more types as needed
    }
    
    // Set dirty flag to trigger rendering update
    this._graphics2DDirty = true;
    console.log(`[RETROGRAPHICS] Set _graphics2DDirty = true`);
};

// Helper functions to draw specific shapes
window.RetroGraphics.drawCircleAtPosition = function(data) {
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
    
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.beginPath();
    ctx.arc(Math.floor(data.x), Math.floor(data.y), Math.floor(data.radius), 0, 2 * Math.PI);
    
    if (data.fill) {
        ctx.fill();
    } else {
        ctx.stroke();
    }
};

window.RetroGraphics.drawRectAtPosition = function(data) {
    const ctx = persistent2DContext;
    let color = getColorFromData(data);
    
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    
    if (data.fill) {
        ctx.fillRect(Math.floor(data.x), Math.floor(data.y), Math.floor(data.width), Math.floor(data.height));
    } else {
        ctx.strokeRect(Math.floor(data.x), Math.floor(data.y), Math.floor(data.width), Math.floor(data.height));
    }
};
