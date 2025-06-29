/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// config.js - Globale Konfiguration für das Retro-Terminal
window.CONFIG = {
    // Text-Canvas Auflösung (wird vom WebGL-System skaliert)
    VIRTUAL_CRT_WIDTH: 800,    // Ursprüngliche Breite
    VIRTUAL_CRT_HEIGHT: 600,   // Ursprüngliche Höhe
    TEXT_COLS: 80,             // Mehr Spalten für schärferen Text
    TEXT_ROWS: 24,             // Mehr Zeilen für schärferen Text
    
    // GRAFIKMODUS bleibt unverändert
    GRAPHICS_WIDTH: 640,
    GRAPHICS_HEIGHT: 480,
    
    // SCHRIFT-KONFIGURATION
    FONT_SIZE_PX: 17,          // Größere Schrift für 760×560 Canvas
    FONT_FAMILY: "monospace", // VT323 als primären Font, Fallback auf generischen Monospace
    
    // AUTOMATISCH BERECHNETE PADDING-WERTE (werden zur Laufzeit bestimmt)
    SCREEN_PADDING_LEFT: 8,   // Wird automatisch für perfekte Zentrierung berechnet
    SCREEN_PADDING_TOP: 8,    // Wird automatisch für perfekte Zentrierung berechnet  
    SCREEN_PADDING_RIGHT: 8,  // Wird automatisch für perfekte Zentrierung berechnet
    SCREEN_PADDING_BOTTOM: 8, // Wird automatisch für perfekte Zentrierung berechnet
    SPRITE_SIZE: 32,
    MAX_SPRITES: 256,
    MAX_VECTORS: 256,
      // Helligkeitsstufen optimiert für den Textfarbton '#5FFF5F'
    // Die höchste Helligkeit (15) entspricht exakt der Textfarbe
    BRIGHTNESS_LEVELS: [
        '#000000', // Schwarz
        '#001500', '#002500', '#003500', '#004500', // Sehr dunkles Grün
        '#005500', '#006000', '#007000', '#008000', // Dunkles Grün
        '#009000', '#00A000', '#00B000', '#00C000', // Mittleres Grün 
        '#00D000', '#00E000', '#5FFF5F'             // Das letzte entspricht der Textfarbe
    ],    // CRT-SHADER-EFFEKTE
    CRT_EFFECTS: {
        // Scanlines (horizontale Linien)
        SCANLINES_ENABLED: true,        // true / false
        SCANLINES_INTENSITY: 0.006,      // Subtile Intensität (0.0 bis 1.0)
        SCANLINES_FREQUENCY: 450.0,     // Dichte der Linien

        // Bildröhren-Wölbung (Barrel Distortion)
        BARREL_DISTORTION_ENABLED: true, // true / false
        BARREL_DISTORTION_STRENGTH: -0.01, // Stärke der Wölbung (-0.1 bis 0.1)

        // Rauschen/Noise (Schnee)
        NOISE_ENABLED: true,            // true / false
        NOISE_INTENSITY: 5.0,           // Stärke des Rauschens (0.0 bis 10.0)
        NOISE_SPEED: 3.5,               // Geschwindigkeit der Rauschanimation

        // Vignette (Abdunklung zu den Rändern)
        VIGNETTE_ENABLED: true,         // true / false
        VIGNETTE_STRENGTH: 0.5,         // Stärke der Abdunklung (0.0 bis 1.0)
        VIGNETTE_RADIUS: 0.85,          // Radius der Vignette (0.0 bis 1.0)

        // Glanzpunkt/Reflexion (z.B. von einer Lampe im Raum)
        GLARE_ENABLED: true,            // true / false
        GLARE_INTENSITY: 0.08,          // Intensität der Reflexion (0.0 bis 1.0)
        GLARE_SIZE: 0.2,                // Größe der Reflexion (0.0 bis 1.0)
        GLARE_POSITION_X: 0.95,         // Horizontale Position (0.0 bis 1.0)
        GLARE_POSITION_Y: 0.95,         // Vertikale Position (0.0 bis 1.0)
        GLARE_FALLOFF: 1.5,             // Wie schnell die Reflexion abfällt (1.0 bis 5.0)
        GLARE_BRIGHTNESS_THRESHOLD: 0.2, // Schwellwert, ab welcher Bildhelligkeit der Glare reduziert wird

        // Afterglow-Effekt (Phosphor-Verblassen wie bei echten CRT-Monitoren)
        AFTERGLOW_ENABLED: true,        // true / false
        AFTERGLOW_PERSISTENCE: 0.91,    // Wie stark das Nachleuchten von Frame zu Frame erhalten bleibt (0.0 bis 0.99). Ein höherer Wert bedeutet längeres Nachleuchten.

        // Burn-In-Effekt (eingebrannte statische Bildelemente)
        BURN_IN_ENABLED: true,         // true / false (noch nicht implementiert, aber als Platzhalter)
        BURN_IN_INTENSITY: 0.4,         // Stärke des Burn-In (0.0 bis 1.0)

        // Hintergrundleuchten der Bildröhre (Phosphor-Grundleuchten)
        BACKGROUND_GLOW: {
            ENABLED: true,                  // true / false
            COLOR: '#003300',               // Ein sichtbares, aber dunkles Grün
            CIRCULAR_FALLOFF_ENABLED: true, // true / false - Leuchten von der Mitte nach außen abschwächen
            CIRCULAR_FALLOFF_RADIUS: 0.7,   // Radius des hellen Bereichs (0.1 bis 1.0)
            CIRCULAR_FALLOFF_INTENSITY: 1.5  // Stärke des Abfalls. Höher = schärfer/konzentrierter. (1.0 - 10.0)
        },

        // Flimmer-Effekt (unregelmäßige Helligkeitsschwankungen)
        FLICKER_ENABLED: true,          // true / false
        FLICKER_INTENSITY: 0.004,        // Stärke des Flimmerns (0.0 bis 0.1)

        // Chromatische Aberration (Farbverschiebung an den Rändern)
        CHROMATIC_ABERRATION_ENABLED: true, // true / false
        CHROMATIC_ABERRATION_STRENGTH: 0.05, // Stärke der Verschiebung (0.0 bis 2.0)

        // Shadow Mask (simuliert die Phosphor-Maske des Monitors)
        SHADOW_MASK_ENABLED: false,      // true / false
        SHADOW_MASK_INTENSITY: 0.02,    // Wie dunkel die Maske ist (0.0 bis 1.0) - Ein höherer Wert kann "Löcher" in Buchstaben verursachen.
        SHADOW_MASK_SCALE: 3.5,         // Größe/Dichte des Maskenmusters (1.0 bis 5.0)

        // Screen Jitter (leichtes Wackeln des Bildes)
        SCREEN_JITTER_ENABLED: true,    // true / false
        SCREEN_JITTER_SPEED: 0.2,       // Geschwindigkeit des Wackelns (0.1 bis 2.0)
        SCREEN_JITTER_AMOUNT: 0.000005,   // Stärke des Wackelns (0.0 bis 0.005)

        // Hum Bar (Störstreifen durch Netzbrummen)
        HUM_BAR_ENABLED: true,          // true / false
        HUM_BAR_INTENSITY: 0.01,         // Wie stark das Rauschen des Streifens ist (0.0 bis 0.2)
        HUM_BAR_SPEED: 0.2,             // Wie schnell der Streifen scrollt (0.1 bis 2.0)
        HUM_BAR_HEIGHT: 0.6,            // Basishöhe des Streifens (0.1 bis 2.0)
        HUM_BAR_HEIGHT_VARIATION_ENABLED: true, // true / false - Ob die Höhe des Balkens flackern soll
        HUM_BAR_HEIGHT_VARIATION_AMOUNT: 0.5,   // Stärke der Höhenänderung (z.B. 0.5 = +/- 50% der Basishöhe)
        HUM_BAR_HEIGHT_VARIATION_SPEED: 4.0,    // Geschwindigkeit des Höhen-Flackerns
        HUM_BAR_FALLOFF_STRENGTH: 1.0,          // Stärke des Helligkeitsabfalls. Höher = steiler. (1.0 - 5.0)
    },    // Debug flags
    DEBUG_VECTOR_MANAGER: false,
    DEBUG_VECTOR_MANAGER_INIT: false,
    DEBUG_RENDER_VECTORS: false,
    DEBUG_VECTOR_PROJECTION: false, // Set to true for detailed projection logs
    DEBUG_MATRIX_OPERATIONS: false, // Set to true for matrix creation/multiplication logs
    DEBUG_SPRITE_RENDER_CALL: false, // Set to true for sprite rendering debug logs
    DEBUG_GRAPHICS_COMMANDS: false, // Temporär aktiviert für CIRCLE-Debug
    SHOW_TEST_CUBE: false, // Testwürfel deaktiviert
    SHOW_AXES: false, // Achsen deaktiviert
    AXES_LENGTH: 50, // Increased from 10 to 50 for better visibility
    AXES_COLOR: '#FFFFFF', // Default color for axes lines if not per-axis colored
};

