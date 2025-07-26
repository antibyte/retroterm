/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// sfxManager.js - Manages PLAYSFX commands for TinyBASIC using sfxr.js

// Global variables
let sfxLibrary = null;

// Use the original SFXR library for authentic sound generation
function generateSound(params, algorithm = null) {
    try {
        console.log(`[SFX-MANAGER] Generating sound with params:`, params, 'algorithm:', algorithm);
        
        // Check if SFXR library is available
        if (typeof jsfxr === 'undefined' || typeof jsfxr.SoundEffect === 'undefined') {
            console.error('[SFX-MANAGER] SFXR library not available');
            return null;
        }

        let soundEffect;
        
        if (algorithm) {
            // Use built-in algorithm
            console.log(`[SFX-MANAGER] Using built-in algorithm: ${algorithm}`);
            soundEffect = new jsfxr.SoundEffect();
            
            // Apply the algorithm if it exists
            if (typeof soundEffect.params[algorithm] === 'function') {
                soundEffect.params[algorithm]();
            } else {
                console.error(`[SFX-MANAGER] Algorithm ${algorithm} not found`);
                return null;
            }
        } else {
            // Use custom parameters from JSON
            console.log(`[SFX-MANAGER] Using custom parameters from JSON`);
            soundEffect = new jsfxr.SoundEffect(params);
        }

        // Generate the sound wave
        const wave = soundEffect.generate();
        if (!wave) {
            console.error('[SFX-MANAGER] SoundEffect.generate() returned null');
            return null;
        }

        console.log(`[SFX-MANAGER] SFXR generated wave object:`, wave);

        // The SFXR library should have its own audio playback method
        if (wave.getAudio && typeof wave.getAudio === 'function') {
            return {
                play: function() {
                    return new Promise(function(resolve, reject) {
                        try {
                            const audioElement = wave.getAudio();
                            if (audioElement && audioElement.play) {
                                audioElement.onended = resolve;
                                audioElement.play();
                            } else {
                                reject(new Error('No audio element available'));
                            }
                        } catch (e) {
                            reject(e);
                        }
                    });
                }
            };
        } else {
            console.error('[SFX-MANAGER] Wave object does not have getAudio method');
            return null;
        }
    } catch (error) {
        console.error('[SFX-MANAGER] Error generating sound:', error);
        console.error('[SFX-MANAGER] Error stack:', error.stack);
        console.error('[SFX-MANAGER] Sound params that caused error:', params);
        return null;
    }
}

// Initialize the SFX manager
function initSFXManager() {
    // Load SFX library from assets
    loadSFXLibrary();
    
    console.log('[SFX-MANAGER] SFX Manager initialized with built-in sound generator');
}

// Load SFX library from JSON file
async function loadSFXLibrary() {
    try {
        // Load main library and individual effect files
        sfxLibrary = {};
        
        const effectFiles = ['explosion', 'shoot', 'laser', 'coin', 'powerup', 'hit', 'jump', 'synth', 'special'];
        
        for (const effect of effectFiles) {
            try {
                const response = await fetch(`/assets/${effect}.json`);
                if (response.ok) {
                    const effectData = await response.json();
                    // Store JSON objects directly - no conversion needed
                    sfxLibrary[effect] = effectData;
                    console.log(`[SFX-MANAGER] Loaded ${effect} with ${sfxLibrary[effect].length} variants`);
                } else {
                    console.warn(`[SFX-MANAGER] Could not load ${effect}.json`);
                }
            } catch (error) {
                console.warn(`[SFX-MANAGER] Error loading ${effect}.json:`, error);
            }
        }
        
        console.log('[SFX-MANAGER] SFX library loading complete');
    } catch (error) {
        console.warn('[SFX-MANAGER] Error loading SFX library:', error);
        sfxLibrary = {};
    }
}

// Play sound effect
function playSFX(effect, variant = -1) {
    console.log(`[SFX-MANAGER] playSFX called with effect: ${effect}, variant: ${variant}`);
    
    let sfxData = null;

    if (variant === -1) {
        // Use built-in algorithms
        const algorithm = getBuiltinAlgorithm(effect);
        console.log(`[SFX-MANAGER] Using builtin algorithm for ${effect}:`, algorithm);
        
        if (algorithm) {
            try {
                console.log(`[SFX-MANAGER] Generating sound with algorithm:`, algorithm);
                // Generate and play the sound using algorithm
                const audio = generateSound(null, algorithm);
                if (audio) {
                    audio.play().catch(e => {
                        console.warn('[SFX-MANAGER] Could not play audio:', e);
                    });
                    console.log(`[SFX-MANAGER] Successfully playing ${effect} sound (builtin algorithm)`);
                } else {
                    console.error('[SFX-MANAGER] generateSound returned null for:', effect, variant);
                }
            } catch (error) {
                console.error('[SFX-MANAGER] Error generating sound:', error);
            }
        } else {
            console.warn('[SFX-MANAGER] No builtin algorithm found for effect:', effect);
        }
    } else {
        // Use custom variant from JSON library
        sfxData = getCustomSFX(effect, variant);
        console.log(`[SFX-MANAGER] Using custom variant ${variant} for ${effect}:`, sfxData);
        
        if (sfxData) {
            try {
                console.log(`[SFX-MANAGER] Generating sound with custom data:`, typeof sfxData, sfxData);
                // Generate and play the sound using custom parameters
                const audio = generateSound(sfxData);
                if (audio) {
                    audio.play().catch(e => {
                        console.warn('[SFX-MANAGER] Could not play audio:', e);
                    });
                    console.log(`[SFX-MANAGER] Successfully playing ${effect} sound (variant ${variant})`);
                } else {
                    console.error('[SFX-MANAGER] generateSound returned null for:', effect, variant);
                }
            } catch (error) {
                console.error('[SFX-MANAGER] Error generating sound:', error);
            }
        } else {
            console.warn('[SFX-MANAGER] No custom SFX data found for effect:', effect, 'variant:', variant);
            console.log('[SFX-MANAGER] Available effects in library:', Object.keys(sfxLibrary || {}));
        }
    }
}

