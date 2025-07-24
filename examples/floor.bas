REM ================================================================================
REM                         GRID FLOOR DEMONSTRATION  
REM                    VECFLOOR AND VECNODE COMMANDS
REM ================================================================================
10 CLS
20 PRINT "Grid Floor Demo - VECFLOOR & VECNODE"
30 PRINT "Creating 3D grid floors with terrain..."
40 PRINT "-----------------------------------"
50 WAIT 1000

100 REM Clean up any existing vector objects
110 FOR I = 0 TO 255
120   VECTOR.HIDE I
130 NEXT I

200 REM Create basic 16x16 grid floor
210 PRINT "Creating 16x16 grid floor..."
220 VECFLOOR 1, 0, -8, -6, 0, 0, 0, 16, 16, 1.5, 12
230 PRINT "Floor created - watching for 5 seconds..."
240 WAIT 5000

300 REM Create some terrain features using VECNODE
310 PRINT "Adding terrain features..."
320 REM Create a mountain in the center
330 VECNODE 7, 7, 4
340 VECNODE 8, 8, 5
350 VECNODE 9, 9, 3
360 VECNODE 8, 7, 4
370 VECNODE 7, 8, 4

400 REM Create a valley on the left side  
410 VECNODE 3, 5, -2
420 VECNODE 4, 5, -3
430 VECNODE 5, 5, -2
440 VECNODE 4, 4, -2
450 VECNODE 4, 6, -2

500 REM Create rolling hills on the right
510 VECNODE 12, 3, 2
520 VECNODE 13, 4, 3
530 VECNODE 14, 5, 1
540 VECNODE 11, 6, 2
550 VECNODE 13, 8, 2

600 REM Add some random terrain variation
610 FOR I = 1 TO 10
620   LET X = RND(14) + 1
630   LET Z = RND(14) + 1  
640   LET H = (RND(3) - 1)
650   VECNODE X, Z, H
660   WAIT 100
670 NEXT I

700 PRINT "Terrain complete!"
705 WAIT 5000
710 PRINT ""
720 PRINT "Commands demonstrated:"
730 PRINT "VECFLOOR id, x,y,z, rotX,rotY,rotZ, width,depth, spacing, brightness"
740 PRINT "VECNODE gridX, gridZ, height"
750 PRINT ""
760 PRINT "Press any key for animated terrain..."
770 IF INKEY$ = "" THEN GOTO 770

800 REM Animated terrain demo
810 PRINT "Creating animated waves..."
820 FOR FRAME = 1 TO 30
830   FOR X = 0 TO 15
840     FOR Z = 0 TO 15
850       LET WAVE = SIN((X + FRAME * 0.3) * 0.4) * COS((Z + FRAME * 0.3) * 0.4) * 1.5
860       VECNODE X, Z, WAVE
870     NEXT Z
880   NEXT X
890   WAIT 150
900 NEXT FRAME

950 PRINT "Animation complete!"
960 PRINT "Press any key to create second floor..."
970 IF INKEY$ = "" THEN GOTO 970

1000 REM Create second floor at different position
1010 REM Hide first floor to avoid conflicts
1020 VECTOR.HIDE 1
1030 PRINT "Creating second 8x8 floor..."
1040 VECFLOOR 2, 15, -3, -4, 15, 0, 0, 8, 8, 2, 8

1100 REM Create a simple pyramid on second floor
1110 FOR I = 0 TO 3
1120   FOR J = 0 TO 3
1130     LET HEIGHT = 4 - (ABS(I-2) + ABS(J-2))
1140     IF HEIGHT > 0 THEN VECNODE I+2, J+2, HEIGHT
1150   NEXT J
1160 NEXT I
1170 PRINT "Second floor with pyramid created!"
1180 WAIT 5000

1200 PRINT "Demo complete!"
1210 PRINT "Two floors with different terrain created."
1220 PRINT ""
1230 PRINT "VECFLOOR parameters:"
1240 PRINT "- id: Vector object ID (0-255)"
1250 PRINT "- x,y,z: World position"
1260 PRINT "- rotX,rotY,rotZ: Rotation angles"
1270 PRINT "- width,depth: Grid dimensions (max 256)"
1280 PRINT "- spacing: Distance between grid points"
1290 PRINT "- brightness: Display brightness (0-15)"
1300 PRINT ""
1310 PRINT "VECNODE parameters:"
1320 PRINT "- gridX,gridZ: Grid coordinates (0-based)"
1330 PRINT "- height: Elevation (-100 to +100)"
1340 PRINT ""
1350 PRINT "Press any key to end..."
1360 IF INKEY$ = "" THEN GOTO 1360

1400 END