// Initialisiere CRT_CONFIG als eine tiefe Kopie von CONFIG.
// Dies stellt sicher, dass alle allgemeinen Einstellungen in CRT_CONFIG vorhanden sind.
window.CRT_CONFIG = JSON.parse(JSON.stringify(window.CONFIG));

// VECTOR GRAPHICS SETTINGS für CRT_CONFIG
// Diese fügen Eigenschaften zu CRT_CONFIG hinzu oder überschreiben sie,
// basierend auf der bereits kopierten window.CONFIG.
Object.assign(window.CRT_CONFIG, {
    VECTOR_SHOW_AXES: false, // Achsen deaktiviert
    VECTOR_AXIS_LENGTH: window.CRT_CONFIG.AXES_LENGTH || 50, // Length of the axis lines
    VECTOR_AXIS_COLOR_X: '#FF0000', // Red for X
    VECTOR_AXIS_COLOR_Y: '#00FF00', // Green for Y
    VECTOR_AXIS_COLOR_Z: '#0000FF', // Blue for Z

    VECTOR_GRID_VISIBLE: false, // Show a grid on the XY plane
    VECTOR_GRID_SIZE: 200,    // Overall size of the grid
    VECTOR_GRID_DIVISIONS: 20, // Number of divisions
    VECTOR_GRID_COLOR: '#404040', // Color of the grid lines

    VECTOR_CAMERA_X: 0, // Default X position of the camera
    VECTOR_CAMERA_Y: 0, // Default Y position of the camera
    VECTOR_CAMERA_Z: 50, 
    VECTOR_FOV: 60, 
    VECTOR_FOCAL_LENGTH: null, 
    VECTOR_NEAR_PLANE: 0.1, 
    VECTOR_FAR_PLANE: 1000, 

    VECTOR_DEFAULT_LINE_COLOR: '#FFFFFF', 
    VECTOR_DEFAULT_POINT_SIZE: 2,      

    // Test Cube Configuration
    TEST_CUBE_SIZE: 20, 
    TEST_CUBE_COLOR: '#00FF00', 
    TEST_CUBE_POSITION_X: 0,
    TEST_CUBE_POSITION_Y: 0,
    TEST_CUBE_POSITION_Z: -30, 
    
    // Basis-URL und Bildschirmdimensionen, falls sie spezifisch für CRT_CONFIG sein müssen
    // und nicht von window.CONFIG übernommen werden sollen.
    // Diese überschreiben die Werte aus window.CONFIG, falls vorhanden.
    BASE_URL: "/",
    SCREEN_WIDTH: window.CRT_CONFIG.VIRTUAL_CRT_WIDTH || 640, // Bevorzuge VIRTUAL_CRT_WIDTH
    SCREEN_HEIGHT: window.CRT_CONFIG.VIRTUAL_CRT_HEIGHT || 480, // Bevorzuge VIRTUAL_CRT_HEIGHT
    // GRAPHICS_WIDTH und GRAPHICS_HEIGHT werden bereits von window.CONFIG übernommen.
});

