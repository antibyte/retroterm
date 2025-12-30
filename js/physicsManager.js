// physicsManager.js - Physics system using Planck.js for TinyBASIC
// Integrates with existing SPRITE and 2D graphics systems

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
        this.timeStep = 1/60; // Higher frequency for better stability
        this.velocityIterations = 8; // Moderate solver iterations - too high causes instability
        this.positionIterations = 3;  // Moderate position correction - too high causes oscillation
        
        // Visual rendering via existing RetroGraphics system
        this.visualBodies = new Map(); // id -> visual properties
        this.linkedObjects = new Map(); // physics_id -> object_id mapping
        
        // Simple timing for sprite updates
        this.lastSpriteUpdateTime = 0;
        
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
            
            // Configure physics world for better stability
            this.world.setAllowSleeping(true); // Enable sleeping to stop jittering objects
            this.world.setWarmStarting(true);  // Improve solver stability
            this.world.setContinuousPhysics(true); // Prevent tunneling
            
            // Set aggressive sleep parameters for energy loss
            if (this.world.setSleepTimeThreshold) {
                this.world.setSleepTimeThreshold(0.5); // Objects sleep after 0.5 seconds of low movement
            }
            if (this.world.setSleepLinearTolerance) {
                this.world.setSleepLinearTolerance(0.1); // Allow small movement before sleeping
            }
            if (this.world.setSleepAngularTolerance) {
                this.world.setSleepAngularTolerance(0.1); // Allow small rotation before sleeping
            }
            
            this.setupCollisionCallbacks();
            console.log('[PHYSICS-MANAGER] Physics world initialized with stability settings');
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

    // Create rectangle collider (static or dynamic body with ID)
    createDynamicRect(x, y, width, height, id = 1, type = "static") {
        if (!this.world) return;

        try {
            const body = this.world.createBody({
                type: type, // 'static' or 'dynamic'
                position: planck.Vec2(this.pixelsToMeters(x + width/2), this.pixelsToMeters(y + height/2)),
                linearDamping: (type === 'dynamic') ? 0.3 : 0,  // Moderate damping for stable physics
                angularDamping: (type === 'dynamic') ? 0.8 : 0,  // High but realistic angular damping
                allowSleep: true,  // Allow body to sleep when at rest
                fixedRotation: false // Allow rotation but heavily dampened
            });

            const density = (type === 'dynamic') ? 2.0 : 0; // Realistic density for stable physics
            body.createFixture(planck.Box(this.pixelsToMeters(width/2), this.pixelsToMeters(height/2)), {
                friction: 0.6,   // Moderate friction for realistic behavior
                restitution: 0.0, // No bouncing at all
                density: density
            });

            // Default: keep center of mass in center for stability
            // Custom pivot points can be set later with PHYSICS PIVOT command

            if (type === 'dynamic') {
                // Register as dynamic body with ID for tracking
                this.bodies.set(id, body);
                body.setUserData({ id: id });
                console.log(`[PHYSICS-MANAGER] Dynamic Rect created: (${x},${y}) ${width}x${height} with ID ${id}`);
            } else {
                // Static body without ID tracking
                this.staticBodies.push(body);
                console.log(`[PHYSICS-MANAGER] Static Rect created: (${x},${y}) ${width}x${height}`);
            }
        } catch (error) {
            console.error('[PHYSICS-MANAGER] Error creating dynamic rect:', error);
        }
    }

    // Create circle collider (dynamic body)
    createCircle(x, y, radius, id = 1) {
        if (!this.world) return;

        try {
            const body = this.world.createBody({
                type: 'dynamic', // Make it dynamic so it falls
                position: planck.Vec2(this.pixelsToMeters(x), this.pixelsToMeters(y)),
                linearDamping: 0.2,  // Light damping for natural movement
                angularDamping: 0.6, // Moderate angular damping
                allowSleep: true     // Allow this body to sleep when at rest
            });

            body.createFixture(planck.Circle(this.pixelsToMeters(radius)), {
                friction: 0.4,   // Moderate friction for realistic rolling
                restitution: 0.0, // No bouncing at all
                density: 1.5 // Realistic density for stable physics
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

    // Set body velocity - scaled down for sluggish response
    setVelocity(id, vx, vy) {
        const body = this.bodies.get(id);
        if (body) {
            // Normal velocity scaling for realistic physics
            body.setLinearVelocity(planck.Vec2(vx / this.scale, vy / this.scale));
        }
    }

    // Apply force to body - heavily scaled down for sluggish response
    applyForce(id, fx, fy) {
        const body = this.bodies.get(id);
        if (body) {
            // Normal force scaling for realistic physics
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


    // Update sprite through the proper SPRITE UPDATE system (like BASIC does)
    updateSpriteViaSystem(spriteId, definitionId, x, y, rotation, visible) {
        // Throttle updates to avoid overwhelming the system
        if (!this.lastSpriteUpdateTime || (performance.now() - this.lastSpriteUpdateTime) > 16) { // ~60fps
            // Use the same system that BASIC uses for SPRITE UPDATE
            if (window.spriteManager && typeof window.spriteManager.handleUpdateSprite === 'function') {
                const updateData = {
                    command: 'UPDATE_SPRITE',
                    id: spriteId,
                    definitionId: definitionId,
                    x: x,
                    y: y,
                    rotation: rotation,
                    visible: visible
                };
                
                // Call the sprite manager's update function directly (same as backend does)
                window.spriteManager.handleUpdateSprite(updateData);
                this.lastSpriteUpdateTime = performance.now();
            }
        }
    }

    // Set custom pivot point (center of mass) for a physics body
    setPivotPoint(id, offsetX, offsetY) {
        const body = this.bodies.get(id);
        if (body) {
            const mass = body.getMass();
            const inertia = body.getInertia();
            
            // Convert pixel offset to physics coordinates
            const centerOffset = planck.Vec2(
                this.pixelsToMeters(offsetX), 
                this.pixelsToMeters(offsetY)
            );
            
            // Set new mass data with custom center of mass and adjusted inertia
            body.setMassData({
                mass: mass,
                center: centerOffset,
                I: inertia * 0.5 // Reduce rotational inertia to prevent oscillation with offset pivot
            });
            
            console.log(`[PHYSICS-MANAGER] Set pivot point for body ${id}: (${offsetX}, ${offsetY})`);
        }
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
            if (body.getType() === 'dynamic' && body.isAwake()) { // Only update awake bodies
                // Check velocity and set to zero if very small (helps with jittering)
                const velocity = body.getLinearVelocity();
                const angularVelocity = body.getAngularVelocity();
                const velocityThreshold = 0.1; // Natural threshold - let objects settle naturally
                
                // Only stop completely still objects
                if (Math.abs(velocity.x) < 0.01 && Math.abs(velocity.y) < 0.01 && Math.abs(angularVelocity) < 0.01) {
                    body.setLinearVelocity(planck.Vec2(0, 0));
                    body.setAngularVelocity(0);
                    body.setAwake(false); // Let physics engine handle energy loss naturally
                }
                
                
                const position = body.getPosition();
                const angle = body.getAngle();
                
                // Convert physics coordinates to pixel coordinates for 2D graphics
                const pixelX = this.metersToPixels(position.x);
                const pixelY = this.metersToPixels(position.y);
                const degrees = angle * 180 / Math.PI;

                // Check if this physics body is linked to a SPRITE/2D object
                const objectId = this.linkedObjects.get(physicsId);
                if (objectId) {
                    let updated = false;
                    
                    // PRIORITY 1: Update 2D graphics objects (CIRCLE, RECT) - this must work for physics.bas!
                    if (window.RetroGraphics && window.RetroGraphics.updatePhysicsObject) {
                        window.RetroGraphics.updatePhysicsObject(objectId, Math.round(pixelX), Math.round(pixelY), degrees);
                        updated = true;
                    }
                    
                    // PRIORITY 2: Also try updating SPRITE objects if they exist
                    if (!updated && window.spriteManager && window.spriteManager.spriteInstances) {
                        try {
                            const sprite = window.spriteManager.spriteInstances.get(objectId);
                            if (sprite) {
                                const newX = Math.round(pixelX);
                                const newY = Math.round(pixelY);
                                const newRotation = Math.round(degrees);
                                
                                // Only update if position changed significantly
                                const positionChanged = Math.abs(sprite.x - newX) > 2 || Math.abs(sprite.y - newY) > 2 || Math.abs(sprite.rotation - newRotation) > 5;
                                
                                if (positionChanged) {
                                    // USE THE PROPER SPRITE UPDATE SYSTEM instead of direct manipulation
                                    this.updateSpriteViaSystem(objectId, sprite.definitionId, newX, newY, newRotation, sprite.visible);
                                }
                                
                                updated = true;
                            }
                        } catch (error) {
                            console.error('[PHYSICS-MANAGER] Error updating SPRITE position:', error);
                        }
                    }
                }
            }
        }
    }

    // Enable/disable automatic physics updates using traditional setInterval for better predictability
    setAutoUpdate(enabled) {
        this.autoUpdate = enabled;
        
        if (enabled && !this.updateInterval) {
            // Use setInterval for consistent physics timing, separate from render timing
            this.updateInterval = setInterval(() => {
                this.step();
                // Don't call render() here - let the main render system handle it
            }, this.timeStep * 1000);
            console.log('[PHYSICS-MANAGER] Auto-update enabled with setInterval');
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
                    this.createDynamicRect(params.x, params.y, params.width, params.height, params.id || 1, params.type || "static");
                    // Automatically register the 2D graphics object for physics updates if dynamic
                    if ((params.type === "dynamic") && window.RetroGraphics && window.RetroGraphics.registerPhysicsObject) {
                        const rectData = {
                            x: params.x,
                            y: params.y,
                            width: params.width,
                            height: params.height,
                            color: 8, // Default color
                            fill: 1   // Default fill
                        };
                        window.RetroGraphics.registerPhysicsObject(params.id || 1, 'RECT', rectData);
                        
                        // Also link the physics ID to the visual ID
                        this.linkedObjects.set(params.id || 1, params.id || 1);
                    }
                    break;

                case 'CIRCLE':
                    this.createCircle(params.x, params.y, params.radius, params.id || 1);
                    // Register PHYSICS CIRCLE as dynamic 2D graphics object for 2D physics
                    if (window.RetroGraphics && window.RetroGraphics.registerPhysicsObject) {
                        const circleData = {
                            x: params.x,
                            y: params.y,
                            radius: params.radius,
                            color: 4, // Default color
                            fill: 1   // Default fill
                        };
                        
                        // Remove any static CIRCLE object at the same position that might exist
                        const staticId = params.x + params.y * 1000; // Same ID generation as in gfx_commands.go
                        if (window.RetroGraphics._all2DObjects && window.RetroGraphics._all2DObjects.has(staticId)) {
                            console.log(`[PHYSICS-MANAGER] Removing static CIRCLE ${staticId} to replace with dynamic physics CIRCLE`);
                            window.RetroGraphics._all2DObjects.delete(staticId);
                        }
                        
                        window.RetroGraphics.registerPhysicsObject(params.id || 1, 'CIRCLE', circleData);
                        
                        // Also link the physics ID to the visual ID (same ID for simplicity)
                        this.linkedObjects.set(params.id || 1, params.id || 1);
                    }
                    break;

                case 'SET_VISUAL':
                    this.setVisualProperties(params.id, params.shape, params.color, params.size);
                    break;

                case 'SPRITE':
                    // Get sprite position from spriteManager
                    if (window.spriteManager && window.spriteManager.spriteInstances) {
                        const sprite = window.spriteManager.spriteInstances.get(params.id);
                        if (sprite) {
                            this.addBody(params.id, params.type, params.shape, sprite.x, sprite.y, 32, 32, params.density || 1.0);
                            
                            // Link physics ID to sprite ID for position updates
                            this.linkedObjects.set(params.id, params.id);
                            
                            console.log(`[PHYSICS-MANAGER] Added physics to sprite ${params.id} at (${sprite.x}, ${sprite.y})`);
                        } else {
                            console.error(`[PHYSICS-MANAGER] Sprite ${params.id} not found in spriteInstances`);
                        }
                    } else {
                        console.error('[PHYSICS-MANAGER] spriteManager.spriteInstances not available');
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

                case 'PIVOT':
                    this.setPivotPoint(params.id, params.offsetX, params.offsetY);
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

    // Render is handled by updateVisualPositions() through 2D graphics and SPRITE systems
    render() {
        // Physics rendering is now handled entirely through 2D graphics primitives
        // and SPRITE system updates in updateVisualPositions()
        // VECTOR system integration has been removed as requested
        
        // No longer needed - render loop integration handles this automatically
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