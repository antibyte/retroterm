10 REM 5 Pacman Sprites mit Physics Demo
20 CLS

30 PRINT "=== 5 PACMAN PHYSICS DEMO ==="
40 PRINT "Creating Pacman sprites with physics..."

50 REM Initialize Physics
60 PHYSICS WORLD 0, 200
70 PHYSICS SCALE 30

80 REM Create boundaries
90 PHYSICS FLOOR 50, 420, 590, 420
100 PHYSICS WALL 50, 50, 50, 420
110 PHYSICS WALL 590, 50, 590, 420

120 REM Create Pacman sprite data (simplified for exact 1024 values)
130 LET PACMAN$ = ""
140 FOR Y = 0 TO 31
150   FOR X = 0 TO 31
160     REM Create Pacman shape
170     LET DX = X - 16
180     LET DY = Y - 16
190     LET DIST = SQR(DX*DX + DY*DY)
200     
210     REM Pacman circle with mouth
220     IF DIST > 14 THEN LET PIXEL = 0 : GOTO 260
230     IF Y > 14 AND Y < 18 AND X > 16 THEN LET PIXEL = 0 : GOTO 260
240     LET PIXEL = 10
250     
260     LET PACMAN$ = PACMAN$ + STR$(PIXEL)
270     IF X < 31 OR Y < 31 THEN LET PACMAN$ = PACMAN$ + ","
280   NEXT X
290 NEXT Y

300 PRINT "Defining Pacman sprite..."
310 SPRITE 1, PACMAN$

320 PRINT "Creating 5 Pacman sprites..."
330 REM Create 5 Pacman sprite instances with correct syntax
340 SPRITE 1 AT 150, 100
350 SPRITE 1 ON
360 
370 SPRITE 2 AT 250, 120
380 SPRITE 2 ON

390 SPRITE 3 AT 350, 140
400 SPRITE 3 ON

410 SPRITE 4 AT 450, 160
420 SPRITE 4 ON

430 SPRITE 5 AT 550, 180
440 SPRITE 5 ON

450 PRINT "Adding physics to sprites..."
460 REM Add physics to all 5 sprites
470 SPRITE PHYSICS 1 dynamic box 1.5
480 SPRITE PHYSICS 2 dynamic box 1.8
490 SPRITE PHYSICS 3 dynamic box 2.0
500 SPRITE PHYSICS 4 dynamic box 1.2
510 SPRITE PHYSICS 5 dynamic box 1.7

520 PRINT "Setting initial velocities..."
530 REM Give each Pacman different velocities
540 PHYSICS VELOCITY 1, 60, -80
550 PHYSICS VELOCITY 2, -50, -100
560 PHYSICS VELOCITY 3, 80, -60
570 PHYSICS VELOCITY 4, -70, -90
580 PHYSICS VELOCITY 5, 40, -120

590 PRINT "Starting physics simulation..."
600 PHYSICS AUTO ON

610 PRINT ""
620 PRINT "5 PACMAN PHYSICS ACTIVE!"
630 PRINT "Pacmans should bounce around without trails"
640 PRINT "Observing for 20 seconds..."

650 REM Run simulation for 20 seconds
660 FOR T = 1 TO 2000
670   WAIT 30
680   IF T MOD 500 = 0 THEN PRINT "Time: "; T*30; "ms - Pacmans bouncing"
690 NEXT T

700 PHYSICS AUTO OFF
710 PRINT ""
720 PRINT "=== DEMO COMPLETE ==="
730 PRINT "Did you see 5 Pacmans bouncing around?"
740 PRINT "Were there any sprite trails?"
750 END