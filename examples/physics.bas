10 REM 2D Physics Demo mit Sprites und Grafiken
20 CLS
30 PRINT "2D PHYSICS DEMO"
40 PRINT "==============="
50 PRINT "Sprites, Circles, Lines, Rectangles mit Physics"

60 REM Initialisiere Physics für 2D (Pixel-Koordinaten)
70 PHYSICS WORLD 0, 300
80 PHYSICS SCALE 30
90 PRINT "Physics Welt für 2D initialisiert"

100 REM Erstelle Boden und Wände (Pixel-Koordinaten)
110 PHYSICS FLOOR 0, 450, 640, 450
120 PHYSICS WALL 0, 0, 0, 480
130 PHYSICS WALL 640, 0, 640, 480
140 PRINT "Boden und Wände für 2D erstellt"

150 REM Erstelle 2D-Grafiken mit Physics
160 REM 1. Sprite (falls verfügbar)
170 REM SPRITE 1, "11111100,10000010,10000010,10000010,10000010,10000010,10000010,11111100"
180 REM SPRITE UPDATE 1, 1, 100, 50, 0, 1
190 REM PHYSICS CIRCLE 100, 50, 16, 11
200 REM PHYSICS LINK 11, 1, "SPRITE"

210 REM 2. Grafik-Kreise (verwende gleiche IDs für CIRCLE und PHYSICS)
220 CIRCLE 200, 100, 20, 4, 1
230 PHYSICS CIRCLE 200, 100, 20, 1
240 PRINT "Roter Kreis mit Physics erstellt (ID 1)"

250 CIRCLE 300, 80, 15, 2, 1
260 PHYSICS CIRCLE 300, 80, 15, 2
270 PRINT "Grüner Kreis mit Physics erstellt (ID 2)"

280 CIRCLE 400, 120, 25, 6, 1
290 PHYSICS CIRCLE 400, 120, 25, 3
300 PRINT "Gelber Kreis mit Physics erstellt (ID 3)"

310 REM 3. Grafik-Rechtecke (statisch als Plattformen)
320 RECT 150, 300, 100, 20, 8, 1
330 PHYSICS RECT 150, 300, 100, 20
340 PRINT "Plattform 1 erstellt"

350 RECT 350, 250, 120, 15, 9, 1
360 PHYSICS RECT 350, 250, 120, 15
370 PRINT "Plattform 2 erstellt"

380 REM Setze Physics-Eigenschaften für die Kreise
390 PHYSICS BOUNCE 1, 0.8
400 PHYSICS FRICTION 1, 0.3
410 PHYSICS VELOCITY 1, 50, -100

420 PHYSICS BOUNCE 2, 0.9
430 PHYSICS FRICTION 2, 0.2
440 PHYSICS VELOCITY 2, -30, -150

450 PHYSICS BOUNCE 3, 0.7
460 PHYSICS FRICTION 3, 0.4
470 PHYSICS VELOCITY 3, 20, -80

480 PRINT "Physics-Eigenschaften gesetzt"

490 REM Starte Physics Simulation
500 PHYSICS AUTO ON
510 PRINT "2D Physics aktiviert!"
520 PRINT "Kreise fallen und springen auf Plattformen"

530 REM Simulation für 10 Sekunden
540 FOR I = 1 TO 1000
550   WAIT 30
560   
570   REM Alle 200 Frames etwas Chaos
580   IF I MOD 200 = 0 THEN GOSUB 700
590 NEXT I

600 PHYSICS AUTO OFF
610 PRINT "Demo beendet"
620 END

700 REM Chaos-Subroutine
710 LET LUCKY = RND(3) + 1
720 LET IMPULSE_X = (RND(400) - 200)
730 LET IMPULSE_Y = -(RND(200) + 100)
740 PHYSICS FORCE LUCKY, IMPULSE_X, IMPULSE_Y
750 PRINT "Chaos! Objekt "; LUCKY; " bekommt Impuls!"
760 RETURN