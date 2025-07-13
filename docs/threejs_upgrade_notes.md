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

## Updated Status (July 2025)
- Console warning confirmed but system fully functional
- Chess interface working correctly with Three.js graphics
- All bitmap rendering and 3D graphics operational
- Deprecation warning is cosmetic and doesn't affect functionality

## Immediate Workaround
To suppress the warning temporarily, you could:
1. Comment out the warning line in `js/three.min.js` (line 1)
2. Or wait for the warning to be addressed in a future Three.js update

## Long-term Solution Plan
1. Switch to ES Modules when ready for a major update
2. Update build system to handle ES Module imports
3. Test all graphics functionality after migration
4. Consider using a CDN version instead of local files
