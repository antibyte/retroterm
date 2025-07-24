// imageManager.js - Manages IMAGE commands for TinyBASIC

// Use global CFG from config.js

// Global variables
let imageCanvas = null;
let imageObjects = [];
let maxImageHandle = 8;

// Initialize the image manager
function initImageManager() {
    imageObjects = [];
    
    // Initialize image dirty flag
    if (!window.RetroGraphics) {
        window.RetroGraphics = {};
    }
    if (typeof window.RetroGraphics._imagesDirty === 'undefined') {
        window.RetroGraphics._imagesDirty = false;
    }
}

// Create a new image object
function createImageObject(handle, imageData, width, height, filename) {
    const img = new Image();
    
    return new Promise((resolve, reject) => {
        img.onload = function() {
            const imageObj = {
                handle: handle,
                image: img,
                width: width,
                height: height,
                filename: filename,
                visible: false,
                x: 0,
                y: 0,
                scale: 0, // 0 = original size
                rotation: 0, // in radians
                loaded: true
            };
            
            // Remove existing image with same handle
            imageObjects = imageObjects.filter(obj => obj.handle !== handle);
            
            // Add new image
            imageObjects.push(imageObj);
            
            resolve(imageObj);
        };
        
        img.onerror = function() {
            console.error('[IMAGE-MANAGER] Failed to load image:', filename);
            reject(new Error('Failed to load image'));
        };
        
        // Set image source from base64 data
        img.src = 'data:image/png;base64,' + imageData;
    });
}

// Find image by handle
function findImageByHandle(handle) {
    return imageObjects.find(obj => obj.handle === handle);
}

// Convert scale parameter to actual scale factor
function getScaleFactor(scaleParam) {
    if (scaleParam === 0) return 1.0; // Original size
    if (scaleParam > 0) {
        // Positive: 1 = double, 2 = quadruple
        return 1.0 + scaleParam;
    } else {
        // Negative: -1 = half, -2 = quarter
        return 1.0 / (1.0 - scaleParam);
    }
}

// Render all visible images
function renderImages(ctx, canvasWidth, canvasHeight) {
    if (!ctx) {
        return;
    }
    
    // Render all visible images
    imageObjects.forEach(imgObj => {
        if (!imgObj.visible || !imgObj.loaded) {
            return;
        }
        
        ctx.save();
        
        // Calculate actual scale factor
        const scaleFactor = getScaleFactor(imgObj.scale);
        
        // Calculate dimensions
        const displayWidth = imgObj.width * scaleFactor;
        const displayHeight = imgObj.height * scaleFactor;
        
        // Set up transformation
        ctx.translate(imgObj.x + displayWidth / 2, imgObj.y + displayHeight / 2);
        ctx.rotate(imgObj.rotation);
        ctx.scale(scaleFactor, scaleFactor);
        
        // Draw image centered at origin
        ctx.drawImage(imgObj.image, -imgObj.width / 2, -imgObj.height / 2);
        
        ctx.restore();
    });
}

// Handle LOAD_IMAGE command
function handleLoadImage(data) {
    const handle = data.id;
    const customData = data.customData;
    
    if (!customData || !customData.imageData) {
        console.error('[IMAGE-MANAGER] LOAD_IMAGE missing image data');
        return false;
    }
    
    const imageData = customData.imageData;
    const width = customData.width || 0;
    const height = customData.height || 0;
    const filename = customData.filename || 'unknown';
    
    // Create image object asynchronously
    createImageObject(handle, imageData, width, height, filename)
        .then(imageObj => {
            // Mark for re-render
            window.RetroGraphics._imagesDirty = true;
        })
        .catch(error => {
            console.error('[IMAGE-MANAGER] Failed to load image:', error);
        });
    
    return true;
}

// Handle SHOW_IMAGE command
function handleShowImage(data) {
    const handle = data.id;
    const position = data.position;
    const scale = data.scale || 0;
    
    const imgObj = findImageByHandle(handle);
    if (!imgObj) {
        console.warn('[IMAGE-MANAGER] SHOW_IMAGE: Image not found for handle:', handle);
        return false;
    }
    
    // Update image properties
    imgObj.visible = true;
    imgObj.x = position.x || 0;
    imgObj.y = position.y || 0;
    imgObj.scale = scale;
    
    // Mark for re-render
    window.RetroGraphics._imagesDirty = true;
    
    return true;
}

// Handle HIDE_IMAGE command
function handleHideImage(data) {
    const handle = data.id;
    
    const imgObj = findImageByHandle(handle);
    if (!imgObj) {
        console.warn('[IMAGE-MANAGER] HIDE_IMAGE: Image not found for handle:', handle);
        return false;
    }
    
    imgObj.visible = false;
    
    // Mark for re-render
    window.RetroGraphics._imagesDirty = true;
    
    return true;
}

// Handle ROTATE_IMAGE command
function handleRotateImage(data) {
    const handle = data.id;
    const rotation = data.vecRotation;
    
    const imgObj = findImageByHandle(handle);
    if (!imgObj) {
        console.warn('[IMAGE-MANAGER] ROTATE_IMAGE: Image not found for handle:', handle);
        return false;
    }
    
    // Update rotation (rotation.z is in radians)
    if (rotation && typeof rotation.z === 'number') {
        imgObj.rotation = rotation.z;
    }
    
    // Mark for re-render
    window.RetroGraphics._imagesDirty = true;
    
    return true;
}

// Clear all images
function clearAllImages() {
    imageObjects = [];
    window.RetroGraphics._imagesDirty = true;
}

// Get debug info
function getImageDebugInfo() {
    return {
        totalImages: imageObjects.length,
        visibleImages: imageObjects.filter(obj => obj.visible).length,
        loadedImages: imageObjects.filter(obj => obj.loaded).length,
        images: imageObjects.map(obj => ({
            handle: obj.handle,
            filename: obj.filename,
            visible: obj.visible,
            loaded: obj.loaded,
            size: obj.width + 'x' + obj.height
        }))
    };
}

// Export functions for global access
window.imageManager = {
    initImageManager,
    renderImages,
    handleLoadImage,
    handleShowImage,
    handleHideImage,
    handleRotateImage,
    clearAllImages,
    getImageDebugInfo,
    getImageObjects: () => imageObjects
};

// Auto-initialize when loaded
if (typeof window !== 'undefined') {
    // Immediately initialize imageManager
    initImageManager();
    
    // Also dispatch an event to notify other components
    if (typeof document !== 'undefined') {
        const event = new CustomEvent('imagemanagerready');
        document.dispatchEvent(event);
    }
}