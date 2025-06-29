/*
 * BUILD SYSTEM REMINDER:
 * This file is bundled by build.js for production. If you add new JavaScript files 
 * or modify the structure, update the bundleJsFiles array in build.js
 */

const CFG = window.CRT_CONFIG;

// Globale Variable _vectorsDirty im window.RetroGraphics Namespace erstellen, falls nicht vorhanden
if (!window.RetroGraphics) {
    window.RetroGraphics = {};
}
if (typeof window.RetroGraphics._vectorsDirty === 'undefined') {
    window.RetroGraphics._vectorsDirty = false;
}

// Module-level variables
let graphicsCanvas = null;
// let graphicsContext = null; // Context is passed to renderVectors, not stored globally from init
let camera = null;
let vectorObjects = [];
let objectIdCounter = 0;
let testCube = null;
let axes = null;

// Utility functions
function toRadians(degrees) {
    return degrees * (Math.PI / 180);
}

// Default Object Creation
function createTestCube(size = 10, position = { x: 0, y: 0, z: 0 }, color = '#00FF00') {
    const halfSize = size / 2; // Corrected: size should be the full width, so halfSize = size/2
    const vertices = [
        // Front face
        { x: -halfSize, y: -halfSize, z: -halfSize }, // 0
        { x:  halfSize, y: -halfSize, z: -halfSize }, // 1
        { x:  halfSize, y:  halfSize, z: -halfSize }, // 2
        { x: -halfSize, y:  halfSize, z: -halfSize }, // 3
        // Back face
        { x: -halfSize, y: -halfSize, z:  halfSize }, // 4
        { x:  halfSize, y: -halfSize, z:  halfSize }, // 5
        { x:  halfSize, y:  halfSize, z:  halfSize }, // 6
        { x: -halfSize, y:  halfSize, z:  halfSize }  // 7
    ];

    const edges = [
        [0, 1], [1, 2], [2, 3], [3, 0], // Front face
        [4, 5], [5, 6], [6, 7], [7, 4], // Back face
        [0, 4], [1, 5], [2, 6], [3, 7]  // Connecting edges
    ];
    const id = `test_cube_${objectIdCounter++}`;
    const obj = {
        id: id,
        vertices: vertices,
        edges: edges,
        color: color,
        worldMatrix: createIdentityMatrix(), // Initial world matrix
        transform: { // Store individual transformations
            translation: { x: position.x, y: position.y, z: position.z },
            rotation: { x: 0, y: 0, z: 0 }, // Radians
            scale: { x: 1, y: 1, z: 1 }
        },
        visible: true
    };
    updateWorldMatrix(obj); // Apply initial transform
    return obj;
}

// Create sphere with wireframe representation
function createSphere(radius = 5, position = { x: 0, y: 0, z: 0 }, color = '#00FF00', segments = 8) {
    const vertices = [];
    const edges = [];
    
    // Create sphere vertices using spherical coordinates
    // Add poles
    vertices.push({ x: 0, y: radius, z: 0 }); // North pole (index 0)
    vertices.push({ x: 0, y: -radius, z: 0 }); // South pole (index 1)
    
    // Create latitude rings
    const rings = segments;
    const pointsPerRing = segments;
    
    for (let ring = 1; ring < rings; ring++) {
        const phi = Math.PI * ring / rings; // latitude angle
        const y = radius * Math.cos(phi);
        const ringRadius = radius * Math.sin(phi);
        
        for (let point = 0; point < pointsPerRing; point++) {
            const theta = 2 * Math.PI * point / pointsPerRing; // longitude angle
            const x = ringRadius * Math.cos(theta);
            const z = ringRadius * Math.sin(theta);
            
            vertices.push({ x, y, z });
        }
    }
    
    // Create edges
    const vertexCount = vertices.length;
    
    // Connect poles to first and last rings
    for (let i = 0; i < pointsPerRing; i++) {
        edges.push([0, 2 + i]); // North pole to first ring
        edges.push([1, vertexCount - pointsPerRing + i]); // South pole to last ring
    }
    
    // Connect rings horizontally (longitude lines)
    for (let ring = 0; ring < rings - 2; ring++) {
        const ringStart = 2 + ring * pointsPerRing;
        const nextRingStart = 2 + (ring + 1) * pointsPerRing;
        
        for (let point = 0; point < pointsPerRing; point++) {
            const currentVertex = ringStart + point;
            const nextRingVertex = nextRingStart + point;
            edges.push([currentVertex, nextRingVertex]);
        }
    }
    
    // Connect points within rings (latitude lines)
    for (let ring = 0; ring < rings - 1; ring++) {
        const ringStart = 2 + ring * pointsPerRing;
        
        for (let point = 0; point < pointsPerRing; point++) {
            const currentVertex = ringStart + point;
            const nextVertex = ringStart + ((point + 1) % pointsPerRing);
            edges.push([currentVertex, nextVertex]);
        }
    }
    
    const id = `sphere_${objectIdCounter++}`;
    const obj = {
        id: id,
        vertices: vertices,
        edges: edges,
        color: color,
        worldMatrix: createIdentityMatrix(),
        transform: {
            translation: { x: position.x, y: position.y, z: position.z },
            rotation: { x: 0, y: 0, z: 0 },
            scale: { x: 1, y: 1, z: 1 }        },
        visible: true,
        shape: 'sphere'
    };
    updateWorldMatrix(obj);
    return obj;
}

