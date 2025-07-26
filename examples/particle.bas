10 REM PARTICLE System Demo - TinyBASIC Graphics
20 REM Demonstrates all PARTICLE commands and effects
30 REM Shows different emitter types, gravity, and animations
40 REM 
50 PRINT "PARTICLE SYSTEM DEMONSTRATION"
60 PRINT "============================"
70 PRINT ""
80 PRINT "This demo shows all PARTICLE commands:"
90 PRINT "- PARTICLE CREATE: 4 different emitter types"
100 PRINT "- PARTICLE MOVE: Position emitters"
110 PRINT "- PARTICLE SHOW/HIDE: Control visibility"
120 PRINT "- PARTICLE GRAVITY: Physics simulation"
130 PRINT ""
140 PRINT "Press any key to start..."
150 A$ = INKEY$: IF A$ = "" THEN GOTO 150
160 PRINT ""
170 REM
180 REM Phase 1: Create different emitter types
190 REM
200 PRINT "PHASE 1: Creating Emitters"
210 PRINT "-------------------------"
220 PRINT "Creating POINT emitter (ID 1)..."
230 PARTICLE CREATE 1, point, 15, 30, 4
240 PRINT "Creating STAR emitter (ID 2)..."
250 PARTICLE CREATE 2, star, 12, 25, 5
260 PRINT "Creating CIRCLE emitter (ID 3)..."
270 PARTICLE CREATE 3, circle, 20, 35, 4
280 PRINT "Creating RECT emitter (ID 4)..."
290 PARTICLE CREATE 4, rect, 18, 28, 6
300 PRINT ""
310 PRINT "All emitters created! (Not visible until positioned)"
320 PRINT "Press any key for Phase 2..."
330 A$ = INKEY$: IF A$ = "" THEN GOTO 330
340 PRINT ""
350 REM
360 REM Phase 2: Position and activate emitters
370 REM
380 PRINT "PHASE 2: Positioning Emitters"
390 PRINT "-----------------------------"
400 PRINT "Positioning POINT emitter at (200,400)..."
410 PARTICLE MOVE 1, 200, 400
420 WAIT 1000
430 PRINT "Positioning STAR emitter at (400,400)..."
440 PARTICLE MOVE 2, 400, 400
450 WAIT 1000
460 PRINT "Positioning CIRCLE emitter at (600,400)..."
470 PARTICLE MOVE 3, 600, 400
480 WAIT 1000
490 PRINT "Positioning RECT emitter at (300,200)..."
500 PARTICLE MOVE 4, 300, 200
510 WAIT 2000
520 PRINT ""
530 PRINT "All emitters active! Watch the different patterns."
540 PRINT "Press any key for Phase 3..."
550 A$ = INKEY$: IF A$ = "" THEN GOTO 550
560 PRINT ""
570 REM
580 REM Phase 3: Gravity demonstration
590 REM
600 PRINT "PHASE 3: Gravity Effects"
610 PRINT "------------------------"
620 PRINT "Adding very light gravity (15)..."
630 PARTICLE GRAVITY 15
640 WAIT 3000
650 PRINT "Light gravity (40)..."
660 PARTICLE GRAVITY 40
670 WAIT 3000
680 PRINT "Medium gravity (80)..."
690 PARTICLE GRAVITY 80
700 WAIT 3000
705 PRINT "Strong gravity (150)..."
706 PARTICLE GRAVITY 150
707 WAIT 3000
710 PRINT "Maximum gravity (255)!"
720 PARTICLE GRAVITY 255
730 WAIT 3000
740 PRINT "Removing gravity..."
750 PARTICLE GRAVITY 0
760 WAIT 2000
770 PRINT ""
780 PRINT "Press any key for Phase 4..."
790 A$ = INKEY$: IF A$ = "" THEN GOTO 790
800 PRINT ""
810 REM
820 REM Phase 4: Show/Hide demonstration
830 REM
840 PRINT "PHASE 4: Show/Hide Demo"
850 PRINT "-----------------------"
860 PRINT "Hiding emitters one by one..."
870 PRINT "Hiding POINT emitter..."
880 PARTICLE HIDE 1
890 WAIT 1500
900 PRINT "Hiding STAR emitter..."
910 PARTICLE HIDE 2
920 WAIT 1500
930 PRINT "Hiding CIRCLE emitter..."
940 PARTICLE HIDE 3
950 WAIT 1500
960 PRINT "Hiding RECT emitter..."
970 PARTICLE HIDE 4
980 WAIT 2000
990 PRINT ""
1000 PRINT "All emitters hidden! Showing them again..."
1010 PRINT "Showing POINT emitter..."
1020 PARTICLE SHOW 1
1030 WAIT 1000
1040 PRINT "Showing STAR emitter..."
1050 PARTICLE SHOW 2
1060 WAIT 1000
1070 PRINT "Showing CIRCLE emitter..."
1080 PARTICLE SHOW 3
1090 WAIT 1000
1100 PRINT "Showing RECT emitter..."
1110 PARTICLE SHOW 4
1120 WAIT 2000
1130 PRINT ""
1140 PRINT "Press any key for Phase 5..."
1150 A$ = INKEY$: IF A$ = "" THEN GOTO 1150
1160 PRINT ""
1170 REM
1180 REM Phase 5: Dynamic movement and effects
1190 REM
1200 PRINT "PHASE 5: Dynamic Animation"
1210 PRINT "--------------------------"
1220 PRINT "Moving emitters in patterns..."
1230 PARTICLE GRAVITY 60
1240 FOR T = 0 TO 360 STEP 10
1250 REM Calculate positions for circular motion
1260 X1 = 400 + 150 * SIN(T * 3.14159 / 180)
1270 Y1 = 300 + 100 * COS(T * 3.14159 / 180)
1280 X2 = 400 + 150 * SIN((T + 90) * 3.14159 / 180)
1290 Y2 = 300 + 100 * COS((T + 90) * 3.14159 / 180)
1300 X3 = 400 + 150 * SIN((T + 180) * 3.14159 / 180)
1310 Y3 = 300 + 100 * COS((T + 180) * 3.14159 / 180)
1320 X4 = 400 + 150 * SIN((T + 270) * 3.14159 / 180)
1330 Y4 = 300 + 100 * COS((T + 270) * 3.14159 / 180)
1340 REM Move emitters
1350 PARTICLE MOVE 1, X1, Y1
1360 PARTICLE MOVE 2, X2, Y2
1370 PARTICLE MOVE 3, X3, Y3
1380 PARTICLE MOVE 4, X4, Y4
1390 WAIT 100
1400 NEXT T
1410 PRINT "Animation complete!"
1420 WAIT 2000
1430 PRINT ""
1440 REM
1450 REM Phase 6: Speed and lifetime variations
1460 REM
1470 PRINT "PHASE 6: Parameter Variations"
1480 PRINT "-----------------------------"
1490 PRINT "Creating high-speed fountain (emitter 5)..."
1500 PARTICLE CREATE 5, point, 100, 200, 1.5
1510 PARTICLE MOVE 5, 400, 500
1520 WAIT 2000
1530 PRINT "Creating slow, long-lasting emitter (emitter 6)..."
1540 PARTICLE CREATE 6, circle, 15, 30, 8
1550 PARTICLE MOVE 6, 200, 300
1560 WAIT 3000
1570 PRINT "Creating burst emitter (emitter 7)..."
1580 PARTICLE CREATE 7, star, 200, 150, 0.8
1590 PARTICLE MOVE 7, 600, 300
1600 WAIT 3000
1610 PRINT ""
1620 PRINT "Multiple emitters with different behaviors!"
1630 PRINT "Press any key for finale..."
1640 A$ = INKEY$: IF A$ = "" THEN GOTO 1640
1650 PRINT ""
1660 REM
1670 REM Finale: Spectacular display
1680 REM
1690 PRINT "FINALE: Spectacular Display"
1700 PRINT "---------------------------"
1710 PRINT "Creating 8 emitters for grand finale..."
1720 PARTICLE CREATE 8, rect, 80, 120, 2
1730 PARTICLE MOVE 8, 500, 200
1740 PRINT "Gravity waves!"
1750 FOR G = 0 TO 255 STEP 15
1760 PARTICLE GRAVITY G
1770 WAIT 200
1780 NEXT G
1790 FOR G = 255 TO 0 STEP -15
1800 PARTICLE GRAVITY G
1810 WAIT 200
1820 NEXT G
1830 PRINT "Final burst!"
1840 PARTICLE GRAVITY 120
1850 WAIT 3000
1860 PRINT ""
1870 REM
1880 REM Cleanup
1890 REM
1900 PRINT "DEMONSTRATION COMPLETE"
1910 PRINT "======================"
1920 PRINT ""
1930 PRINT "Cleaning up - hiding all emitters..."
1940 FOR I = 1 TO 8
1950 PARTICLE HIDE I
1960 NEXT I
1970 PARTICLE GRAVITY 0
1980 PRINT ""
1990 PRINT "PARTICLE System Features Demonstrated:"
2000 PRINT "- 4 emitter types: POINT, STAR, CIRCLE, RECT"
2010 PRINT "- Particles per second: 15-200 PPS"
2020 PRINT "- Speed control: 30-200 pixels/second"
2030 PRINT "- Lifetime control: 0.8-8.0 seconds"
2040 PRINT "- Gravity physics: 0-255 acceleration"
2050 PRINT "- Dynamic positioning and movement"
2060 PRINT "- Show/hide emitter control"
2070 PRINT "- Up to 16 simultaneous emitters"
2080 PRINT "- Automatic particle fading"
2090 PRINT ""
2100 PRINT "Demo finished! Try creating your own particle effects."
2110 PRINT ""
2120 PRINT "PARTICLE Command Reference:"
2130 PRINT "PARTICLE CREATE id,type,[pps],[speed],[lifetime]"
2140 PRINT "PARTICLE MOVE id,x,y"
2150 PRINT "PARTICLE SHOW id"
2160 PRINT "PARTICLE HIDE id"
2170 PRINT "PARTICLE GRAVITY value"
2180 END