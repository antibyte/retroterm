// physicsManager.js - Physics system using Planck.js for TinyBASIC
// Integrates with existing SPRITE and VECTOR systems

class PhysicsManager {
    constructor() {
        this.world = null;
        this.bodies = new Map(); // id -> body mapping
        this.scale = 30; // Pixel to meter ratio (30 pixels = 1 meter)
        this.autoUpdate = false;
        this.collisionCallbacks = new Map(); // collision detection callbacks
        this.groups = new Map(); // collision groups
        this.groupCollisions = new Map(); // group collision settings
        this.staticBodies = []; // Static geometry bodies
        this.timeStep = 1/60;
        this.velocityIterations = 8;
        this.positionIterations = 3;
        
        // Visual rendering via existing RetroGraphics system
        this.visualBodies = new Map(); // id -> visual properties
        this.linkedObjects = new Map(); // physics_id -> vector_id mapping
        
        console.log('[PHYSICS-MANAGER] Physics Manager created');
    }

    // Use existing graphics system for rendering
    useRetroGraphics() {
        // Check if RetroGraphics is available
        if (typeof window.RetroGraphics !== 'undefined') {
            console.log('[PHYSICS-MANAGER] Using existing RetroGraphics system');
            return true;
        } else {
            console.warn('[PHYSICS-MANAGER] RetroGraphics system not available');
            return false;
        }
    }

    // Initialize physics world
    init() {
        if (typeof planck === 'undefined') {
            console.error('[PHYSICS-MANAGER] Planck.js library not available');
            return false;
        }

        try {
            this.world = planck.World(planck.Vec2(0, 0)); // Start with no gravity
            this.setupCollisionCallbacks();
            console.log('[PHYSICS-MANAGER] Physics world initialized');
            return true;
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Failed to initialize physics world:', error);
            return false;
        }
    }

    // Set world gravity
    setGravity(x, y) {
        if (!this.world) return;
        this.world.setGravity(planck.Vec2(x, y));
        console.log(`[PHYSICS-MANAGER] Gravity set to (${x}, ${y})`);
    }

    // Set pixel to meter scale
    setScale(scale) {
        this.scale = scale;
        console.log(`[PHYSICS-MANAGER] Scale set to ${scale} pixels per meter`);
    }

    // Convert pixel coordinates to physics coordinates
    pixelsToMeters(pixels) {
        return pixels / this.scale;
    }

    // Convert physics coordinates to pixel coordinates
    metersToPixels(meters) {
        return meters * this.scale;
    }

    // Create floor line (static body)
    createFloor(x1, y1, x2, y2) {
        if (!this.world) return;

        try {
            const centerX = (x1 + x2) / 2;
            const centerY = (y1 + y2) / 2;
            const length = Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2);
            const angle = Math.atan2(y2 - y1, x2 - x1);

            const body = this.world.createBody({
                type: 'static',
                position: planck.Vec2(this.pixelsToMeters(centerX), this.pixelsToMeters(centerY)),
                angle: angle
            });

            // Create very thin box as line
            body.createFixture(planck.Box(this.pixelsToMeters(length / 2), this.pixelsToMeters(1)), {
                friction: 0.5,
                restitution: 0.1,
                density: 0
            });

            this.staticBodies.push(body);
            console.log(`[PHYSICS-MANAGER] Floor created: (${x1},${y1}) to (${x2},${y2})`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating floor:', error);
        }
    }

    // Create wall line (static body)
    createWall(x1, y1, x2, y2) {
        if (!this.world) return;

        try {
            const centerX = (x1 + x2) / 2;
            const centerY = (y1 + y2) / 2;
            const length = Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2);
            const angle = Math.atan2(y2 - y1, x2 - x1);

            const body = this.world.createBody({
                type: 'static',
                position: planck.Vec2(this.pixelsToMeters(centerX), this.pixelsToMeters(centerY)),
                angle: angle
            });

            // Create very thin box as line
            body.createFixture(planck.Box(this.pixelsToMeters(length / 2), this.pixelsToMeters(1)), {
                friction: 0.3,
                restitution: 0.8,
                density: 0
            });

            this.staticBodies.push(body);
            console.log(`[PHYSICS-MANAGER] Wall created: (${x1},${y1}) to (${x2},${y2})`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating wall:', error);
        }
    }

