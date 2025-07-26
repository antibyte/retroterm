/**
 * Simple jsfxr implementation
 * Generates sound effects using Web Audio API
 */

(function() {
    "use strict";

    function jsfxr(params) {
        try {
            // Validate input
            if (!params || !Array.isArray(params)) {
                console.error('[jsfxr] Invalid params');
                return null;
            }

            // Extract basic parameters
            var waveType = params[0] || 0;
            var baseFreq = params[2] || 0.3;
            var envSustain = params[13] || 0.3;
            var envDecay = params[15] || 0.4;
            var masterVol = params[24] || 0.5;

            // Create audio context
            var audioContext;
            try {
                audioContext = new (window.AudioContext || window.webkitAudioContext)();
            } catch (e) {
                console.error('[jsfxr] AudioContext not supported');
                return null;
            }

            // Generate simple sound
            var sampleRate = audioContext.sampleRate;
            var duration = (envSustain + envDecay) || 0.5; // Default 0.5 seconds
            var samples = Math.floor(duration * sampleRate);
            
            // Create buffer
            var buffer = audioContext.createBuffer(1, samples, sampleRate);
            var data = buffer.getChannelData(0);

            // Generate waveform
            var frequency = 200 + (baseFreq * 800); // Map to reasonable frequency range
            
            for (var i = 0; i < samples; i++) {
                var t = i / sampleRate;
                var envelope = Math.max(0, 1 - t / duration); // Simple linear decay
                var phase = 2 * Math.PI * frequency * t;
                
                var sample = 0;
                switch (waveType) {
                    case 0: // Square
                        sample = Math.sin(phase) > 0 ? 0.5 : -0.5;
                        break;
                    case 1: // Sawtooth  
                        sample = 2 * (t * frequency % 1) - 1;
                        break;
                    case 2: // Sine
                        sample = Math.sin(phase);
                        break;
                    case 3: // Noise
                        sample = Math.random() * 2 - 1;
                        break;
                    default:
                        sample = Math.sin(phase);
                }
                
                data[i] = sample * envelope * masterVol;
            }

            // Return audio object
            return {
                play: function() {
                    return new Promise(function(resolve, reject) {
                        try {
                            var source = audioContext.createBufferSource();
                            source.buffer = buffer;
                            source.connect(audioContext.destination);
                            source.onended = resolve;
                            source.start(0);
                        } catch (e) {
                            reject(e);
                        }
                    });
                }
            };

        } catch (error) {
            console.error('[jsfxr] Error generating sound:', error);
            return null;
        }
    }

    // Export
    if (typeof module !== 'undefined' && module.exports) {
        module.exports = jsfxr;
    } else {
        window.jsfxr = jsfxr;
        console.log('[jsfxr] Simple jsfxr library loaded successfully');
    }

})();