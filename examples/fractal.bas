10 REM Mandelbrot-Fractal mit Zoom-Animation
20 PRINT "MANDELBROT FRACTAL"
30 PRINT "=================="
40 PRINT "Zoom-Animation started..."
50 CLS

100 REM Bildschirmauflösung und Parameter
110 W = 640: H = 480
120 CX = -0.5: CY = 0
130 MAXITER = 50
140 ZOOM = 1

200 REM Zoom-Animation Hauptschleife
210 FOR FRAME = 1 TO 50
220   GOSUB 2000
230   ZOOM = ZOOM * 1.1
240   LOCATE 1, 1: PRINT "Zoom Level:", ZOOM;"               "
250 NEXT FRAME
260 PRINT "Animation finished!"
270 END

1000 REM Mandelbrot-Berechnung und Zeichnung
1010 REM Parameter für aktuellen Zoom-Level
1020 SCALE = 3 / ZOOM
1030 XMIN = CX - SCALE
1040 XMAX = CX + SCALE  
1050 YMIN = CY - SCALE * H / W
1060 YMAX = CY + SCALE * H / W

1100 REM Raster durch das Bild gehen (jeder 4. Pixel für Performance)
1110 FOR PY = 0 TO H - 1 STEP 4
1120   FOR PX = 0 TO W - 1 STEP 4
1130     REM Pixel-Koordinaten zu komplexer Zahl konvertieren
1140     X0 = XMIN + PX * (XMAX - XMIN) / W
1150     Y0 = YMIN + PY * (YMAX - YMIN) / H
1160     
1170     REM Mandelbrot-Iteration
1180     X = 0: Y = 0: ITER = 0
1190     FOR I = 1 TO MAXITER
1200       REM z = z² + c berechnen
1210       XT = X * X - Y * Y + X0
1220       Y = 2 * X * Y + Y0
1230       X = XT
1240       
1250       REM Escape-Bedingung prüfen
1260       IF X * X + Y * Y > 4 THEN ITER = I: I = MAXITER + 1
1270     NEXT I
1280     
1290     REM Farbe basierend auf Iterationen bestimmen
1300     IF ITER = 0 THEN LET COLOR = 0
1310     IF ITER > 0 AND ITER < MAXITER THEN LET COLOR = 1 + (ITER MOD 15)
1315     IF ITER = MAXITER THEN LET COLOR = 0
1320     REM 2x2 Block zeichnen für bessere Sichtbarkeit
1330     RECT PX, PY, 4, 4, COLOR, 1
1350   NEXT PX
1360 NEXT PY
1370 RETURN

2000 REM Mandelbrot-Berechnung und Zeichnung
2010 REM Parameter für aktuellen Zoom-Level
2020 SCALE = 3 / ZOOM
2030 XMIN = CX - SCALE
2040 XMAX = CX + SCALE  
2050 YMIN = CY - SCALE * H / W
2060 YMAX = CY + SCALE * H / W

2100 REM Raster durch das Bild gehen (jeder 4. Pixel für Performance)
2110 FOR PY = 0 TO H - 1 STEP 4
2120   FOR PX = 0 TO W - 1 STEP 4
2130     REM Pixel-Koordinaten zu komplexer Zahl konvertieren
2140     X0 = XMIN + PX * (XMAX - XMIN) / W
2150     Y0 = YMIN + PY * (YMAX - YMIN) / H
2160     
2170     REM Mandelbrot-Iteration
2180     X = 0: Y = 0: ITER = 0
2190     FOR I = 1 TO MAXITER
2200       REM z = z² + c berechnen
2210       XT = X * X - Y * Y + X0
2220       Y = 2 * X * Y + Y0
2230       X = XT
2240       
2250       REM Escape-Bedingung prüfen
2260       IF X * X + Y * Y > 4 THEN ITER = I: I = MAXITER + 1
2270     NEXT I
2280     
2290     REM Farbe basierend auf Iterationen bestimmen
2300     IF ITER = 0 THEN LET COLOR = 0
2310     IF ITER > 0 AND ITER < MAXITER THEN LET COLOR = 1 + (ITER MOD 15)
2315     IF ITER = MAXITER THEN LET COLOR = 0
2320     REM 2x2 Block zeichnen für bessere Sichtbarkeit
2330     RECT PX, PY, 4, 4, COLOR, 1
2350   NEXT PX
2360 NEXT PY
2370 RETURN