function createAxes(length = 50) {
    const halfLength = length / 2; // Or full length from origin, depending on desired representation
    const vertices = [
        { x: 0, y: 0, z: 0 },          // Origin 0
        { x: length, y: 0, z: 0 },     // X-axis end 1
        { x: 0, y: length, z: 0 },     // Y-axis end 2
        { x: 0, y: 0, z: length }      // Z-axis end 3
    ];
    const edges = [
        [0, 1], // X-axis
        [0, 2], // Y-axis
        [0, 3]  // Z-axis
    ];
    const colors = ['#FF0000', '#00FF00', '#0000FF']; // X=Red, Y=Green, Z=Blue
    const id = `axes_${objectIdCounter++}`;

    // For axes, we might render them as separate objects or a single one with multi-colored lines.
    // For simplicity here, treating as one object, but line coloring needs to be handled in renderVectors.
    // Or, create three separate line objects. Let's try one object with a default color,
    // and rely on a more advanced renderVectors or specific logic if per-edge color is needed.
    // For now, we'll make it a single object and render all lines with its main color.
    // A more complex setup would involve creating 3 distinct objects.

    const obj = {
        id: id,
        vertices: vertices,
        edges: edges, // Edges for the single object
        // To handle individual colors, renderVectors would need to check obj.id or a type
        // and apply colors per edge based on some convention.
        // Or, create 3 objects:
        // createLineObject([{x:0,y:0,z:0}, {x:length,y:0,z:0}], '#FF0000', 'axis_x');
        // createLineObject([{x:0,y:0,z:0}, {x:0,y:length,z:0}], '#00FF00', 'axis_y');
        // createLineObject([{x:0,y:0,z:0}, {x:0,y:0,z:length}], '#0000FF', 'axis_z');
        // For now, let's stick to a single object for simplicity of add/remove.
        color: CFG.AXES_COLOR || '#FFFFFF', // Default white, or specific from config
        worldMatrix: createIdentityMatrix(),
        transform: {
            translation: { x: 0, y: 0, z: 0 },
            rotation: { x: 0, y: 0, z: 0 },
            scale: { x: 1, y: 1, z: 1 }
        },
        visible: true,        // Special property to indicate this object needs per-edge coloring if renderer supports it
        edgeColors: colors // Store intended colors for edges [X, Y, Z]
    };
    updateWorldMatrix(obj);
    return obj;
}


