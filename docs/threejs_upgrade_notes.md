# Three.js Upgrade Notes

## Current Status
- **Version**: Using three.min.js (legacy build)
- **Issue**: Deprecation warning - legacy builds will be removed in r160
- **Impact**: Non-critical, but should be addressed in future maintenance

## Deprecation Warning
```
Scripts "build/three.js" and "build/three.min.js" are deprecated with r150+, and will be removed with r160. 
Please use ES Modules or alternatives: https://threejs.org/docs/index.html#manual/en/introduction/Installation
```

## Recommendation
When updating Three.js:
1. Switch to ES Module imports instead of script tag
2. Update `js/retrographics.js`, `js/spriteManager.js`, and `js/vectorManager.js` to use ES imports
3. Test 3D graphics functionality thoroughly after upgrade

## Files Using Three.js
- `js/retrographics.js` - Main graphics rendering
- `js/spriteManager.js` - Sprite management
- `js/vectorManager.js` - Vector graphics
- `retroterminal.html` - Script inclusion

## Current Priority: LOW
The current implementation works correctly. This is a future maintenance task.