    // Create generic line (static body)
    createLine(x1, y1, x2, y2) {
        if (!this.world) return;

        try {
            const centerX = (x1 + x2) / 2;
            const centerY = (y1 + y2) / 2;
            const length = Math.sqrt((x2 - x1) ** 2 + (y2 - y1) ** 2);
            const angle = Math.atan2(y2 - y1, x2 - x1);

            const body = this.world.createBody({
                type: 'static',
                position: planck.Vec2(this.pixelsToMeters(centerX), this.pixelsToMeters(centerY)),
                angle: angle
            });

            // Create very thin box as line
            body.createFixture(planck.Box(this.pixelsToMeters(length / 2), this.pixelsToMeters(1)), {
                friction: 0.4,
                restitution: 0.5,
                density: 0
            });

            this.staticBodies.push(body);
            console.log(`[PHYSICS-MANAGER] Line created: (${x1},${y1}) to (${x2},${y2})`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating line:', error);
        }
    }

    // Create rectangle collider (static body)
    createRect(x, y, width, height) {
        if (!this.world) return;

        try {
            const body = this.world.createBody({
                type: 'static',
                position: planck.Vec2(this.pixelsToMeters(x + width/2), this.pixelsToMeters(y + height/2))
            });

            body.createFixture(planck.Box(this.pixelsToMeters(width/2), this.pixelsToMeters(height/2)), {
                friction: 0.4,
                restitution: 0.3,
                density: 0
            });

            this.staticBodies.push(body);
            console.log(`[PHYSICS-MANAGER] Rect created: (${x},${y}) ${width}x${height}`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating rect:', error);
        }
    }

    // Create circle collider (dynamic body)
    createCircle(x, y, radius, id = 1) {
        if (!this.world) return;

        try {
            const body = this.world.createBody({
                type: 'dynamic', // Make it dynamic so it falls
                position: planck.Vec2(this.pixelsToMeters(x), this.pixelsToMeters(y))
            });

            body.createFixture(planck.Circle(this.pixelsToMeters(radius)), {
                friction: 0.4,
                restitution: 0.8,
                density: 1.0 // Give it mass so gravity affects it
            });

            // Register with specified ID
            this.bodies.set(id, body);
            body.setUserData({ id: id });
            
            console.log(`[PHYSICS-MANAGER] Dynamic circle created: (${x},${y}) radius ${radius} with ID ${id}`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating circle:', error);
        }
    }

    // Add physics body for sprite/vector
    addBody(id, type, shape, x, y, width = 32, height = 32, density = 1.0) {
        if (!this.world) return;

        try {
            const bodyDef = {
                type: type, // 'static', 'dynamic', 'kinematic'
                position: planck.Vec2(this.pixelsToMeters(x), this.pixelsToMeters(y))
            };

            const body = this.world.createBody(bodyDef);

            // Create fixture based on shape
            let fixture;
            if (shape === 'circle' || shape === 'sphere') {
                const radius = Math.min(width, height) / 2;
                fixture = body.createFixture(planck.Circle(this.pixelsToMeters(radius)), {
                    density: density,
                    friction: 0.3,
                    restitution: 0.3
                });
            } else {
                // Default to box for cube, pyramid, etc.
                fixture = body.createFixture(planck.Box(this.pixelsToMeters(width/2), this.pixelsToMeters(height/2)), {
                    density: density,
                    friction: 0.3,
                    restitution: 0.3
                });
            }

            // Store body with ID
            this.bodies.set(id, body);
            
            // Store ID in body userData for collision detection
            body.setUserData({ id: id });

            console.log(`[PHYSICS-MANAGER] Body added: ID=${id}, type=${type}, shape=${shape}`);
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error adding body:', error);
        }
    }

    // Remove physics body
    removeBody(id) {
        if (!this.world) return;

        const body = this.bodies.get(id);
        if (body) {
            this.world.destroyBody(body);
            this.bodies.delete(id);
            console.log(`[PHYSICS-MANAGER] Body removed: ID=${id}`);
        }
    }

    // Set body velocity
    setVelocity(id, vx, vy) {
        const body = this.bodies.get(id);
        if (body) {
            body.setLinearVelocity(planck.Vec2(vx / this.scale, vy / this.scale));
        }
    }

    // Apply force to body
    applyForce(id, fx, fy) {
        const body = this.bodies.get(id);
        if (body) {
            const force = planck.Vec2(fx / this.scale, fy / this.scale);
            body.applyForceToCenter(force);
        }
    }

    // Set body properties
    setFriction(id, friction) {
        const body = this.bodies.get(id);
        if (body) {
            for (let fixture = body.getFixtureList(); fixture; fixture = fixture.getNext()) {
                fixture.setFriction(friction);
            }
        }
    }

    setBounce(id, restitution) {
        const body = this.bodies.get(id);
        if (body) {
            for (let fixture = body.getFixtureList(); fixture; fixture = fixture.getNext()) {
                fixture.setRestitution(restitution);
            }
        }
    }

    setDensity(id, density) {
        const body = this.bodies.get(id);
        if (body) {
            for (let fixture = body.getFixtureList(); fixture; fixture = fixture.getNext()) {
                fixture.setDensity(density);
            }
            body.resetMassData();
        }
    }

    // Collision groups
    setGroup(id, groupName) {
        this.groups.set(id, groupName);
    }

    setGroupCollision(group1, group2, enabled) {
        const key = `${group1}_${group2}`;
        const reverseKey = `${group2}_${group1}`;
        this.groupCollisions.set(key, enabled);
        this.groupCollisions.set(reverseKey, enabled);
    }

    // Setup collision detection
    setupCollisionCallbacks() {
        if (!this.world) return;

        this.world.on('begin-contact', (contact) => {
            const bodyA = contact.getFixtureA().getBody();
            const bodyB = contact.getFixtureB().getBody();
            
            const dataA = bodyA.getUserData();
            const dataB = bodyB.getUserData();
            
            if (dataA && dataB) {
                this.handleCollision(dataA.id, dataB.id);
            }
        });
    }

    // Handle collision between two objects
    handleCollision(id1, id2) {
        // Check collision callbacks
        const key1 = `${id1}_${id2}`;
        const key2 = `${id2}_${id1}`;
        
        if (this.collisionCallbacks.has(key1)) {
            const lineNumber = this.collisionCallbacks.get(key1);
            console.log(`[PHYSICS-MANAGER] Collision: ${id1} vs ${id2}, jumping to line ${lineNumber}`);
            // TODO: Trigger BASIC GOSUB to line number
        } else if (this.collisionCallbacks.has(key2)) {
            const lineNumber = this.collisionCallbacks.get(key2);
            console.log(`[PHYSICS-MANAGER] Collision: ${id2} vs ${id1}, jumping to line ${lineNumber}`);
            // TODO: Trigger BASIC GOSUB to line number
        }
    }

    // Set collision callback
    setCollisionCallback(id1, id2, lineNumber) {
        const key = `${id1}_${id2}`;
        this.collisionCallbacks.set(key, lineNumber);
    }

    // Link physics body to VECTOR/SPRITE object
    linkPhysicsToVector(physicsId, vectorId) {
        this.linkedObjects.set(physicsId, vectorId);
        console.log(`[PHYSICS-MANAGER] Linked physics body ${physicsId} to VECTOR/SPRITE ${vectorId}`);
    }

    // Physics step
    step() {
        if (!this.world) return;

        try {
            this.world.step(this.timeStep, this.velocityIterations, this.positionIterations);
            this.updateVisualPositions();
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error during physics step:', error);
        }
    }

    // Update visual positions of sprites/graphics based on physics
    updateVisualPositions() {
        for (const [physicsId, body] of this.bodies) {
            if (body.getType() === 'dynamic') {
                const position = body.getPosition();
                const angle = body.getAngle();
                
                // Convert physics coordinates to pixel coordinates for 2D graphics
                const pixelX = this.metersToPixels(position.x);
                const pixelY = this.metersToPixels(position.y);
                const degrees = angle * 180 / Math.PI;

                // Check if this physics body is linked to a VECTOR/SPRITE/2D object
                const objectId = this.linkedObjects.get(physicsId);
                if (objectId) {
                    // Try updating VECTOR objects (keep existing functionality)
                    if (window.vectorManager && window.vectorManager.handleUpdateVector3D) {
                        try {
                            const vectorX = position.x; // Direct coordinates for vectors
                            const vectorY = position.y;
                            const rotX = angle;
                            const rotY = angle * 0.7;
                            const rotZ = angle * 0.5;
                            
                            window.vectorManager.handleUpdateVector3D({
                                id: objectId,
                                shape: 'sphere',
                                position: { x: vectorX, y: vectorY, z: -8 },
                                vecRotation: { x: rotX, y: rotY, z: rotZ },
                                scale: 3.0,
                                brightness: 15,
                                visible: true
                            });
                            console.log(`[PHYSICS-MANAGER] Updated VECTOR ${objectId} from physics ${physicsId}: (${vectorX.toFixed(1)}, ${vectorY.toFixed(1)})`);
                        } catch (error) {
                            console.error('[PHYSICS-MANAGER] Error updating VECTOR position:', error);
                        }
                    }
                    
                    // Try updating SPRITE objects
                    if (window.spriteManager && window.spriteManager.updateSpritePosition) {
                        try {
                            window.spriteManager.updateSpritePosition(objectId, Math.round(pixelX), Math.round(pixelY), degrees);
                            console.log(`[PHYSICS-MANAGER] Updated SPRITE ${objectId} from physics ${physicsId}: (${Math.round(pixelX)}, ${Math.round(pixelY)})`);
                        } catch (error) {
                            console.error('[PHYSICS-MANAGER] Error updating SPRITE position:', error);
                        }
                    }
                    
                    // Try updating 2D graphics objects (CIRCLE, RECT)
                    // For 2D graphics, we need to store and redraw, not update existing
                    if (window.RetroGraphics && window.RetroGraphics.updatePhysicsObject) {
                        try {
                            window.RetroGraphics.updatePhysicsObject(objectId, Math.round(pixelX), Math.round(pixelY), degrees);
                            console.log(`[PHYSICS-MANAGER] Updated 2D graphics object ${objectId} from physics ${physicsId}: (${Math.round(pixelX)}, ${Math.round(pixelY)})`);
                        } catch (error) {
                            console.error('[PHYSICS-MANAGER] Error updating 2D graphics object:', error);
                        }
                    }
                } else if (physicsId === 1) {
                    // Fallback: legacy behavior for physics ID 1
                    if (window.vectorManager && window.vectorManager.handleUpdateVector3D) {
                        try {
                            window.vectorManager.handleUpdateVector3D({
                                id: 1,
                                shape: 'sphere',
                                position: { x: pixelX, y: pixelY, z: -5 },
                                vecRotation: { x: 0, y: 0, z: 0 },
                                scale: 1.0,
                                brightness: 4,
                                visible: true
                            });
                            console.log(`[PHYSICS-MANAGER] Updated VECTOR sphere ID 1 (legacy): (${pixelX.toFixed(1)}, ${pixelY.toFixed(1)})`);
                        } catch (error) {
                            console.error('[PHYSICS-MANAGER] Error updating VECTOR position (legacy):', error);
                        }
                    }
                }
            }
        }
    }

    // Enable/disable automatic physics updates
    setAutoUpdate(enabled) {
        this.autoUpdate = enabled;
        
        if (enabled && !this.updateInterval) {
            this.updateInterval = setInterval(() => {
                this.step();
                this.render();
            }, this.timeStep * 1000);
            console.log('[PHYSICS-MANAGER] Auto-update enabled');
        } else if (!enabled && this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
            console.log('[PHYSICS-MANAGER] Auto-update disabled');
        }
    }

    // Handle physics command from backend
    handleCommand(data) {
        const { command, params } = data;

        try {
            switch (command) {
                case 'WORLD':
                    this.setGravity(params.gravityX, params.gravityY);
                    break;

                case 'SCALE':
                    this.setScale(params.scale);
                    break;

                case 'FLOOR':
                    this.createFloor(params.x1, params.y1, params.x2, params.y2);
                    break;

                case 'WALL':
                    this.createWall(params.x1, params.y1, params.x2, params.y2);
                    break;

                case 'LINE':
                    this.createLine(params.x1, params.y1, params.x2, params.y2);
                    break;

                case 'RECT':
                    this.createRect(params.x, params.y, params.width, params.height);
                    break;

                case 'CIRCLE':
                    this.createCircle(params.x, params.y, params.radius, params.id || 1);
                    // Automatically register the 2D graphics object for physics updates
                    if (window.RetroGraphics && window.RetroGraphics.registerPhysicsObject) {
                        const circleData = {
                            x: params.x,
                            y: params.y,
                            radius: params.radius,
                            color: 4, // Default color
                            fill: 1   // Default fill
                        };
                        window.RetroGraphics.registerPhysicsObject(params.id || 1, 'CIRCLE', circleData);
                        
                        // Also link the physics ID to the visual ID (same ID for simplicity)
                        this.linkPhysicsToVector(params.id || 1, params.id || 1);
                    }
                    break;

                case 'SET_VISUAL':
                    this.setVisualProperties(params.id, params.shape, params.color, params.size);
                    break;

                case 'SPRITE':
                    // Get sprite position from sprite manager
                    if (window.spriteManager && window.spriteManager.getSpritePosition) {
                        const pos = window.spriteManager.getSpritePosition(params.id);
                        if (pos) {
                            this.addBody(params.id, params.type, params.shape, pos.x, pos.y, 32, 32, params.density);
                        }
                    }
                    break;

                case 'VECTOR':
                    // Get vector position from vector manager
                    if (window.vectorManager && window.vectorManager.getVectorPosition) {
                        const pos = window.vectorManager.getVectorPosition(params.id);
                        if (pos) {
                            const size = pos.scale * 32; // Estimate size from scale
                            this.addBody(params.id, params.type, params.shape, pos.x, pos.y, size, size, params.density);
                        }
                    }
                    break;

                case 'STEP':
                    this.step();
                    break;

                case 'AUTO':
                    this.setAutoUpdate(params.enabled);
                    break;

                case 'VELOCITY':
                    this.setVelocity(params.id, params.vx, params.vy);
                    break;

                case 'FORCE':
                    this.applyForce(params.id, params.fx, params.fy);
                    break;

                case 'FRICTION':
                    this.setFriction(params.id, params.friction);
                    break;

                case 'BOUNCE':
                    this.setBounce(params.id, params.bounce);
                    break;

                case 'DENSITY':
                    this.setDensity(params.id, params.density);
                    break;

                case 'GROUP':
                    this.setGroup(params.id, params.group);
                    break;

                case 'COLLIDE':
                    this.setGroupCollision(params.group1, params.group2, params.enabled);
                    break;

                case 'COLLISION':
                    this.setCollisionCallback(params.id1, params.id2, params.lineNumber);
                    break;

                case 'LINK':
                    this.linkPhysicsToVector(params.physics_id, params.vector_id);
                    break;

                default:
                    console.warn('[PHYSICS-MANAGER] Unknown command:', command);
            }
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error handling command:', command, error);
        }
    }

    // Cleanup
    destroy() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
        }

        if (this.world) {
            // Destroy all bodies
            for (const body of this.staticBodies) {
                if (body) this.world.destroyBody(body);
            }
            for (const [id, body] of this.bodies) {
                if (body) this.world.destroyBody(body);
            }

            this.staticBodies = [];
            this.bodies.clear();
            this.world = null;
        }

        console.log('[PHYSICS-MANAGER] Physics Manager destroyed');
    }

    // Render physics bodies using existing RetroGraphics/Vector system
    render() {
        if (!this.world) return;
        
        // Only render if RetroGraphics is available
        if (!this.useRetroGraphics()) return;

        // Update positions of all dynamic bodies using VECTOR system
        for (let [id, body] of this.bodies) {
            const position = body.getPosition();
            const angle = body.getAngle();
            const pixelX = this.metersToPixels(position.x);
            const pixelY = this.metersToPixels(position.y);
            const degrees = angle * 180 / Math.PI;

            // Get visual properties
            const visual = this.visualBodies.get(id) || { shape: 'circle', color: 4, size: 25 };
            
            // Use existing VECTOR system to render physics objects
            if (window.handleUpdateVector3D) {
                try {
                    if (visual.shape === 'circle') {
                        // Update circle position via VECTOR system
                        window.handleUpdateVector3D({
                            id: 1000 + id,
                            shape: 'sphere',
                            position: { x: pixelX, y: pixelY, z: -5 },
                            vecRotation: { x: 0, y: degrees * Math.PI / 180, z: 0 },
                            scale: visual.size / 30,
                            brightness: visual.color,
                            visible: true
                        });
                    } else {
                        // Update box position via VECTOR system  
                        window.handleUpdateVector3D({
                            id: 1000 + id,
                            shape: 'cube',
                            position: { x: pixelX, y: pixelY, z: -5 },
                            vecRotation: { x: 0, y: degrees * Math.PI / 180, z: 0 },
                            scale: visual.size / 30,
                            brightness: visual.color,
                            visible: true
                        });
                    }
                } catch (error) {
                    console.error('[PHYSICS-MANAGER] Error updating VECTOR position:', error);
                }
            }
        }
    }

    // Store visual properties for a body
    setVisualProperties(id, shape, color, size) {
        this.visualBodies.set(id, { shape, color, size });
    }
}

// Create global physics manager instance
if (typeof window !== 'undefined') {
    window.physicsManager = new PhysicsManager();
    
    // Auto-initialize if Planck.js is already loaded
    if (typeof planck !== 'undefined') {
        window.physicsManager.init();
    } else {
        // Wait for Planck.js to load
        document.addEventListener('DOMContentLoaded', () => {
            if (typeof planck !== 'undefined') {
                window.physicsManager.init();
            }
        });
    }
    
    console.log('[PHYSICS-MANAGER] Physics Manager loaded');
}