// Core Initialization
function initVectorManager(canvasInstance, contextInstance) {
    
    if (!canvasInstance) {
        console.error("[VECTOR-MANAGER-INIT] Canvas instance is required!");
        return;
    }
    graphicsCanvas = canvasInstance;
    // graphicsContext = contextInstance; // Store if needed globally, or just use passed ctx in renderVectors

    vectorObjects = []; // Reset objects
    objectIdCounter = 0;
    const cameraSettings = {
        x: CFG.VECTOR_CAMERA_X !== undefined ? CFG.VECTOR_CAMERA_X : 0,
        y: CFG.VECTOR_CAMERA_Y !== undefined ? CFG.VECTOR_CAMERA_Y : 0,
        z: CFG.VECTOR_CAMERA_Z !== undefined ? CFG.VECTOR_CAMERA_Z : 50,
        fov: CFG.VECTOR_FOV || 60,
        near: CFG.VECTOR_NEAR_PLANE || 0.1,
        far: CFG.VECTOR_FAR_PLANE || 1000,
        focalLength: CFG.VECTOR_FOCAL_LENGTH // Can be null for auto-calculation
    };

    // Ensure graphicsCanvas is set before using its dimensions
    if (!graphicsCanvas) {
        console.error("[VECTOR-MANAGER-INIT] graphicsCanvas is not set, cannot calculate focal length.");
        return; // Critical error
    }

    let focalLengthValue;
    if (cameraSettings.focalLength === null || cameraSettings.focalLength === undefined) {
        const fovRad = toRadians(cameraSettings.fov);
        focalLengthValue = (graphicsCanvas.width / 2) / Math.tan(fovRad / 2);
    } else {
        focalLengthValue = cameraSettings.focalLength;
    }
    
    camera = {
        x: cameraSettings.x,
        y: cameraSettings.y,
        z: cameraSettings.z,
        focalLength: focalLengthValue,        fov: cameraSettings.fov,
        near: cameraSettings.near,
        far: cameraSettings.far
    };


    // Initialize default objects like test cube and axes
    // Ensure createTestCube and createAxes are defined before this point
    if (CFG.SHOW_TEST_CUBE !== false) {
        testCube = createTestCube( // This was line 186            CFG.TEST_CUBE_SIZE,
            { x: CFG.TEST_CUBE_POSITION_X, y: CFG.TEST_CUBE_POSITION_Y, z: CFG.TEST_CUBE_POSITION_Z },
            CFG.TEST_CUBE_COLOR
        );
        addVectorObject(testCube);
    }

    if (CFG.SHOW_AXES !== false) {
        axes = createAxes(CFG.AXES_LENGTH);
        addVectorObject(axes);
    }    
}

// Matrix Operations
function createIdentityMatrix() {
    const mat = { m: [
        1, 0, 0, 0,
        0, 1, 0, 0,
        0, 0, 1, 0,
        0, 0, 0, 1
    ]};
    return mat;
}

function createTranslationMatrix(tx, ty, tz) {
    const mat = { m: [
        1,  0,  0,  0,
        0,  1,  0,  0,
        0,  0,  1,  0,
        tx, ty, tz, 1 // Translation components in the last row for column-major multiplication, or last col for row-major.
                      // Assuming standard transformation where P' = M * P (P is column vector)
                      // This setup is for M_translate * M_rotate * M_scale * P
                      // Or if P' = P * M (P is row vector), then translation is in the last row.
                      // Let's stick to a common convention: P' = T * R * S * P_local
                      // So, worldMatrix = TranslationMatrix * RotationMatrix * ScaleMatrix
                      // And vertex_world = worldMatrix * vertex_local
                      // For this, translation components are typically in the last column.
                      // Let's re-verify matrix multiplication order and structure.
                      // If m is stored row-major: [m00, m01, m02, m03, m10, ...]
                      // Translation (tx,ty,tz) goes to m[3], m[7], m[11] if post-multiplying (M*v)
                      // or m[12],m[13],m[14] if pre-multiplying (v*M) and matrix is row-major for v*M
                      // Or if m is column-major: [m00,m10,m20,m30, m01,m11,m21,m31, ...]
                      // Translation goes to m[12],m[13],m[14] for (M*v)
                      // The current multiplyMatrices function implies row-major storage and P' = P * M
                      // Let's assume the current matrix structure is row-major:
                      // [ m[0],  m[1],  m[2],  m[3]  ]
                      // [ m[4],  m[5],  m[6],  m[7]  ]
                      // [ m[8],  m[9],  m[10], m[11] ]
                      // [ m[12], m[13], m[14], m[15] ]
                      // For P' = P * M (P is row vector [x,y,z,1]), translation is in m[12],m[13],m[14]
    ]};
     // Correcting for typical P' = M * P with column vectors, and row-major storage of M
     // Translation components go into m[3], m[7], m[11] if matrix is:
     // [ R, R, R, Tx ]
     // [ R, R, R, Ty ]
     // [ R, R, R, Tz ]
     // [ 0, 0, 0, 1  ]
     // Or if matrix is stored as shown and P is column vector [x,y,z,w]
     // [m0, m4, m8,  m12] [x]
     // [m1, m5, m9,  m13] [y]    ]};
    return mat;
}

