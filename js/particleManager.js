// particleManager.js - Manages PARTICLE commands for TinyBASIC

// Particle system constants
const MAX_EMITTERS = 16;
const MAX_PARTICLES_PER_EMITTER = 500;

// Global variables
let particleCanvas = null;
let particleEmitters = [];
let particles = [];
let globalGravity = 0;
let lastFrameTime = 0;

// Initialize the particle manager
function initParticleManager() {
    particleEmitters = [];
    particles = [];
    globalGravity = 0;
    lastFrameTime = performance.now();
    
    // Initialize particle dirty flag
    if (!window.RetroGraphics) {
        window.RetroGraphics = {};
    }
    if (typeof window.RetroGraphics._particlesDirty === 'undefined') {
        window.RetroGraphics._particlesDirty = false;
    }
}

// Particle class
class Particle {
    constructor(x, y, vx, vy, lifetime, type) {
        this.x = x;
        this.y = y;
        this.vx = vx; // Velocity X
        this.vy = vy; // Velocity Y
        this.lifetime = lifetime;
        this.maxLifetime = lifetime;
        this.type = type;
        this.age = 0;
    }
    
    update(deltaTime) {
        // Convert deltaTime from milliseconds to seconds for more reasonable physics
        const dt = deltaTime * 0.001;
        
        // Apply gravity with proper scaling (0-255 -> 0-1 range)
        this.vy += (globalGravity / 255.0 * dt * 30);
        
        // Update position (much slower movement)
        this.x += this.vx * dt * 50;
        this.y += this.vy * dt * 50;
        
        // Update age
        this.age += deltaTime;
        
        // Return true if particle is still alive
        return this.age < this.lifetime;
    }
    
    getAlpha() {
        // Fade from bright to dark over lifetime
        return Math.max(0, 1 - (this.age / this.lifetime));
    }
}

// Particle emitter class
class ParticleEmitter {
    constructor(id, type, pps, speed, lifetime) {
        this.id = id;
        this.type = type;
        this.x = 0;
        this.y = 0;
        this.pps = pps;
        this.speed = speed;
        this.lifetime = lifetime;
        this.visible = false;
        this.positioned = false;
        this.timeSinceLastEmit = 0;
    }
    
    setPosition(x, y) {
        this.x = x;
        this.y = y;
        this.positioned = true;
        this.visible = true;
        window.RetroGraphics._particlesDirty = true;
    }
    
    show() {
        if (this.positioned) {
            this.visible = true;
            window.RetroGraphics._particlesDirty = true;
        }
    }
    
    hide() {
        this.visible = false;
        window.RetroGraphics._particlesDirty = true;
    }
    
    update(deltaTime) {
        if (!this.visible || !this.positioned) return;
        
        this.timeSinceLastEmit += deltaTime;
        
        const emitInterval = 1000 / this.pps; // Milliseconds between emissions
        
        while (this.timeSinceLastEmit >= emitInterval) {
            this.emitParticle();
            this.timeSinceLastEmit -= emitInterval;
        }
    }
    
    emitParticle() {
        if (particles.length >= MAX_PARTICLES_PER_EMITTER * MAX_EMITTERS) {
            // Remove oldest particle to make room
            particles.shift();
        }
        
        let vx, vy;
        
        switch (this.type) {
            case 'point':
                // Single direction upward (much slower)
                vx = (Math.random() - 0.5) * this.speed * 0.1;
                vy = -Math.random() * this.speed * 0.5;
                break;
                
            case 'star':
                // 8 directions like a star (slower)
                const angle = (Math.floor(Math.random() * 8) * 45) * Math.PI / 180;
                vx = Math.cos(angle) * this.speed * 0.3;
                vy = Math.sin(angle) * this.speed * 0.3;
                break;
                
            case 'circle':
                // 360 degree circle (slower)
                const circleAngle = Math.random() * 2 * Math.PI;
                vx = Math.cos(circleAngle) * this.speed * 0.3;
                vy = Math.sin(circleAngle) * this.speed * 0.3;
                break;
                
            case 'rect':
                // 4 directions (cardinal directions, slower)
                const directions = [
                    { x: 0, y: -1 },  // Up
                    { x: 1, y: 0 },   // Right
                    { x: 0, y: 1 },   // Down
                    { x: -1, y: 0 }   // Left
                ];
                const direction = directions[Math.floor(Math.random() * 4)];
                vx = direction.x * this.speed * 0.3;
                vy = direction.y * this.speed * 0.3;
                break;
                
            default:
                vx = 0;
                vy = 0;
        }
        
        const particle = new Particle(
            this.x + (Math.random() - 0.5) * 4, // Small random offset
            this.y + (Math.random() - 0.5) * 4,
            vx,
            vy,
            this.lifetime * 1000, // Convert to milliseconds
            this.type
        );
        
        particles.push(particle);
    }
}

// Find emitter by ID
function findEmitterById(id) {
    return particleEmitters.find(emitter => emitter.id === id);
}

