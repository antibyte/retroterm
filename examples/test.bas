10 REM Test der repartierten Physics Engine
20 CLS
30 
40 PRINT "=== PHYSICS REPAIR TEST ==="
50 PRINT "Testing fixed physics system..."
60 
70 REM Initialize Physics exactly like physics.bas
80 PHYSICS WORLD 0, 300
90 PHYSICS SCALE 30
100 
110 REM Create boundaries
120 PHYSICS FLOOR 0, 450, 640, 450
130 PHYSICS WALL 0, 0, 0, 480
140 PHYSICS WALL 640, 0, 640, 480
150 
160 REM Create falling objects (exactly like physics.bas)
170 CIRCLE 200, 100, 20, 4, 1
180 PHYSICS CIRCLE 200, 100, 20, 1
190 PHYSICS BOUNCE 1, 0.0
200 PHYSICS FRICTION 1, 0.3
210 PHYSICS VELOCITY 1, 50, -100
220 
230 CIRCLE 400, 120, 25, 6, 1
240 PHYSICS CIRCLE 400, 120, 25, 2
250 PHYSICS BOUNCE 2, 0.0
260 PHYSICS FRICTION 2, 0.4
270 PHYSICS VELOCITY 2, -60, -80
280 
290 REM Create dynamic rectangles
300 RECT 300, 80, 35, 18, 8, 1
310 PHYSICS RECT 300, 80, 35, 18, 10, "dynamic"
320 PHYSICS DENSITY 10, 1.2
330 PHYSICS FRICTION 10, 0.7
340 PHYSICS BOUNCE 10, 0.0
350 
360 REM Start physics simulation
370 PHYSICS AUTO ON
380 PRINT "Physics simulation started"
390 PRINT "Objects should fall, bounce and settle"
400 
410 REM Run simulation for 15 seconds (like physics.bas)
420 FOR I = 1 TO 1500
430   WAIT 30
440   IF I MOD 300 = 0 THEN PRINT "Time: "; I*30; "ms - should see movement"
450 NEXT I
460 
470 PHYSICS AUTO OFF
480 PRINT ""
490 PRINT "=== TEST COMPLETE ==="
500 PRINT "Did you see moving circles and rectangles?"
510 PRINT "If YES: Physics is repaired!"
520 PRINT "If NO: Still broken"
530 END