function createScaleMatrix(sx, sy, sz) {
    const mat = { m: [ // Column-major
        sx, 0,  0,  0,
        0,  sy, 0,  0,
        0,  0,  sz, 0,
        0,  0,  0,  1
    ]};

    return mat;
}

function createRotationMatrixX(angleRad) {
    const c = Math.cos(angleRad);
    const s = Math.sin(angleRad);
    const mat = { m: [ // Column-major
        1, 0,  0, 0,
        0, c,  s, 0, // sin is here for standard rotation around X affecting Y and Z
        0, -s, c, 0, // -sin is here
        0, 0,  0, 1
    ]};

    return mat;
}

function createRotationMatrixY(angleRad) {
    const c = Math.cos(angleRad);
    const s = Math.sin(angleRad);
    const mat = { m: [ // Column-major
        c, 0, -s, 0, // -sin is here for standard rotation around Y affecting X and Z
        0, 1,  0, 0,
        s, 0,  c, 0, // sin is here
        0, 0,  0, 1
    ]};

    return mat;
}

function createRotationMatrixZ(angleRad) {
    const c = Math.cos(angleRad);
    const s = Math.sin(angleRad);
    const mat = { m: [ // Column-major
        c,  s, 0, 0, // sin is here for standard rotation around Z affecting X and Y
        -s, c, 0, 0, // -sin is here
        0,  0, 1, 0,
        0,  0, 0, 1
    ]};

    return mat;
}

function multiplyMatrices(matA, matB) { // result = matA * matB
    const out = { m: [] };
    const a = matA.m;
    const b = matB.m;

    if (!a || !b) {
        console.error("[MATRIX] Error: Invalid matrices provided for multiplication.", matA, matB);
        return createIdentityMatrix(); 
    }
    
    // Assuming column-major storage: M_ij = M[i + j*4]
    // where i is row index (0-3), j is col index (0-3)
    // So a[0] is a00, a[1] is a10, a[4] is a01 etc.
    // C_ij = sum_k (A_ik * B_kj)

    for (let j = 0; j < 4; j++) { // Iterate over columns of B (and C)
        for (let i = 0; i < 4; i++) { // Iterate over rows of A (and C)
            let sum = 0;
            for (let k = 0; k < 4; k++) { // Iterate for dot product
                // C[i+j*4] = A[i+k*4] * B[k+j*4]
                sum += a[i + k * 4] * b[k + j * 4];
            }
            out.m[i + j * 4] = sum;
        }
    }


    return out;
}


// Object Management
function addVectorObject(obj) {
    if (!obj || typeof obj.id === 'undefined') {
        console.error("[VECTOR-MANAGER] Versuch, ein ungültiges oder ID-loses Objekt hinzuzufügen:", obj);
        return null;
    }
    // Check for duplicate ID
    if (vectorObjects.some(o => o.id === obj.id)) {
        console.warn(`[VECTOR-MANAGER] Objekt mit ID ${obj.id} existiert bereits. Überspringe Hinzufügen.`);
        return vectorObjects.find(o => o.id === obj.id); // Return existing object
    }
    vectorObjects.push(obj);

    return obj;
}