// Update particle system
function updateParticles() {
    const currentTime = performance.now();
    const deltaTime = currentTime - lastFrameTime;
    lastFrameTime = currentTime;
    
    // Update emitters
    particleEmitters.forEach(emitter => {
        emitter.update(deltaTime);
    });
    
    // Update and remove dead particles
    particles = particles.filter(particle => {
        return particle.update(deltaTime);
    });
    
    // Manage dirty flag based on particle activity
    // This ensures continuous animation when particles are active
    const hasActiveContent = particles.length > 0 || particleEmitters.some(e => e.visible);
    
    if (hasActiveContent) {
        window.RetroGraphics._particlesDirty = true;
    } else {
        // Reset dirty flag when no particles are active
        window.RetroGraphics._particlesDirty = false;
    }
}

// Render all particles
function renderParticles(ctx, canvasWidth, canvasHeight) {
    if (!ctx) {
        return;
    }
    
    // Update particle system
    updateParticles();
    
    // Render particles
    particles.forEach(particle => {
        const alpha = particle.getAlpha();
        if (alpha <= 0) return;
        
        // Use bright green with fading
        const brightness = Math.floor(alpha * 255);
        ctx.fillStyle = `rgb(0, ${brightness}, 0)`;
        
        // Draw particle as larger, more visible rectangle
        const size = Math.max(1, Math.floor(alpha * 4)); // Size varies with alpha
        ctx.fillRect(
            Math.floor(particle.x), 
            Math.floor(particle.y), 
            size, 
            size
        );
    });
}

// Handle CREATE_EMITTER command
function handleCreateEmitter(data) {
    const id = data.id;
    const customData = data.customData;
    
    if (!customData) {
        console.error('[PARTICLE-MANAGER] CREATE_EMITTER missing custom data');
        return false;
    }
    
    // Remove existing emitter with same ID
    particleEmitters = particleEmitters.filter(emitter => emitter.id !== id);
    
    // Create new emitter
    const emitter = new ParticleEmitter(
        id,
        customData.type || 'point',
        customData.pps || 50,
        customData.speed || 100,
        customData.lifetime || 2.0
    );
    
    particleEmitters.push(emitter);
    
    return true;
}

// Handle MOVE_EMITTER command
function handleMoveEmitter(data) {
    const customData = data.customData;
    
    if (!customData || typeof customData.id === 'undefined') {
        console.warn('[PARTICLE-MANAGER] MOVE_EMITTER: Missing emitter ID');
        return false;
    }
    
    const emitter = findEmitterById(customData.id);
    if (!emitter) {
        console.warn('[PARTICLE-MANAGER] MOVE_EMITTER: Emitter not found for ID:', customData.id);
        return false;
    }
    
    emitter.setPosition(customData.x || 0, customData.y || 0);
    return true;
}

// Handle SHOW_EMITTER command
function handleShowEmitter(data) {
    const customData = data.customData;
    
    if (!customData || typeof customData.id === 'undefined') {
        console.warn('[PARTICLE-MANAGER] SHOW_EMITTER: Missing emitter ID');
        return false;
    }
    
    const emitter = findEmitterById(customData.id);
    if (!emitter) {
        console.warn('[PARTICLE-MANAGER] SHOW_EMITTER: Emitter not found for ID:', customData.id);
        return false;
    }
    
    emitter.show();
    return true;
}

// Handle HIDE_EMITTER command
function handleHideEmitter(data) {
    const customData = data.customData;
    
    if (!customData || typeof customData.id === 'undefined') {
        console.warn('[PARTICLE-MANAGER] HIDE_EMITTER: Missing emitter ID');
        return false;
    }
    
    const emitter = findEmitterById(customData.id);
    if (!emitter) {
        console.warn('[PARTICLE-MANAGER] HIDE_EMITTER: Emitter not found for ID:', customData.id);
        return false;
    }
    
    emitter.hide();
    return true;
}

// Handle SET_GRAVITY command
function handleSetGravity(data) {
    const customData = data.customData;
    
    if (!customData || typeof customData.gravity === 'undefined') {
        console.warn('[PARTICLE-MANAGER] SET_GRAVITY: Missing gravity value');
        return false;
    }
    
    globalGravity = customData.gravity;
    window.RetroGraphics._particlesDirty = true;
    return true;
}

// Clear all emitters and particles
function clearAllParticles() {
    // Nur Partikel löschen, Emitter-Definitionen behalten
    particles = [];
    
    // Alle Emitter auf "hidden" setzen aber nicht löschen
    particleEmitters.forEach(emitter => {
        if (emitter) {
            emitter.visible = false;
            emitter.particles = [];
        }
    });
    
    globalGravity = 0;
    window.RetroGraphics._particlesDirty = true;
}

// Get debug info
function getParticleDebugInfo() {
    return {
        emitters: particleEmitters.length,
        particles: particles.length,
        gravity: globalGravity,
        visibleEmitters: particleEmitters.filter(e => e.visible).length
    };
}

// Export functions for global access
window.particleManager = {
    initParticleManager,
    renderParticles,
    handleCreateEmitter,
    handleMoveEmitter,
    handleShowEmitter,
    handleHideEmitter,
    handleSetGravity,
    clearAllParticles,
    getParticleDebugInfo,
    getParticleEmitters: () => particleEmitters,
    getParticles: () => particles
};

// Auto-initialize when loaded
if (typeof window !== 'undefined') {
    // Immediately initialize particleManager
    initParticleManager();
    
    // Also dispatch an event to notify other components
    if (typeof document !== 'undefined') {
        const event = new CustomEvent('particlemanagerready');
        document.dispatchEvent(event);
    }
}