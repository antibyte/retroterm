10 REM Test für CIRCLE Sichtbarkeit
20 GRAPHICS
30 FILL 0, 0, "#000000"
40 
50 REM Einfacher CIRCLE ohne Physics
60 CIRCLE 100, 100, 30, 4, 1
70 PRINT "Statischer Kreis bei (100,100) erstellt"
80 
90 REM CIRCLE mit Physics
100 CIRCLE 300, 100, 25, 2, 1
110 PHYSICS CIRCLE 300, 100, 25, 1
120 PHYSICS WORLD 0, -10
130 PHYSICS SCALE 100
140 PRINT "Physics-Kreis bei (300,100) erstellt"
150 
160 REM Starte Physics
170 PHYSICS START
180 PRINT "Physics gestartet - Kreis sollte fallen"
190 
200 END