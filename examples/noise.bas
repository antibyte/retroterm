10 REM Noise Demo
20 PRINT "NOISE DEMO"
30 PRINT "------------"
40 PRINT ""
50 PRINT "Test 1: Einzelner Noise-Effekt"
60 PRINT "Pitch=128, Attack=50, Decay=100"
70 NOISE 128, 50, 100
80 WAIT 2000 : REM 2 Sekunden warten
90 PRINT ""
100 PRINT "Test 2: Verschiedene Pitch-Werte"
110 PRINT "Attack=30, Decay=80"
120 FOR P = 0 TO 255 STEP 50
130   PRINT "Pitch = "; P
140   NOISE P, 30, 80
150   WAIT 500
160 NEXT P
170 WAIT 1000
180 PRINT ""
190 PRINT "Test 3: Verschiedene Attack-Werte"
200 PRINT "Pitch=100, Decay=120"
210 FOR A = 0 TO 255 STEP 50
220   PRINT "Attack = "; A
230   NOISE 100, A, 120
240   WAIT 700
250 NEXT A
260 WAIT 1000
270 PRINT ""
280 PRINT "Test 4: Verschiedene Decay-Werte"
290 PRINT "Pitch=150, Attack=40"
300 FOR D = 0 TO 255 STEP 50
310   PRINT "Decay = "; D
320   NOISE 150, 40, D
330   WAIT 700
340 NEXT D
350 WAIT 1000
360 PRINT ""
370 PRINT "Test 5: Rate Limiting Test"
380 PRINT "Versuche, 15 NOISE-Befehle schnell hintereinander auszuführen."
390 PRINT "(Erwartung: Die ersten 10 sollten hörbar sein, die nächsten 5 nicht oder mit Fehler)"
400 FOR I = 1 TO 15
410   PRINT "Noise Versuch "; I
420   NOISE 100, 20, 50
430   REM Kurze Pause, aber nicht genug, um das Rate Limit für alle zu umgehen
440   REM Wenn jeder NOISE ~50ms dauert + 20ms WAIT, sind das 70ms.
450   REM 10 * 70ms = 700ms. Die ersten 10 sollten also innerhalb einer Sekunde kommen.
460   WAIT 20  REM Sehr kurze Pause
470 NEXT I
480 WAIT 2000
490 PRINT ""
500 PRINT "Test 6: Parameter-Grenzwerte"
510 PRINT "Pitch=0, Attack=0, Decay=0"
520 NOISE 0,0,0
530 WAIT 1000
540 PRINT "Pitch=255, Attack=255, Decay=255"
550 NOISE 255,255,255
560 WAIT 2000
570 PRINT ""
580 PRINT "Noise Demo Ende."
590 END
