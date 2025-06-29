' Cursor.bas - Demo für INKEY$ mit drehbarem Würfel
' Verwende die Pfeiltasten um den Würfel zu drehen
' ESC zum Beenden

10 REM Cursor-Demo mit drehbarem Würfel
20 CLS
30 PRINT "CURSOR DEMO - Drehbarer Würfel"
40 PRINT "==============================="
50 PRINT ""
60 PRINT "Steuerung:"
70 PRINT "  Pfeiltasten = Würfel drehen"
80 PRINT "  ESC = Beenden"
90 PRINT ""
100 PRINT "Druecke eine Taste zum Starten..."
110 WAIT 100

120 REM Warte auf Tastendruck zum Starten
130 IF INKEY$ = "" THEN GOTO 130

140 CLS
150 PRINT "Verwende die Pfeiltasten!"

200 REM Initialisierung
210 LET CUBE_X = 0.0
220 LET CUBE_Y = 0.0
230 LET CUBE_Z = -8.0
240 LET ROT_X = 0.0
250 LET ROT_Y = 0.0
260 LET ROT_Z = 0.0
270 LET CUBE_SIZE = 2.0
280 LET CUBE_COLOR = 10
290 LET ROT_SPEED = 5.0
295 LET FRAME = 0

300 REM Hauptschleife (unendlich)
305 LET FRAME = FRAME + 1
310 REM Tasteneingabe prüfen
315 LET KEY$ = INKEY$
320 IF KEY$ = KEYESC THEN GOTO 900
330 IF KEY$ = KEYCURUP THEN LET ROT_X = ROT_X - ROT_SPEED
340 IF KEY$ = KEYCURDOWN THEN LET ROT_X = ROT_X + ROT_SPEED
350 IF KEY$ = KEYCURLEFT THEN LET ROT_Y = ROT_Y - ROT_SPEED
360 IF KEY$ = KEYCURRIGHT THEN LET ROT_Y = ROT_Y + ROT_SPEED

380 REM Rotation normalisieren
390 IF ROT_X >= 360 THEN LET ROT_X = ROT_X - 360
400 IF ROT_X < 0 THEN LET ROT_X = ROT_X + 360
410 IF ROT_Y >= 360 THEN LET ROT_Y = ROT_Y - 360
420 IF ROT_Y < 0 THEN LET ROT_Y = ROT_Y + 360

430 REM Würfel zeichnen
440 VECTOR 1, "cube", CUBE_X, CUBE_Y, CUBE_Z, ROT_X, ROT_Y, ROT_Z, CUBE_SIZE, CUBE_COLOR

450 REM Frame-Info (alle 120 Frames)
460 IF FRAME MOD 120 = 0 THEN PRINT "Frame: "; FRAME; " | Rotation X:"; INT(ROT_X); " Y:"; INT(ROT_Y)

470 REM Kurze Pause für flüssige Animation
480 WAIT 50

490 REM Zurück zur Hauptschleife
500 GOTO 305

900 REM Programmende
910 CLS
920 PRINT "Demo beendet!"
930 PRINT "Danke fuers Testen!"
940 END
