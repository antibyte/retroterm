/*
 * Fullscreen Management for RetroTerm
 * Automatisches Vollbild beim Start + F11/ESC Shortcuts
 */

window.FullscreenManager = {
    isFullscreen: false,
    autoFullscreenOnStart: true, // Set to false to disable auto-fullscreen
    
    // Check if fullscreen is supported
    isSupported: function() {
        return !!(document.documentElement.requestFullscreen || 
                 document.documentElement.mozRequestFullScreen || 
                 document.documentElement.webkitRequestFullscreen || 
                 document.documentElement.msRequestFullscreen);
    },
    
    // Enter fullscreen mode
    enter: function() {
        console.log('[FULLSCREEN] Attempting to enter fullscreen...');
        
        if (!this.isSupported()) {
            console.log('[FULLSCREEN] Fullscreen API not supported in this browser');
            return false;
        }
        
        // Check if already in fullscreen
        if (this.checkState()) {
            console.log('[FULLSCREEN] Already in fullscreen mode');
            return true;
        }
        
        const element = document.documentElement;
        console.log('[FULLSCREEN] Using element:', element.tagName);
        
        try {
            let promise;
            let method;
            
            if (element.requestFullscreen) {
                method = 'requestFullscreen';
                promise = element.requestFullscreen();
            } else if (element.mozRequestFullScreen) {
                method = 'mozRequestFullScreen';
                promise = element.mozRequestFullScreen();
            } else if (element.webkitRequestFullscreen) {
                method = 'webkitRequestFullscreen';
                promise = element.webkitRequestFullscreen();
            } else if (element.msRequestFullscreen) {
                method = 'msRequestFullscreen';
                promise = element.msRequestFullscreen();
            }
            
            console.log('[FULLSCREEN] Using method:', method);
            
            // Handle promise if returned (modern browsers)
            if (promise && typeof promise.then === 'function') {
                console.log('[FULLSCREEN] Promise-based API detected');
                promise.then(() => {
                    console.log('[FULLSCREEN] Promise resolved - fullscreen activated');
                }).catch(error => {
                    console.log('[FULLSCREEN] Promise rejected:', error.name, error.message);
                });
            } else {
                console.log('[FULLSCREEN] Non-promise API used');
            }
            
            return true;
        } catch (error) {
            console.log('[FULLSCREEN] Error entering fullscreen:', error.name, error.message);
            return false;
        }
    },
    
    // Exit fullscreen mode
    exit: function() {
        try {
            if (document.exitFullscreen) {
                document.exitFullscreen();
            } else if (document.mozCancelFullScreen) {
                document.mozCancelFullScreen();
            } else if (document.webkitExitFullscreen) {
                document.webkitExitFullscreen();
            } else if (document.msExitFullscreen) {
                document.msExitFullscreen();
            }
            return true;
        } catch (error) {
            console.log('[FULLSCREEN] Error exiting fullscreen:', error);
            return false;
        }
    },
    
    // Toggle fullscreen mode
    toggle: function() {
        if (this.isFullscreen) {
            return this.exit();
        } else {
            return this.enter();
        }
    },
    
    // Check current fullscreen state
    checkState: function() {
        this.isFullscreen = !!(document.fullscreenElement || 
                              document.mozFullScreenElement || 
                              document.webkitFullscreenElement || 
                              document.msFullscreenElement);
        return this.isFullscreen;
    },
    
    // Initialize fullscreen event listeners
    init: function() {
        // Listen for fullscreen change events
        const fullscreenEvents = [
            'fullscreenchange',
            'mozfullscreenchange', 
            'webkitfullscreenchange',
            'msfullscreenchange'
        ];
        
        fullscreenEvents.forEach(event => {
            document.addEventListener(event, () => {
                this.checkState();
                console.log('[FULLSCREEN] State changed:', this.isFullscreen ? 'ENTERED' : 'EXITED');
                
                // Trigger viewport recalculation on fullscreen change
                if (window.DynamicViewport && typeof window.DynamicViewport.recalculate === 'function') {
                    setTimeout(() => {
                        window.DynamicViewport.recalculate();
                    }, 100);
                }
            });
        });
        
        // Add keyboard shortcuts
        document.addEventListener('keydown', (event) => {
            console.log('[FULLSCREEN] Keydown event:', event.key);
            
            // F11 key - toggle fullscreen
            if (event.key === 'F11') {
                console.log('[FULLSCREEN] F11 pressed - toggling fullscreen');
                event.preventDefault();
                this.toggle();
            }
            
            // ESC key - exit fullscreen (only if currently in fullscreen)
            if (event.key === 'Escape' && this.isFullscreen) {
                console.log('[FULLSCREEN] ESC pressed - browser will handle exit');
                // Let browser handle ESC for fullscreen exit naturally
                // No need to call this.exit() as browser does it automatically
            }
        });
        
        console.log('[FULLSCREEN] Manager initialized');
        console.log('[FULLSCREEN] Auto-fullscreen on start:', this.autoFullscreenOnStart);
        console.log('[FULLSCREEN] API supported:', this.isSupported());
        
        // Auto-enter fullscreen on start if enabled
        if (this.autoFullscreenOnStart) {
            console.log('[FULLSCREEN] Starting auto-fullscreen setup...');
            this.requestAutoFullscreen();
        } else {
            console.log('[FULLSCREEN] Auto-fullscreen disabled');
        }
    },
    
    // Request auto-fullscreen with user interaction detection
    requestAutoFullscreen: function() {
        console.log('[FULLSCREEN] Starting auto-fullscreen setup...');
        
        // If immediate fullscreen fails, wait for first user interaction
        const autoFullscreenOnInteraction = (event) => {
            console.log('[FULLSCREEN] User interaction detected:', event.type);
            
            // Small delay to ensure the interaction is complete
            setTimeout(() => {
                console.log('[FULLSCREEN] Attempting to enter fullscreen...');
                const success = this.enter();
                if (success) {
                    console.log('[FULLSCREEN] Auto-fullscreen activated on user interaction');
                } else {
                    console.log('[FULLSCREEN] Failed to enter fullscreen');
                }
            }, 100);
        };
        
        // Listen for any user interaction to trigger fullscreen
        console.log('[FULLSCREEN] Setting up event listeners...');
        
        // Test if event listeners are working
        document.addEventListener('click', (e) => {
            console.log('[FULLSCREEN] Click detected at:', e.clientX, e.clientY, 'Target:', e.target.tagName);
        }, { once: false });
        
        document.addEventListener('click', autoFullscreenOnInteraction, { once: true });
        document.addEventListener('keydown', autoFullscreenOnInteraction, { once: true });
        document.addEventListener('touchstart', autoFullscreenOnInteraction, { once: true });
        
        console.log('[FULLSCREEN] Waiting for user interaction to enable auto-fullscreen');
    },
    
    // Show fullscreen status in terminal
    showStatus: function() {
        const status = this.checkState() ? 'ACTIVE' : 'INACTIVE';
        const message = `Fullscreen Mode: ${status}\\nPress F11 to toggle fullscreen`;
        
        if (window.RetroConsole && typeof window.RetroConsole.print === 'function') {
            window.RetroConsole.print(message);
        } else {
            console.log('[FULLSCREEN] Status:', status);
        }
    }
};

// Initialize on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.FullscreenManager.init();
    });
} else {
    // DOM already loaded
    window.FullscreenManager.init();
}

// Add FULLSCREEN command for manual testing
if (window.RetroConsole) {
    // This will be available when RetroConsole is loaded
    document.addEventListener('DOMContentLoaded', () => {
        if (window.RetroConsole && window.RetroConsole.addCommand) {
            window.RetroConsole.addCommand('FULLSCREEN', () => {
                window.FullscreenManager.showStatus();
            });
            
            window.RetroConsole.addCommand('FULLSCREEN ON', () => {
                if (window.FullscreenManager.enter()) {
                    window.RetroConsole.print('Entering fullscreen mode...');
                } else {
                    window.RetroConsole.print('Fullscreen not supported or failed');
                }
            });
            
            window.RetroConsole.addCommand('FULLSCREEN OFF', () => {
                if (window.FullscreenManager.exit()) {
                    window.RetroConsole.print('Exiting fullscreen mode...');
                } else {
                    window.RetroConsole.print('Exit fullscreen failed');
                }
            });
        }
    });
}