function removeVectorObject(id) {
    const initialLength = vectorObjects.length;
    vectorObjects = vectorObjects.filter(obj => obj.id !== id);
    if (vectorObjects.length < initialLength) {

        return true;
    }
    if (CFG.DEBUG_VECTOR_MANAGER) console.warn(`[VECTOR-MANAGER] Objekt zum Entfernen nicht gefunden (ID: ${id}).`);
    return false;
}

function getObjectById(id) {
    return vectorObjects.find(obj => obj.id === id) || null;
}

function getAllObjectIds() {
    return vectorObjects.map(obj => obj.id);
}

function setObjectVisibility(id, visible) {
    const obj = getObjectById(id);
    if (obj) {
        obj.visible = !!visible; // Ensure boolean

        return true;
    }
    if (CFG.DEBUG_VECTOR_MANAGER) console.warn(`[VECTOR-MANAGER] Objekt ${id} für Sichtbarkeitsänderung nicht gefunden.`);
    return false;
}


// Transformation Logic
function updateWorldMatrix(obj) {
    if (!obj || !obj.transform) {
        console.error("[MATRIX] updateWorldMatrix: Ungültiges Objekt oder fehlende Transformation.", obj);
        return;
    }

    let matrix = createIdentityMatrix();
    
    // Scale
    matrix = multiplyMatrices(matrix, createScaleMatrix(obj.transform.scale.x, obj.transform.scale.y, obj.transform.scale.z));
    
    // Rotate
    matrix = multiplyMatrices(matrix, createRotationMatrixX(obj.transform.rotation.x));
    matrix = multiplyMatrices(matrix, createRotationMatrixY(obj.transform.rotation.y));
    matrix = multiplyMatrices(matrix, createRotationMatrixZ(obj.transform.rotation.z));
    
    // Translate
    matrix = multiplyMatrices(matrix, createTranslationMatrix(obj.transform.translation.x, obj.transform.translation.y, obj.transform.translation.z));
    
    obj.worldMatrix = matrix;

}

// Added definition for updateObjectTransform
function updateObjectTransform(id, newTransform) {
    const obj = getObjectById(id);
    if (obj && obj.transform && newTransform) {
        if (newTransform.translation) {
            obj.transform.translation = { ...obj.transform.translation, ...newTransform.translation };
        }
        if (newTransform.rotation) {
            obj.transform.rotation = { ...obj.transform.rotation, ...newTransform.rotation };
        }
        if (newTransform.scale) {
            obj.transform.scale = { ...obj.transform.scale, ...newTransform.scale };
        }
        updateWorldMatrix(obj); // Recalculate the world matrix

        return true;
    } else {
        console.warn(`[VECTOR-MANAGER] updateObjectTransform: Objekt ${id} nicht gefunden, hat keine Transformationseigenschaft oder keine neue Transformation bereitgestellt.`);
        return false;
    }
}

function rotateObject(id, axis, angleRad) {
    const obj = getObjectById(id);
    if (obj) {
        if (axis === 'x') obj.transform.rotation.x += angleRad;
        else if (axis === 'y') obj.transform.rotation.y += angleRad;
        else if (axis === 'z') obj.transform.rotation.z += angleRad;
        else {
            console.error("[VECTOR-MANAGER] rotateObject: Ungültige Achse:", axis);
            return;
        }
        updateWorldMatrix(obj);

    } else {
        console.warn(`[VECTOR-MANAGER] rotateObject: Objekt ${id} nicht gefunden.`);
    }
}

function translateObject(id, dx, dy, dz) {
    const obj = getObjectById(id);
    if (obj) {
        obj.transform.translation.x += dx;
        obj.transform.translation.y += dy;
        obj.transform.translation.z += dz;
        updateWorldMatrix(obj);

    } else {
        console.warn(`[VECTOR-MANAGER] translateObject: Objekt ${id} nicht gefunden.`);
    }
}

function scaleObject(id, sx, sy, sz) {
    const obj = getObjectById(id);
    if (obj) {
        // Assuming sx, sy, sz are multipliers for the current scale
        obj.transform.scale.x *= sx;
        obj.transform.scale.y *= sy;
        obj.transform.scale.z *= sz;
        updateWorldMatrix(obj);

    } else {
        console.warn(`[VECTOR-MANAGER] scaleObject: Objekt ${id} nicht gefunden.`);
    }
}


