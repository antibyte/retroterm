REM Pyramid and Cylinder Demo
REM Demonstrates the new 3D primitive commands

CLS
PRINT "Pyramid and Cylinder Demo"
PRINT "========================="
PRINT ""
PRINT "Creating various pyramid shapes..."
PRINT ""

REM Clear any existing vectors
FOR I = 0 TO 10
    VECTOR.HIDE I
NEXT I

REM Create different pyramid types
PYRAMID 1, "triangle", -30, 0, -20, 0, 0, 0, 15, 20, 15
PYRAMID 2, "square", 0, 0, -20, 0, 30, 0, 15, 25, 12
PYRAMID 3, "pentagon", 30, 0, -20, 0, 60, 0, 12, 18, 10
PYRAMID 4, "hexagon", 60, 0, -20, 0, 90, 0, 10, 22, 8

PRINT "Triangle pyramid (ID 1) - green"
PRINT "Square pyramid (ID 2) - rotating 30° Y"  
PRINT "Pentagon pyramid (ID 3) - rotating 60° Y"
PRINT "Hexagon pyramid (ID 4) - rotating 90° Y"
PRINT ""
PRINT "Creating cylinders with different line counts..."
PRINT ""

REM Create cylinders with different properties
CYLINDER 5, -20, -30, -10, 0, 0, 0, 8, 25, 15
CYLINDER 6, 20, -30, -10, 0, 45, 0, 10, 20, 12
CYLINDER 7, 0, -30, 10, 45, 0, 0, 6, 30, 10

PRINT "Cylinder 1 (ID 5) - radius 8, height 25, 8 lines"
PRINT "Cylinder 2 (ID 6) - radius 10, height 20, rotated 45° Y"
PRINT "Cylinder 3 (ID 7) - radius 6, height 30, rotated 45° X"
PRINT ""
PRINT "Animation loop starting..."
PRINT "Press any key to stop..."

REM Animation loop
T = 0
WHILE INKEY$ = ""
    T = T + 2
    
    REM Rotate pyramids
    PYRAMID 1, "triangle", -30, SIN(T/10)*5, -20, T, 0, T/2, 15, 20, 15
    PYRAMID 2, "square", 0, COS(T/15)*3, -20, T/2, 30+T, 0, 15, 25, 12
    PYRAMID 3, "pentagon", 30, SIN(T/20)*4, -20, T/3, 60+T/2, 0, 12, 18, 10
    PYRAMID 4, "hexagon", 60, COS(T/12)*6, -20, T/4, 90+T/3, 0, 10, 22, 8
    
    REM Animate cylinders  
    CYLINDER 5, -20, -30+SIN(T/8)*5, -10, 0, T, 0, 8, 25, 15
    CYLINDER 6, 20, -30+COS(T/12)*3, -10, 0, 45+T/2, 0, 10, 20, 12
    CYLINDER 7, SIN(T/10)*10, -30, 10, 45+T/3, 0, T/4, 6, 30, 10
    
    WAIT 50
WEND

PRINT ""
PRINT "Animation stopped."
PRINT ""
PRINT "Commands used in this demo:"
PRINT "PYRAMID id, base, x, y, z, rotX, rotY, rotZ, scale, height, [brightness]"
PRINT "CYLINDER id, x, y, z, rotX, rotY, rotZ, radius, height, [brightness]"
PRINT ""
PRINT "Base shapes: triangle, square, pentagon, hexagon"
PRINT "Cylinder has 8 connecting lines by default"