// Get built-in algorithm names for SFXR
function getBuiltinAlgorithm(effect) {
    const algorithms = {
        'explosion': 'explosion',
        'shoot': 'laserShoot',
        'laser': 'laserShoot',
        'coin': 'pickupCoin',
        'powerup': 'powerUp',
        'hit': 'hitHurt',
        'jump': 'jump',
        'synth': 'synth',
        'special': 'tone',
        'random': 'random'
    };

    return algorithms[effect] || null;
}

// Convert JSON object format to sfxr array format
function convertToSfxrArrays(jsonData) {
    // If it's already an array of objects, convert each one
    if (Array.isArray(jsonData)) {
        return jsonData.map(convertSingleObject);
    }
    
    // If it's a single object, convert and wrap in array
    return [convertSingleObject(jsonData)];
}

// Convert single JSON object to sfxr parameter array
function convertSingleObject(obj) {
    if (obj.oldParams) {
        // New extended parameter format
        return [
            obj.wave_type || 0,
            0, // unused
            obj.p_base_freq || 0.3,
            obj.p_freq_limit || 0,
            obj.p_freq_ramp || 0,
            obj.p_freq_dramp || 0,
            obj.p_duty || 0,
            obj.p_duty_ramp || 0,
            0, // vol_sweep placeholder
            obj.p_vib_strength || 0,
            obj.p_vib_speed || 0,
            0, // vib_delay
            obj.p_env_attack || 0,
            obj.p_env_sustain || 0.3,
            obj.p_env_punch || 0,
            obj.p_env_decay || 0.4,
            obj.p_lpf_resonance || 0,
            obj.p_lpf_freq || 1,
            obj.p_lpf_ramp || 0,
            obj.p_hpf_freq || 0,
            obj.p_hpf_ramp || 0,
            obj.p_pha_offset || 0,
            obj.p_pha_ramp || 0,
            obj.p_repeat_speed || 0,
            obj.sound_vol || 0.5
        ];
    } else {
        // Simple array format - return as-is
        return obj;
    }
}

// Get custom SFX from JSON library
function getCustomSFX(effect, variant) {
    if (!sfxLibrary || !sfxLibrary[effect] || !sfxLibrary[effect][variant - 1]) {
        return null;
    }
    
    // Return the raw JSON object - the sound generator now handles this directly
    return sfxLibrary[effect][variant - 1];
}

// Generate random sound effect
function generateRandomSFX() {
    const waveform = Math.floor(Math.random() * 4);  // 0=square, 1=sawtooth, 2=sine, 3=noise
    const baseFreq = 0.1 + Math.random() * 0.6;     // Base frequency
    const freqRamp = (Math.random() - 0.5) * 0.8;   // Frequency slide
    const sustain = 0.1 + Math.random() * 0.6;      // Sustain time
    const decay = 0.1 + Math.random() * 0.8;        // Decay time
    const punch = Math.random() * 0.5;              // Envelope punch
    const attack = Math.random() * 0.3;             // Attack time
    const vibStrength = Math.random() * 0.3;        // Vibrato strength
    const vibSpeed = Math.random() * 0.5;           // Vibrato speed
    const duty = 0.3 + Math.random() * 0.4;         // Square wave duty cycle
    const volume = 0.3 + Math.random() * 0.4;       // Master volume
    
    return [
        waveform,          // wave_type
        0,                 // unused
        baseFreq,          // p_base_freq
        0,                 // p_freq_limit
        freqRamp,          // p_freq_ramp (slide)
        0,                 // p_freq_dramp
        duty,              // p_duty
        0,                 // p_duty_ramp
        0,                 // vol_sweep
        vibStrength,       // p_vib_strength
        vibSpeed,          // p_vib_speed
        0,                 // vib_delay
        attack,            // p_env_attack
        sustain,           // p_env_sustain
        punch,             // p_env_punch
        decay,             // p_env_decay
        0,                 // p_lpf_resonance
        1,                 // p_lpf_freq
        0,                 // p_lpf_ramp
        0,                 // p_hpf_freq
        0,                 // p_hpf_ramp
        0,                 // p_pha_offset
        0,                 // p_pha_ramp
        0,                 // p_repeat_speed
        volume             // sound_vol
    ];
}

// Handle SFX command from backend
function handleSFXCommand(data) {
    if (!data || !data.command) {
        console.error('[SFX-MANAGER] Invalid SFX command data');
        return false;
    }

    switch (data.command) {
        case 'PLAY_SFX':
            return handlePlaySFX(data.data);
        default:
            console.warn('[SFX-MANAGER] Unknown SFX command:', data.command);
            return false;
    }
}

// Handle PLAY_SFX command
function handlePlaySFX(data) {
    if (!data || !data.effect) {
        console.error('[SFX-MANAGER] PLAY_SFX: Missing effect data');
        return false;
    }

    const effect = data.effect;
    const variant = data.variant || -1;

    playSFX(effect, variant);
    return true;
}

// Export functions for global access
window.sfxManager = {
    initSFXManager,
    playSFX,
    handleSFXCommand,
    loadSFXLibrary,
    getBuiltinAlgorithm,
    getCustomSFX
};

// Auto-initialize when loaded
if (typeof window !== 'undefined') {
    console.log('[SFX-MANAGER] sfxManager.js loaded, initializing with built-in sound generator...');
    initSFXManager();
}