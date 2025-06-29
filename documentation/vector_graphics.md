# Vector-Grafik-Befehle für TinyBASIC

Diese Dokumentation beschreibt die neu implementierten 3D-Vektor-Grafikbefehle für TinyBASIC, mit denen einfache 3D-Objekte wie Würfel, Pyramiden und Kugeln erstellt und manipuliert werden können.

## Verfügbare Befehle

### VECTOR

Erstellt oder aktualisiert ein 3D-Vektorobjekt mit der angegebenen ID.

```basic
VECTOR id, shape, x, y, z, rotX, rotY, rotZ, scale, [brightness]
```

Parameter:
- `id`: Numerische ID des Vektorobjekts (0-255)
- `shape`: Form des Objekts als String ("cube", "pyramid" oder "sphere")
- `x`, `y`, `z`: Position des Objekts im 3D-Raum
- `rotX`, `rotY`, `rotZ`: Rotation des Objekts in Grad
- `scale`: Skalierungsfaktor (einheitlich für alle Achsen)
- `brightness`: (optional) Helligkeit/Farbe des Objekts (0-15)

Beispiel:
```basic
VECTOR 1, "sphere", 0, 0, -10, 0, 30, 0, 2.0, 15
```

### VECTOR.SCALE

Aktualisiert die Skalierung eines Vektorobjekts mit unterschiedlichen Werten für jede Achse.

```basic
VECTOR.SCALE id, scaleX, scaleY, scaleZ, [brightness]
```

Parameter:
- `id`: Numerische ID des zu aktualisierenden Vektorobjekts
- `scaleX`, `scaleY`, `scaleZ`: Skalierungsfaktoren für die jeweiligen Achsen
- `brightness`: (optional) Helligkeit/Farbe des Objekts (0-15)

Beispiel:
```basic
VECTOR.SCALE 1, 2.0, 1.0, 3.0, 10
```

### VECTOR.HIDE

Versteckt ein Vektorobjekt, ohne es zu löschen.

```basic
VECTOR.HIDE id
```

Parameter:
- `id`: Numerische ID des zu versteckenden Vektorobjekts

### VECTOR.SHOW

Zeigt ein zuvor verstecktes Vektorobjekt wieder an.

```basic
VECTOR.SHOW id
```

Parameter:
- `id`: Numerische ID des anzuzeigenden Vektorobjekts

## Beispielprogramme

In der Examples-Sammlung wurden die folgenden Beispielprogramme hinzugefügt:

1. `boingball.bas` - Einfache Animation einer rotierenden Kugel
2. `boingball_enhanced.bas` - Verbesserte Version mit Hintergrundgitter
3. `boingball_pro.bas` - Professionelle Version mit Schattenwurf und Physik

## Tipps für die 3D-Programmierung

- Die Z-Koordinate bestimmt die Entfernung vom Betrachter. Negative Werte gehen in den Bildschirm hinein.
- Die Rotation wird in Grad angegeben und intern in Radiant umgerechnet.
- Objekte mit einer größeren Z-Koordinate (näher am Betrachter) überdecken Objekte mit kleinerer Z-Koordinate.
- Die maximale Anzahl von Vektorobjekten ist auf 255 begrenzt.

## Technische Details

Die 3D-Rendering-Engine verwendet Three.js im Browser und zeigt die Vektorobjekte als Drahtmodelle an. Die Kommunikation zwischen dem TinyBASIC-Backend und dem Frontend erfolgt über das gleiche Nachrichtensystem wie für 2D-Grafiken, jedoch mit einem speziellen Befehlstyp "UPDATE_VECTOR".

Alle Vektorgrafik-Befehle sind in der Datei `vector_commands.go` implementiert.