// Rendering Logic
function projectVertex(vertex, worldMatrix, camera, graphicsCanvas) {
    // Apply world matrix to vertex
    const world_x = vertex.x * worldMatrix.m[0] + vertex.y * worldMatrix.m[4] + vertex.z * worldMatrix.m[8] + worldMatrix.m[12];
    const world_y = vertex.x * worldMatrix.m[1] + vertex.y * worldMatrix.m[5] + vertex.z * worldMatrix.m[9] + worldMatrix.m[13];
    const world_z = vertex.x * worldMatrix.m[2] + vertex.y * worldMatrix.m[6] + vertex.z * worldMatrix.m[10] + worldMatrix.m[14];

    const x_rotated = world_x - camera.x; 
    const y_rotated = world_y - camera.y; 
    const depth = camera.z - world_z;

    if (depth <= camera.near || depth >= camera.far) { 
        return { x: 0, y: 0, projectedX: 0, projectedY: 0, scale: 0, visible: false, reason: depth <= camera.near ? "near_clip" : "far_clip" };
    }

    const scale = camera.focalLength / depth;
    const projectedX = x_rotated * scale;
    const projectedY = y_rotated * scale; 
    const screenX = graphicsCanvas.width / 2 + projectedX;
    const screenY = graphicsCanvas.height / 2 - projectedY; 


    return { x: world_x, y: world_y, z: world_z, projectedX, projectedY, screenX, screenY, scale, visible: true };
}


function renderVectors(ctx, canvasWidth, canvasHeight) {
    if (!graphicsCanvas || !camera || !ctx) { // Added ctx check here
        console.warn("[VECTOR-MANAGER-RENDER] Graphics components not initialized. Skipping render.", 
                     { hasCanvas: !!graphicsCanvas, hasCamera: !!camera, hasContext: !!ctx });
        return;
    }
    
    // Clear is done by retrographics before calling this

    vectorObjects.forEach((obj, index) => {
        if (!obj || !obj.visible) {
            return; // Skip if object is null or not visible
        }

        if (!obj.vertices || !obj.edges) {
            console.warn('[VECTOR-MANAGER-RENDER] Skipping invalid object (missing vertices, or edges):', obj.id, obj);
            return; 
        }
        
        if (obj.worldMatrix === undefined || obj.worldMatrix === null || typeof obj.worldMatrix.m === 'undefined') { 
            console.error(`[VECTOR-MANAGER-RENDER] CRITICAL: worldMatrix is invalid for object ID ${obj.id}. Skipping rendering this object.`, obj.worldMatrix);
            return; 
        }

        const objectColor = obj.color || '#FFFFFF';
        ctx.strokeStyle = objectColor;
        ctx.lineWidth = obj.lineWidth || 1; // Allow objects to specify line width
        ctx.beginPath();

        obj.edges.forEach((edge, edgeIndex) => {
            if (!Array.isArray(edge) || edge.length < 2) {
                console.warn('[VECTOR-MANAGER-RENDER] Skipping invalid edge (not an array or less than 2 vertices):', edge, 'for object:', obj.id);
                return; 
            }

            const projected = edge.map(vertexIndex => {
                if (typeof vertexIndex !== 'number' || vertexIndex < 0 || vertexIndex >= obj.vertices.length) {
                    console.error(`[VECTOR-MANAGER-RENDER] Invalid vertexIndex ${vertexIndex} for object ${obj.id} with ${obj.vertices.length} vertices.`);
                    return { visible: false, reason: "invalid_vertex_index" };
                }
                const vertex = obj.vertices[vertexIndex];
                if (!vertex || typeof vertex.x === 'undefined' || typeof vertex.y === 'undefined' || typeof vertex.z === 'undefined') {
                     console.error(`[VECTOR-MANAGER-RENDER] Vertex at index ${vertexIndex} for object ${obj.id} is undefined or malformed.`, vertex);
                     return { visible: false, reason: "undefined_or_malformed_vertex" };
                }
                return projectVertex(vertex, obj.worldMatrix, camera, graphicsCanvas);
            });

            const visiblePoints = projected.filter(p => p && p.visible);

            if (visiblePoints.length === 2) {
                // Handle per-edge coloring for axes if applicable
                if (obj.id && obj.id.startsWith('axes_') && obj.edgeColors && obj.edgeColors[edgeIndex]) {
                    ctx.stroke(); // Stroke previous path if any with default color
                    ctx.strokeStyle = obj.edgeColors[edgeIndex];
                    ctx.beginPath(); // Start new path for this colored edge
                    ctx.moveTo(visiblePoints[0].screenX, visiblePoints[0].screenY);
                    ctx.lineTo(visiblePoints[1].screenX, visiblePoints[1].screenY);
                    ctx.stroke(); // Stroke this colored edge
                    ctx.strokeStyle = objectColor; // Reset to object's default color
                    ctx.beginPath(); // Start new path for subsequent edges of this object
                } else {
                    ctx.moveTo(visiblePoints[0].screenX, visiblePoints[0].screenY);
                    ctx.lineTo(visiblePoints[1].screenX, visiblePoints[1].screenY);
                }
            }
        });
        ctx.stroke(); // Stroke any remaining path for the object
    });
}


