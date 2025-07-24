10 REM IMAGE System Demo - TinyBASIC Graphics
20 REM Demonstrates all IMAGE commands: OPEN, SHOW, HIDE, ROTATE
30 REM Requires PNG files in examples/ directory
40 REM 
50 PRINT "IMAGE SYSTEM DEMONSTRATION"
60 PRINT "========================="
70 PRINT ""
80 PRINT "This demo shows all IMAGE commands:"
90 PRINT "- IMAGE OPEN: Load PNG files (convert to 16 green shades)"
100 PRINT "- IMAGE SHOW: Display at position with scaling"
110 PRINT "- IMAGE HIDE: Hide images from display"
120 PRINT "- IMAGE ROTATE: Rotate images in degrees"
130 PRINT ""
140 PRINT "Press any key to continue..."
150 A$ = INKEY$: IF A$ = "" THEN GOTO 150
160 PRINT ""
170 REM
180 REM Phase 1: Load multiple images
190 REM
200 PRINT "PHASE 1: Loading Images"
210 PRINT "-----------------------"
220 PRINT "Loading mcp.png to handle 1..."
230 IMAGE OPEN "mcp.png", 1
240 PRINT "Loading skynet.png to handle 2..."
250 IMAGE OPEN "skynet.png", 2
260 PRINT "Loading tiny.png to handle 3..."
270 IMAGE OPEN "tiny.png", 3
280 PRINT ""
290 PRINT "Images loaded! (Converted to 16 green shades)"
300 PRINT "Press any key for Phase 2..."
310 A$ = INKEY$: IF A$ = "" THEN GOTO 310
320 PRINT ""
330 REM
340 REM Phase 2: Show images at different positions
350 REM
360 PRINT "PHASE 2: Displaying Images"
370 PRINT "--------------------------"
380 PRINT "Showing image 1 at (50,50) quarter size..."
390 IMAGE SHOW 1, 50, 50, -2
400 WAIT 2000
410 PRINT "Showing image 2 at (200,50) quarter size..."
420 IMAGE SHOW 2, 200, 50, -2
430 WAIT 2000
440 PRINT "Showing image 3 at (350,50) quarter size..."
450 IMAGE SHOW 3, 350, 50, -2
460 WAIT 2000
470 PRINT ""
480 PRINT "All images displayed! Press any key for Phase 3..."
490 A$ = INKEY$: IF A$ = "" THEN GOTO 490
500 PRINT ""
510 REM
520 REM Phase 3: Rotation demonstration
530 REM
540 PRINT "PHASE 3: Rotation Demo"
550 PRINT "----------------------"
560 PRINT "Rotating image 1 through 360 degrees..."
570 FOR R = 0 TO 360 STEP 15
580 IMAGE ROTATE 1, R
590 WAIT 100
600 NEXT R
610 PRINT "Rotation complete!"
620 WAIT 1000
630 PRINT ""
640 PRINT "Rotating image 2 backwards..."
650 FOR R = 0 TO -180 STEP -10
660 IMAGE ROTATE 2, R
670 WAIT 80
680 NEXT R
690 PRINT "Reverse rotation complete!"
700 WAIT 1000
710 PRINT ""
720 PRINT "Press any key for Phase 4..."
730 A$ = INKEY$: IF A$ = "" THEN GOTO 730
740 PRINT ""
750 REM
760 REM Phase 4: Scaling demonstration
770 REM
780 PRINT "PHASE 4: Scaling Demo"
790 PRINT "---------------------"
800 PRINT "Demonstrating different scale values..."
810 PRINT "Scale -2 (quarter size - good for 1024x1024):"
820 IMAGE SHOW 1, 50, 50, -2
830 WAIT 1500
840 PRINT "Scale -1.5 (smaller):"
850 IMAGE SHOW 1, 50, 50, -1.5
860 WAIT 1500
870 PRINT "Scale -1 (half size):"
880 IMAGE SHOW 1, 50, 50, -1
890 WAIT 1500
900 PRINT "Scale -0.5 (larger):"
910 IMAGE SHOW 1, 50, 50, -0.5
920 WAIT 1500
930 PRINT "Scale 0 (original - may be too large!):"
940 IMAGE SHOW 1, 50, 50, 0
950 WAIT 1500
960 PRINT ""
970 PRINT "Fine-tuned scaling (0.1 increments):"
980 FOR S = 0 TO 1 STEP 0.1
990 IMAGE SHOW 3, 50, 50, S
1000 WAIT 200
1010 NEXT S
1020 PRINT "Fine scaling complete!"
1030 WAIT 1000
1040 PRINT ""
1050 PRINT "Press any key for Phase 5..."
1060 A$ = INKEY$: IF A$ = "" THEN GOTO 1060
1070 PRINT ""
1080 REM
1090 REM Phase 5: Hide/Show demonstration
1100 REM
1110 PRINT "PHASE 5: Hide/Show Demo"
1120 PRINT "-----------------------"
1130 PRINT "Blinking effect - hiding and showing images..."
1140 FOR B = 1 TO 10
1150 IMAGE HIDE 1
1160 IMAGE HIDE 2
1170 IMAGE HIDE 3
1180 WAIT 300
1190 IMAGE SHOW 1, 50, 50, -2
1200 IMAGE SHOW 2, 200, 50, -2
1210 IMAGE SHOW 3, 350, 50, -2
1220 WAIT 300
1230 NEXT B
1240 PRINT "Blinking complete!"
1250 WAIT 1000
1260 PRINT ""
1270 REM
1280 REM Phase 6: Complex animation
1290 REM
1300 PRINT "PHASE 6: Complex Animation"
1310 PRINT "--------------------------"
1320 PRINT "Animated orbit with rotation and scaling..."
1330 FOR A = 0 TO 720 STEP 5
1340 REM Calculate circular motion
1350 X = 400 + 200 * SIN(A * 3.14159 / 180)
1360 Y = 300 + 150 * COS(A * 3.14159 / 180)
1370 REM Vary scale based on position
1380 S = SIN(A * 3.14159 / 90) * 0.5
1390 IMAGE SHOW 1, X, Y, S
1400 IMAGE ROTATE 1, A / 2
1410 WAIT 50
1420 NEXT A
1430 PRINT "Animation complete!"
1440 WAIT 1000
1450 PRINT ""
1460 REM
1470 REM Cleanup and summary
1480 REM
1490 PRINT "DEMONSTRATION COMPLETE"
1500 PRINT "======================"
1510 PRINT ""
1520 PRINT "IMAGE System Features Demonstrated:"
1530 PRINT "- Load PNG files and convert to 16 green shades"
1540 PRINT "- Position images at any x,y coordinates"
1550 PRINT "- Scale from -2.0 to 2.0 (0.1 step precision)"
1560 PRINT "- Rotate -360 to 360 degrees"
1570 PRINT "- Hide/show images dynamically"
1580 PRINT "- Handle system supports 1-8 simultaneous images"
1590 PRINT "- Maximum image size: 2048x2048 pixels"
1600 PRINT ""
1610 PRINT "Hiding all images..."
1620 IMAGE HIDE 1
1630 IMAGE HIDE 2
1640 IMAGE HIDE 3
1650 PRINT ""
1660 PRINT "Demo finished! Try creating your own image programs."
1670 PRINT ""
1680 PRINT "IMAGE Command Reference:"
1690 PRINT "IMAGE OPEN 'filename.png', handle"
1700 PRINT "IMAGE SHOW handle, x, y, [scale]"
1710 PRINT "IMAGE HIDE handle"
1720 PRINT "IMAGE ROTATE handle, degrees"
1730 END