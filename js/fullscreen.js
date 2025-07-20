/*
 * Fullscreen Management for RetroTerm
 * Automatisches Vollbild beim Start + F11/ESC Shortcuts
 */

window.FullscreenManager = {
    isFullscreen: false,
    autoFullscreenOnStart: false, // Disabled - using doubleclick instead
    
    // Doubleclick activation
    doubleClickActivation: {
        enabled: true,
        clickTimeout: 500, // milliseconds between clicks
        lastClick: 0
    },
    
    // Handle doubleclick activation
    handleDoubleClick: function(event) {
        if (!this.doubleClickActivation.enabled) {
            return false;
        }
        
        const now = Date.now();
        const timeSinceLastClick = now - this.doubleClickActivation.lastClick;
        
        console.log('[FULLSCREEN] Click detected, time since last:', timeSinceLastClick + 'ms');
        
        if (timeSinceLastClick < this.doubleClickActivation.clickTimeout) {
            console.log('[FULLSCREEN] Doubleclick detected - toggling fullscreen');
            this.toggle();
            this.doubleClickActivation.lastClick = 0; // Reset
            return true;
        } else {
            console.log('[FULLSCREEN] First click registered');
            this.doubleClickActivation.lastClick = now;
            return false;
        }
    },
    
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
                    // If user gesture required, don't show error - this is expected for auto-activation
                    if (error.name === 'NotAllowedError') {
                        console.log('[FULLSCREEN] User gesture required - this is normal for auto-activation');
                    }
                });
            } else {
                console.log('[FULLSCREEN] Non-promise API used');
            }
            
            return true;
        } catch (error) {
            console.log('[FULLSCREEN] Error entering fullscreen:', error.name, error.message);
            // If user gesture required, don't treat as error - this is expected for auto-activation
            if (error.name === 'NotAllowedError' || error.message.includes('user gesture')) {
                console.log('[FULLSCREEN] User gesture required - this is normal for auto-activation attempts');
                return false; // Return false but don't log as error
            }
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
        
        // Add simple keyboard shortcuts
        document.addEventListener('keydown', (event) => {
            // F11 key - toggle fullscreen
            if (event.key === 'F11') {
                console.log('[FULLSCREEN] F11 pressed - toggling fullscreen');
                event.preventDefault();
                this.toggle();
            }
            // ESC key - let browser handle normally (exit fullscreen)
            // No special protection needed
        });
        
        // Add doubleclick listener for fullscreen activation
        document.addEventListener('click', (event) => {
            this.handleDoubleClick(event);
        });
        
        console.log('[FULLSCREEN] Manager initialized');
        console.log('[FULLSCREEN] Doubleclick activation:', this.doubleClickActivation.enabled);
        console.log('[FULLSCREEN] API supported:', this.isSupported());
    },
    
    // Show fullscreen status in terminal
    showStatus: function() {
        const status = this.checkState() ? 'ACTIVE' : 'INACTIVE';
        const doubleClickStatus = this.doubleClickActivation.enabled ? 'ON' : 'OFF';
        
        const message = `Fullscreen Mode: ${status}\\nDoubleclick activation: ${doubleClickStatus}\\nPress F11 to toggle\\nPress ESC to exit (normal browser behavior)\\nDoubleclick anywhere to toggle fullscreen`;
        
        if (window.RetroConsole && typeof window.RetroConsole.print === 'function') {
            window.RetroConsole.print(message);
        } else {
            console.log('[FULLSCREEN] Status:', status);
            console.log('[FULLSCREEN] Doubleclick activation:', this.doubleClickActivation.enabled);
        }
    },
    
    // Configure doubleclick activation
    configureDoubleClick: function(enabled, timeout = 500) {
        this.doubleClickActivation.enabled = enabled;
        this.doubleClickActivation.clickTimeout = timeout;
        this.doubleClickActivation.lastClick = 0; // Reset
        
        console.log('[FULLSCREEN] Doubleclick activation configured:', {
            enabled,
            timeout
        });
        
        if (window.RetroConsole && typeof window.RetroConsole.print === 'function') {
            const status = enabled ? 
                `Doubleclick activation enabled (${timeout}ms timeout)` :
                'Doubleclick activation disabled';
            window.RetroConsole.print(`[Fullscreen] ${status}`);
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
            
            window.RetroConsole.addCommand('FULLSCREEN DOUBLECLICK', () => {
                const doubleclick = window.FullscreenManager.doubleClickActivation;
                if (doubleclick.enabled) {
                    window.FullscreenManager.configureDoubleClick(false);
                    window.RetroConsole.print('Doubleclick activation DISABLED');
                } else {
                    window.FullscreenManager.configureDoubleClick(true, 500);
                    window.RetroConsole.print('Doubleclick activation ENABLED - doubleclick anywhere to toggle');
                }
            });
            
            window.RetroConsole.addCommand('FULLSCREEN DOUBLECLICK ON', () => {
                window.FullscreenManager.configureDoubleClick(true, 500);
                window.RetroConsole.print('Doubleclick activation ENABLED - doubleclick anywhere to toggle');
            });
            
            window.RetroConsole.addCommand('FULLSCREEN DOUBLECLICK OFF', () => {
                window.FullscreenManager.configureDoubleClick(false);
                window.RetroConsole.print('Doubleclick activation DISABLED');
            });
        }
    });
}