// Handler function for UPDATE_VECTOR commands from backend
function handleUpdateVector3D(data) {
    // Parse the vector update data from the BASIC VECTOR command
    // Backend sendet lowercase Feldnamen: id, shape, position, vecRotation, scale, brightness, visible
    
    // Handle both full vector updates and partial updates (e.g., visibility/scale-only updates)
    if (data.id !== undefined) {
        const id = data.id;
        
        // Find existing object
        let obj = vectorObjects.find(o => o.id === `vector_${id}`);
        
        // For full vector creation/update (has shape, position, rotation, scale)
        if (data.shape && data.position && data.vecRotation && data.scale !== undefined) {
            const shape = data.shape; // "cube", "sphere", etc.
            const pos = data.position; // {x, y, z}
            const rot = data.vecRotation; // {x, y, z} in radians
            const scale = data.scale;
            const brightness = data.brightness || 15;
            
            if (!obj) {
                // Create new vector object
                if (shape === "cube") {
                    obj = createTestCube(scale, pos, getBrightnessColor(brightness));
                    obj.id = `vector_${id}`;
                    obj.originalId = id;
                    obj.shape = shape;
                } else if (shape === "sphere") {
                    // Use proper sphere geometry
                    obj = createSphere(scale / 2, pos, getBrightnessColor(brightness)); // scale/2 because createSphere takes radius
                    obj.id = `vector_${id}`;
                    obj.originalId = id;
                    obj.shape = shape;
                } else {
                    // Default to cube for unknown shapes
                    obj = createTestCube(scale, pos, getBrightnessColor(brightness));
                    obj.id = `vector_${id}`;
                    obj.originalId = id;
                    obj.shape = shape;
                }
                vectorObjects.push(obj);
            }
            
            // Update object properties
            obj.transform.translation.x = pos.x;
            obj.transform.translation.y = pos.y;
            obj.transform.translation.z = pos.z;
            obj.transform.rotation.x = rot.x;
            obj.transform.rotation.y = rot.y;
            obj.transform.rotation.z = rot.z;
            
            // Update scale - handle both uniform and per-axis scaling
            if (typeof scale === 'number') {
                obj.transform.scale.x = scale;
                obj.transform.scale.y = scale;
                obj.transform.scale.z = scale;
            } else if (scale && typeof scale === 'object') {
                obj.transform.scale.x = scale.x || obj.transform.scale.x;
                obj.transform.scale.y = scale.y || obj.transform.scale.y;
                obj.transform.scale.z = scale.z || obj.transform.scale.z;
            }
            
            // Update color based on brightness
            obj.color = getBrightnessColor(brightness);
        }
        
        // Handle partial updates for existing objects
        if (obj) {
            // Update visibility (for VECTOR.HIDE/VECTOR.SHOW commands)
            if (data.visible !== undefined) {
                obj.visible = !!data.visible;
            }
            
            // Update scale only (for VECTOR.SCALE commands)
            if (data.scale !== undefined && !data.shape) {
                if (typeof data.scale === 'number') {
                    obj.transform.scale.x = data.scale;
                    obj.transform.scale.y = data.scale;
                    obj.transform.scale.z = data.scale;
                } else if (data.scale && typeof data.scale === 'object') {
                    obj.transform.scale.x = data.scale.x || obj.transform.scale.x;
                    obj.transform.scale.y = data.scale.y || obj.transform.scale.y;
                    obj.transform.scale.z = data.scale.z || obj.transform.scale.z;
                }
            }
              
            // Update brightness (for VECTOR.SCALE with brightness parameter)
            if (data.brightness !== undefined) {
                obj.color = getBrightnessColor(data.brightness);
            }            
            // Update world matrix
            updateWorldMatrix(obj);
            
            // Setze das Dirty-Flag für Vector-Rendering
            window.RetroGraphics._vectorsDirty = true;
        }
    } else {
        // Fallback: Default behavior for legacy test cube
        if (testCube) {
            testCube.transform.rotation.y += toRadians(1);
            updateWorldMatrix(testCube);
            // Setze das Dirty-Flag für Vector-Rendering
            window.RetroGraphics._vectorsDirty = true;
        }
    }
}

