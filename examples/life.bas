10 REM Conway's Game of Life - 2D Array Version
20 PRINT "CONWAY'S GAME OF LIFE (2D Arrays)"
30 PRINT "================================="
40 PRINT "Using true 2D arrays..."
50 CLS

100 REM Spielfeld-Parameter
110 W = 39: H = 29: REM Spielfeld-Größe (0-39, 0-29)
120 CW = 16: CH = 16: REM Zell-Größe in Pixeln
130 GENERATIONS = 200: REM Anzahl Generationen

200 REM 2D-Arrays initialisieren
210 DIM FIELD(39,29): REM Aktuelles Spielfeld als 2D-Array
220 DIM NEWFIELD(39,29): REM Nächste Generation als 2D-Array

300 REM Spielfeld leeren
310 FOR X = 0 TO W
320   FOR Y = 0 TO H
330     FIELD(X,Y) = 0
340     NEWFIELD(X,Y) = 0
350   NEXT Y
360 NEXT X

400 REM Startmuster: Glider und andere Muster
410 GOSUB 1000

500 REM Hauptspiel-Schleife
510 FOR GEN = 1 TO GENERATIONS
520   GOSUB 2000: REM Spielfeld zeichnen
530   GOSUB 3000: REM Nächste Generation berechnen
540   GOSUB 4000: REM Felder tauschen
550   IF GEN MOD 10 = 0 THEN LOCATE 1,1: PRINT "Generation:", GEN
560   REM Kleine Pause für Animation
570   WAIT 1
580 NEXT GEN
590 PRINT "Simulation beendet!"
600 END

1000 REM Startmuster setzen (2D-Array-Version)
1010 REM Glider-Muster
1020 X = 5: Y = 5
1030 FIELD(X+1,Y) = 1
1040 FIELD(X+2,Y+1) = 1
1050 FIELD(X,Y+2) = 1
1060 FIELD(X+1,Y+2) = 1
1070 FIELD(X+2,Y+2) = 1

1100 REM Blinker-Muster (2D)
1110 X = 15: Y = 10
1120 FIELD(X,Y) = 1
1130 FIELD(X+1,Y) = 1
1140 FIELD(X+2,Y) = 1

1200 REM Block-Muster (Still Life) (2D)
1210 X = 25: Y = 8
1220 FIELD(X,Y) = 1
1230 FIELD(X+1,Y) = 1
1240 FIELD(X,Y+1) = 1
1250 FIELD(X+1,Y+1) = 1

1300 REM Toad-Muster (Oszillator) (2D)
1310 X = 10: Y = 20
1320 FIELD(X+1,Y) = 1
1330 FIELD(X+2,Y) = 1
1340 FIELD(X+3,Y) = 1
1350 FIELD(X,Y+1) = 1
1360 FIELD(X+1,Y+1) = 1
1370 FIELD(X+2,Y+1) = 1

1500 RETURN

2000 REM Spielfeld zeichnen (2D-Array-Version)
2010 FOR Y = 0 TO H
2020   FOR X = 0 TO W
2030     IF FIELD(X,Y) = 1 THEN COLOR = 15 ELSE COLOR = 0
2040     RECT X * CW, Y * CH, CW, CH, COLOR, 1
2050   NEXT X
2060 NEXT Y
2070 RETURN

3000 REM Nächste Generation berechnen (2D-Array-Version)
3010 FOR Y = 1 TO H - 1
3020   FOR X = 1 TO W - 1
3030     REM Nachbarn zählen mit 2D-Arrays
3040     NEIGHBORS = 0
3050     REM Alle 8 Nachbarn prüfen
3060     REM Oben links
3070     IF FIELD(X-1,Y-1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3080     REM Oben
3090     IF FIELD(X,Y-1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3100     REM Oben rechts
3110     IF FIELD(X+1,Y-1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3120     REM Links
3130     IF FIELD(X-1,Y) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3140     REM Rechts
3150     IF FIELD(X+1,Y) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3160     REM Unten links
3170     IF FIELD(X-1,Y+1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3180     REM Unten
3190     IF FIELD(X,Y+1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3200     REM Unten rechts
3210     IF FIELD(X+1,Y+1) = 1 THEN NEIGHBORS = NEIGHBORS + 1
3220     
3230     REM Game of Life Regeln anwenden
3240     IF FIELD(X,Y) = 1 THEN GOSUB 3300 ELSE GOSUB 3400
3250   NEXT X
3260 NEXT Y
3270 RETURN

3300 REM Regeln für lebende Zellen
3310 IF NEIGHBORS < 2 THEN NEWFIELD(X,Y) = 0: RETURN: REM Stirbt
3320 IF NEIGHBORS = 2 OR NEIGHBORS = 3 THEN NEWFIELD(X,Y) = 1: RETURN: REM Überlebt
3330 IF NEIGHBORS > 3 THEN NEWFIELD(X,Y) = 0: RETURN: REM Stirbt
3340 RETURN

3400 REM Regeln für tote Zellen
3410 IF NEIGHBORS = 3 THEN NEWFIELD(X,Y) = 1: RETURN: REM Wird geboren
3420 NEWFIELD(X,Y) = 0: REM Bleibt tot
3430 RETURN

4000 REM Spielfelder tauschen (2D-Array-Version)
4010 FOR Y = 0 TO H
4020   FOR X = 0 TO W
4030     FIELD(X,Y) = NEWFIELD(X,Y)
4040     NEWFIELD(X,Y) = 0
4050   NEXT X
4060 NEXT Y
4070 RETURN

5000 REM Alternative Startmuster: R-Pentomino (2D)
5010 REM Berühmtes chaotisches Muster
5020 X = 20: Y = 15
5030 FIELD(X+1,Y) = 1
5040 FIELD(X+2,Y) = 1
5050 FIELD(X,Y+1) = 1
5060 FIELD(X+1,Y+1) = 1
5070 FIELD(X+1,Y+2) = 1
5080 RETURN

6000 REM Einfache Glider Gun (2D-Version)
6010 REM Erzeugt Glider
6020 X = 5: Y = 5
6030 REM Vereinfachtes Muster für kleineres Spielfeld
6040 FIELD(X,Y) = 1
6050 FIELD(X+1,Y) = 1
6060 FIELD(X,Y+1) = 1
6070 FIELD(X+1,Y+1) = 1
6080 FIELD(X,Y+10) = 1
6090 FIELD(X+1,Y+10) = 1
6100 FIELD(X+2,Y+10) = 1
6110 FIELD(X-1,Y+11) = 1
6120 FIELD(X+3,Y+11) = 1
6130 FIELD(X-2,Y+12) = 1
6140 FIELD(X+4,Y+12) = 1
6150 RETURN

7000 REM Zufälliges Startmuster (2D)
7010 FOR I = 1 TO 50
7020   X = INT(RND * (W - 2)) + 1
7030   Y = INT(RND * (H - 2)) + 1
7040   FIELD(X,Y) = 1
7050 NEXT I
7060 RETURN

8000 REM Spielfeld-Statistiken (2D)
8010 ALIVE = 0
8020 FOR Y = 0 TO H
8030   FOR X = 0 TO W
8040     IF FIELD(X,Y) = 1 THEN ALIVE = ALIVE + 1
8050   NEXT X
8060 NEXT Y
8070 PRINT "Lebende Zellen:", ALIVE
8080 RETURN