2400 REM Alternative: Interessante Mandelbrot-Punkte
2410 REM Diese können für verschiedene Zoom-Zentren verwendet werden
2420 REM CX = -0.74529, CY = 0.11307    : REM Spiralen
2430 REM CX = -0.1, CY = 0.8            : REM Dendrit-Bereich  
2440 REM CX = -0.75, CY = 0.1           : REM Seepferdchen-Tal
2450 REM CX = 0.3, CY = 0.5             : REM Interessante Strukturen
2460 RETURN

3000 REM Zoom-Animation mit mehreren interessanten Punkten
3010 DATA -0.74529, 0.11307
3020 DATA -0.1, 0.8  
3030 DATA -0.75, 0.1
3040 DATA 0.3, 0.5
3050 DATA -0.5, 0
3060 DATA -1, 0
3070 DATA 999, 999: REM End marker
3080 RETURN

4000 REM Erweiterte Animation mit Punkt-Wechsel
4010 RESTORE 3010
4020 FOR SEQUENCE = 1 TO 6
4030   READ CX, CY
4040   IF CX = 999 THEN RETURN: REM Ende erreicht
4050   ZOOM = 1
4060   
4070   REM 200 Frames pro Punkt zoomen
4080   FOR ZOOMFRAMES = 1 TO 200
4090     GOSUB 1000: REM Mandelbrot zeichnen
4100     ZOOM = ZOOM * 1.03
4110     FOR P = 1 TO 50: NEXT P: REM Kleine Pause
4120   NEXT ZOOMFRAMES
4130   
4140   CLS: REM Bildschirm löschen für nächsten Punkt
4150   PRINT "Nächster interessanter Punkt..."
4160   FOR P = 1 TO 500: NEXT P: REM Pause zwischen Punkten
4170 NEXT SEQUENCE
4180 RETURN

5000 REM Hochauflösende Version (langsamer aber detaillierter)
5010 FOR PY = 0 TO H - 1 STEP 2
5020   FOR PX = 0 TO W - 1 STEP 2
5030     X0 = XMIN + PX * (XMAX - XMIN) / W
5040     Y0 = YMIN + PY * (YMAX - YMIN) / H
5050     
5060     X = 0: Y = 0: ITER = 0
5070     FOR I = 1 TO MAXITER
5080       XT = X * X - Y * Y + X0
5090       Y = 2 * X * Y + Y0
5100       X = XT
5110       IF X * X + Y * Y > 4 THEN ITER = I: I = MAXITER + 1
5120     NEXT I
5130     
5140     IF ITER = 0 THEN LET COLOR = 0
5150     IF ITER > 0 THEN LET COLOR = ITER MOD 16
5160     RECT PX, PY, 2, 2, COLOR, 1
5170   NEXT PX
5180 NEXT PY
5190 RETURN

6000 REM Farbverlauf-Version mit schöneren Farben
6010 FOR PY = 0 TO H - 1 STEP 3
6020   FOR PX = 0 TO W - 1 STEP 3
6030     X0 = XMIN + PX * (XMAX - XMIN) / W
6040     Y0 = YMIN + PY * (YMAX - YMIN) / H
6050     
6060     X = 0: Y = 0: ITER = 0
6070     FOR I = 1 TO MAXITER
6080       XT = X * X - Y * Y + X0
6090       Y = 2 * X * Y + Y0
6100       X = XT
6110       IF X * X + Y * Y > 4 THEN ITER = I: I = MAXITER + 1
6120     NEXT I
6130     
6140     REM Erweiterte Farbberechnung
6150     IF ITER = 0 THEN LET COLOR = 0
6160     IF ITER > 0 AND ITER < 10 THEN LET COLOR = 1 + ITER
6170     IF ITER >= 10 AND ITER < 30 THEN LET COLOR = 5 + (ITER - 10) / 3
6180     IF ITER >= 30 THEN LET COLOR = 12 + (ITER MOD 4)
6190     RECT PX, PY, 3, 3, COLOR, 1
6200   NEXT PX
6210 NEXT PY
6220 RETURN

9000 REM Alternative Startpunkte für verschiedene Effekte
9010 REM Um das Programm zu ändern, ersetze Zeile 220 mit:
9020 REM GOSUB 4000 für Punkt-Wechsel-Animation
9030 REM GOSUB 5000 für höhere Auflösung
9040 REM GOSUB 6000 für schönere Farben
9050 END