// Helper function to convert brightness (0-15) to color
function getBrightnessColor(brightness) {
    brightness = Math.max(0, Math.min(15, brightness || 15));
    
    // Use the green color palette from config instead of grayscale
    const colors = CFG.BRIGHTNESS_LEVELS || [
        '#000000', // 0 - black
        '#001500', '#002500', '#003500', '#004500', // Very dark green
        '#005500', '#006000', '#007000', '#008000', // Dark green
        '#009000', '#00A000', '#00B000', '#00C000', // Medium green 
        '#00D000', '#00E000', '#5FFF5F'             // Bright green (matches text color)
    ];
    
    return colors[brightness];
}

// Clear all vector objects
function clearAllVectorObjects3D() {

    
    vectorObjects = [];
    testCube = null;
    axes = null;
    objectIdCounter = 0;
    
    // Clear the graphics canvas
    if (graphicsCanvas && graphicsCanvas.getContext) {
        const ctx = graphicsCanvas.getContext('2d');
        if (ctx) {
            ctx.clearRect(0, 0, graphicsCanvas.width, graphicsCanvas.height);
        }
    }
}


// Exposed API
window.vectorManager = {
    initVectorManager,
    addVectorObject,
    removeVectorObject,
    updateObjectTransform, // Higher-level if it combines T,R,S
    getObjectById,
    getAllObjectIds,
    setObjectVisibility,
    
    // Direct transform setters/modifiers
    rotateObject, 
    translateObject,
    scaleObject,
    
    renderVectors, // Exposed primarily for retrographics to call

    // Debug / Accessors
    getTestCube: () => testCube,
    getAxes: () => axes,
    getCamera: () => camera,
    getGraphicsCanvas: () => graphicsCanvas,

    // Utilities that might be useful externally
    toRadians,
    createIdentityMatrix,    createTranslationMatrix,
    createScaleMatrix,
    createRotationMatrixX,
    createRotationMatrixY,
    createRotationMatrixZ,
    multiplyMatrices,
    updateWorldMatrix, // If external logic modifies obj.transform directly
    handleUpdateVector3D,
    clearAllVectorObjects3D
};



// Make vectorManager globally available
window.vectorManager = {
    initVectorManager,
    renderVectors,
    createTestCube,
    createSphere,
    createAxes,
    addVectorObject,
    removeVectorObject,
    getVectorObjects: () => vectorObjects,
    getCamera: () => camera,
    getGraphicsCanvas: () => graphicsCanvas,
    
    // Utilities that might be useful externally
    toRadians,
    createIdentityMatrix,
    createTranslationMatrix,
    createScaleMatrix,
    createRotationMatrixX,
    createRotationMatrixY,
    createRotationMatrixZ,
    multiplyMatrices,
    updateWorldMatrix,
    handleUpdateVector3D,
    clearAllVectorObjects3D
};
