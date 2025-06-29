/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

// sidPlayer.js - Real C64 SID Music Player for RetroWaves Demo
// Uses jsSID library for authentic SID file playback

(function() {
    'use strict';    // SID Player State
    let currentSidPlayer = null;
    let isPlaying = false;
    let isPaused = false;
    let currentFilename = '';
    let audioContext = null;
    let scriptProcessor = null;
    let gainNode = null;
    
    // Performance-Optimierung für Grafik-intensive Szenen
    let graphicsIntensiveMode = false;
    let audioProcessingPriority = 'normal'; // 'normal' oder 'high'

    // jsSID Integration
    let jsSIDLoaded = false;    // Load jsSID library dynamically
    function loadJSSID() {
        return new Promise((resolve, reject) => {
            if (jsSIDLoaded && window.jsSID) {
                resolve();
                return;
            }            // Try to load from local file first, then fallback
            const script = document.createElement('script');
            script.onload = () => {
                jsSIDLoaded = true;
                resolve();
            };
            script.onerror = () => {
                console.warn('Failed to load local jsSID, using fallback...');
                loadFallbackSID().then(resolve).catch(reject);
            };
            script.src = '/js/jsSID.js';
            document.head.appendChild(script);
        });
    }

    // Fallback SID emulator for when jsSID is not available
    function loadFallbackSID() {
        return new Promise((resolve) => {
            // Create a minimal SID emulator
            window.jsSID = {
                SIDPlayer: class FallbackSIDPlayer {
                    constructor() {
                        this.isLoaded = false;
                        this.sampleRate = 44100;
                        this.channels = 3;
                        this.voices = [
                            { freq: 440, wave: 'sawtooth', volume: 0 },
                            { freq: 554, wave: 'pulse', volume: 0 },
                            { freq: 659, wave: 'triangle', volume: 0 }
                        ];
                        this.phase = [0, 0, 0];
                    }                    loadSID(data) {
                        this.isLoaded = true;
                        // Simple pattern based on data
                        const pattern = Array.from(data.slice(0, 32));
                        this.voices[0].freq = 220 + (pattern[0] || 0);
                        this.voices[1].freq = 330 + (pattern[1] || 0);
                        this.voices[2].freq = 440 + (pattern[2] || 0);
                        return true;
                    }

                    generateSamples(buffer, length) {
                        if (!this.isLoaded) return;

                        for (let i = 0; i < length; i++) {
                            let sample = 0;
                            
                            // Generate audio for each voice
                            for (let v = 0; v < this.channels; v++) {
                                const voice = this.voices[v];
                                if (voice.volume > 0) {
                                    const phaseInc = (voice.freq * 2 * Math.PI) / this.sampleRate;
                                    
                                    let voiceSample = 0;
                                    switch (voice.wave) {
                                        case 'sawtooth':
                                            voiceSample = (this.phase[v] / Math.PI) - 1;
                                            break;
                                        case 'pulse':
                                            voiceSample = this.phase[v] < Math.PI ? 1 : -1;
                                            break;
                                        case 'triangle':
                                            voiceSample = this.phase[v] < Math.PI ? 
                                                (this.phase[v] / Math.PI) * 2 - 1 :
                                                1 - ((this.phase[v] - Math.PI) / Math.PI) * 2;
                                            break;
                                        default:
                                            voiceSample = Math.sin(this.phase[v]);
                                    }
                                    
                                    sample += voiceSample * voice.volume * 0.2;
                                    this.phase[v] += phaseInc;
                                    if (this.phase[v] > 2 * Math.PI) {
                                        this.phase[v] -= 2 * Math.PI;
                                    }
                                }
                            }
                            
                            buffer[i] = sample;
                        }
                    }

                    play() {
                        // Activate voices with random pattern
                        this.voices[0].volume = 0.8;
                        this.voices[1].volume = 0.6;
                        this.voices[2].volume = 0.4;
                    }

                    stop() {
                        this.voices.forEach(voice => voice.volume = 0);
                    }
                }
            };
            jsSIDLoaded = true;            resolve();
        });
    }
    
    // Initialize SID Music Support in RetroSound
    function initSidSupport() {
        if (!window.RetroSound) {
            window.RetroSound = {};
        }

        // Initialize audio context
        audioContext = new (window.AudioContext || window.webkitAudioContext)();
        gainNode = audioContext.createGain();
        gainNode.gain.setValueAtTime(0.5, audioContext.currentTime);
        gainNode.connect(audioContext.destination);        // Load SID library
        loadJSSID().then(() => {
            console.log('[SID-PLAYER] SID Player initialized successfully');
        }).catch(error => {
            console.error('[SID-PLAYER] Failed to initialize SID Player:', error);
        });

        // Load SID file from server
        async function loadSidFile(filename) {
            try {
                // Try different paths for the SID file using the correct API format
                const possiblePaths = [
                    `/api/file?path=${encodeURIComponent(filename)}`,
                    `/api/file?path=${encodeURIComponent(filename + '.sid')}`,
                    `/api/file?path=examples/${encodeURIComponent(filename)}`,
                    `/api/file?path=examples/${encodeURIComponent(filename + '.sid')}`,
                    `/examples/${filename}`,
                    `/examples/${filename}.sid`
                ];

                let sidData = null;
                let loadedPath = null;

                for (const path of possiblePaths) {
                    try {
                        const response = await fetch(path);
                        if (response.ok) {
                            sidData = await response.arrayBuffer();
                            loadedPath = path;
                            break;
                        }
                    } catch (e) {
                        // Continue to next path
                    }
                }

                if (!sidData) {
                    console.error(`SID file not found: ${filename}`);
                    return false;
                }

                // Ensure any existing player is completely cleaned up
                if (currentSidPlayer) {
                    try {
                        currentSidPlayer.stop();
                    } catch (e) {
                        console.warn('Error stopping old SID player:', e);
                    }
                    currentSidPlayer = null;
                }

                // Create new jsSID player instance 
                // jsSID expects buffer length and noise amount parameters
                currentSidPlayer = new window.jsSID(4096, 0.0005);
                
                // jsSID uses loadstart method with URL
                // Since we have ArrayBuffer, we need to create a blob URL
                const blob = new Blob([sidData], { type: 'application/octet-stream' });
                const url = URL.createObjectURL(blob);
                
                // Load with subtune 0 (default)
                currentSidPlayer.loadstart(url, 0);
                
                // Clean up blob URL after a moment
                setTimeout(() => URL.revokeObjectURL(url), 1000);

                return true;

            } catch (error) {
                console.error('Error loading SID file:', error);
                return false;
            }
        }

        // Open and load a SID file
        window.RetroSound.openSidMusic = function(filename) {
            // Stop any currently playing music
            if (currentSidPlayer) {
                window.RetroSound.stopSidMusic();
            }

            return loadJSSID().then(() => {
                return loadSidFile(filename).then(success => {
                    if (success) {
                        currentFilename = filename;
                    } else {
                        console.error(`Failed to load SID music: ${filename}`);
                    }
                    return success;
                });
            });
        };

        // Play the loaded SID music
        window.RetroSound.playSidMusic = function() {
            if (!currentSidPlayer) {
                console.warn('No SID music loaded. Use MUSIC OPEN first.');
                return;
            }

            if (isPaused) {
                // Resume from pause
                startAudioProcessing();
                isPaused = false;
                isPlaying = true;
            } else if (!isPlaying) {
                // Start from beginning
                currentSidPlayer.play();
                startAudioProcessing();
                isPlaying = true;
            }
        };
        
        // Stop SID music playback
        window.RetroSound.stopSidMusic = function() {
            if (currentSidPlayer) {
                currentSidPlayer.stop();
                stopAudioProcessing();
                isPlaying = false;
                isPaused = false;
                currentSidPlayer = null; // Clear the player reference
                currentFilename = '';     // Clear the filename
            }        };
        
        // Audio priority control for graphics-intensive scenes
        window.RetroSound.setAudioPriority = function(priority) {
            audioProcessingPriority = priority;
        };

        // Toggle graphics-intensive mode
        window.RetroSound.setGraphicsIntensiveMode = function(enabled) {
            graphicsIntensiveMode = enabled;
            if (enabled) {
                window.RetroSound.setAudioPriority('high');
            } else {
                window.RetroSound.setAudioPriority('normal');
            }        };
          // Pause SID music playbook
        window.RetroSound.pauseSidMusic = function() {
            if (isPlaying) {
                stopAudioProcessing();
                isPlaying = false;
                isPaused = true;
            }
        };

        // Resume SID music playback
        window.RetroSound.resumeSidMusic = function() {
            if (isPaused && currentSidPlayer) {
                startAudioProcessing();
                isPaused = false;
                isPlaying = true;
            }
        };

        // Check if SID music is playing
        window.RetroSound.isSidMusicPlaying = function() {
            return isPlaying;
        };        // Get current SID music info
        window.RetroSound.getSidMusicInfo = function() {
            return {
                filename: currentFilename,
                isPlaying: isPlaying,
                isPaused: isPaused,                currentTime: 0, // TODO: Implement time tracking
                duration: 0     // TODO: Implement duration tracking
            };
        };

    // Start audio processing
    function startAudioProcessing() {
        if (scriptProcessor || !audioContext || !currentSidPlayer) return;

        if (audioContext.state === 'suspended') {
            audioContext.resume();
        }        // Create script processor for audio generation
        // Note: ScriptProcessorNode is deprecated but still widely supported
        // AudioWorkletNode would be the modern alternative but requires more complex setup
        
        // Kleinere Buffer-Größe für weniger Latenz bei intensiver Grafik
        const bufferSize = 2048; // Reduziert von 4096 für bessere Responsivität
        
        // Temporarily suppress deprecation warnings for ScriptProcessorNode
        const originalWarn = console.warn;
        console.warn = function(message) {
            if (typeof message === 'string' && message.includes('ScriptProcessorNode')) {
                return; // Suppress ScriptProcessorNode deprecation warnings
            }
            originalWarn.apply(console, arguments);
        };
        
        scriptProcessor = audioContext.createScriptProcessor(bufferSize, 0, 2);
        
        // Restore original console.warn
        console.warn = originalWarn;
        
        // Audio-Priorität erhöhen durch Timeouts minimieren
        let audioProcessingActive = false;
        
        scriptProcessor.onaudioprocess = function(event) {
            // Verhindere überlappende Audio-Processing-Calls
            if (audioProcessingActive || !isPlaying || !currentSidPlayer) {
                // Silence als Fallback
                const outputL = event.outputBuffer.getChannelData(0);
                const outputR = event.outputBuffer.getChannelData(1);
                outputL.fill(0);
                outputR.fill(0);
                return;
            }
            
            audioProcessingActive = true;
            
            try {
                const outputL = event.outputBuffer.getChannelData(0);
                const outputR = event.outputBuffer.getChannelData(1);                // Generate audio samples
                if (currentSidPlayer.generateSamples) {
                    // For stereo output
                    const monoBuffer = new Float32Array(bufferSize);
                    currentSidPlayer.generateSamples(monoBuffer, bufferSize);
                    
                    // Copy mono to stereo
                    for (let i = 0; i < bufferSize; i++) {
                        outputL[i] = monoBuffer[i];
                        outputR[i] = monoBuffer[i];
                    }
                } else {
                    // Fallback: silence
                    outputL.fill(0);
                    outputR.fill(0);
                }
            } catch (error) {
                // Bei Fehlern Audio stoppen um Stottern zu vermeiden
                console.warn('SID audio processing error:', error);
                const outputL = event.outputBuffer.getChannelData(0);
                const outputR = event.outputBuffer.getChannelData(1);
                outputL.fill(0);
                outputR.fill(0);
            } finally {
                audioProcessingActive = false;
            }
        };

        scriptProcessor.connect(gainNode);    }

    // Stop audio processing
    function stopAudioProcessing() {
        if (scriptProcessor) {
            try {
                scriptProcessor.disconnect();
            } catch (e) {
                console.warn('AudioNode already disconnected:', e);
            }
            scriptProcessor = null;
        }    }
    
    // End of initSidSupport function
    }

    // Export sidPlayer API to window
    window.sidPlayer = {
        loadSID: function(url, filename) {
            return window.RetroSound.openSidMusic(filename || 'unknown.sid');
        },
        stop: function() {
            return window.RetroSound.stopSidMusic();
        },
        pause: function() {
            return window.RetroSound.pauseSidMusic();
        },
        resume: function() {
            return window.RetroSound.resumeSidMusic();
        },
        isPlaying: function() {
            return isPlaying;
        }
    };

    // Auto-stop music when page is unloaded
    window.addEventListener('beforeunload', function() {
        if (window.RetroSound && window.RetroSound.stopSidMusic) {
            window.RetroSound.stopSidMusic();
        }
    });    // Handle audio context suspension/resumption
    document.addEventListener('click', function() {
        if (audioContext && audioContext.state === 'suspended') {
            audioContext.resume();
        }
    }, { once: true });

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initSidSupport);
    } else {
        initSidSupport();
    }

})();
