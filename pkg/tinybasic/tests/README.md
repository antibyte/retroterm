# TinyBASIC Tests

Diese Test-Suite überprüft die Funktionalität des TinyBASIC-Interpreters im TinyOS-System.

## Test-Struktur

### `expression_test.go`
Basis-Tests für die fundamentalen BASIC-Funktionen:
- **TestBasicArithmetic**: Grundrechenarten (+, -, *, /, Klammern)
- **TestStringOperations**: String-Literale und -verarbeitung
- **TestVariableOperations**: Variablen-Zuweisung und -zugriff (LET, numerisch und String)
- **TestErrorHandling**: Fehlerbehandlung (z.B. Division durch Null)
- **TestProgramExecution**: LIST-Befehl und Programm-Management
- **TestBasicFunctions**: Mathematische Funktionen (ABS, RND)
- **TestLoopCommands**: FOR-NEXT Schleifen
- **TestConditionalCommands**: IF-THEN Bedingungen
- **TestInputOutput**: PRINT-Befehle und Ausgabe-Formatierung
- **TestProgramManagement**: NEW, LIST und andere Programmverwaltung

### `advanced_test.go`
Erweiterte Tests für komplexere Features:
- **TestFileOperations**: SAVE/LOAD-Funktionalität (testet Fehlerbehandlung ohne VFS)
- **TestBasicMath**: Mathematische Funktionen (SQR, SIN, COS, INT)
- **TestStringFunctions**: String-Funktionen (LEN)
- **TestDataReadCommands**: DATA/READ-Statements (übersprungen - benötigt bessere async Ausgabe-Behandlung)
- **TestGosubReturn**: GOSUB/RETURN-Funktionalität (übersprungen - benötigt bessere async Ausgabe-Behandlung)

### `testutil/`
Verzeichnis für Test-Hilfsfunktionen und Mock-Objekte (für zukünftige Erweiterungen).

## Test-Hilfsfunktionen

### `NewTestBasic()`
Erstellt eine TinyBASIC-Instanz für Tests ohne externe Abhängigkeiten.

### `executeAndGetOutput(basic, command)`
Führt einen BASIC-Befehl aus und gibt die erste Text-Nachricht zurück.

### `executeAndGetAllOutput(basic, command)`  
Sammelt alle Text-Ausgaben aus einem BASIC-Befehl.

### `runProgramAndGetOutput(basic)`
Führt ein geladenes BASIC-Programm aus und sammelt die Ausgabe aus dem OutputChan.

## Tests ausführen

```powershell
# Alle TinyBASIC Tests
go test ./pkg/tinybasic/tests/...

# Mit detaillierter Ausgabe
go test -v ./pkg/tinybasic/tests/...

# Spezifische Tests
go test -v ./pkg/tinybasic/tests/... -run="TestBasicArithmetic"

# Mit Timeout (für Tests mit RUN-Befehlen)
go test -v ./pkg/tinybasic/tests/... -timeout=10s
```

## Aktuelle Test-Abdeckung

✅ **Funktioniert:**
- Grundrechenarten und Ausdrücke
- String-Verarbeitung
- Variablen (numerisch und String)
- Mathematische Funktionen (ABS, RND, SQR, SIN, COS, INT, LEN)
- Fehlerbehandlung (Division durch Null, etc.)
- FOR-NEXT Schleifen  
- IF-THEN Bedingungen
- PRINT-Befehle mit verschiedenen Formaten
- Programm-Management (NEW, LIST)
- Datei-Operationen (Fehlerbehandlung ohne VFS)

⏸️ **Übersprungen (benötigt Verbesserung):**
- DATA/READ-Befehle (async Ausgabe-Sammlung)
- GOSUB/RETURN (async Ausgabe-Sammlung)

## Bekannte Einschränkungen

1. **Async Output**: Tests für RUN-Befehle, die Ausgaben über `OutputChan` senden, sind komplex zu implementieren. Einige Tests wurden übersprungen und benötigen eine bessere async Ausgabe-Sammel-Infrastruktur.

2. **Debug-Ausgaben**: Die Tests erzeugen Debug-Ausgaben, die in Produktionsumgebung deaktiviert werden sollten.

3. **VFS-Abhängigkeit**: Datei-Tests können nur Fehlerbehandlung testen, da keine VFS-Instanz verfügbar ist.

## Zukünftige Erweiterungen

- Bessere async Output-Sammlung für RUN-Tests
- Tests für Grafik-Befehle (DRAW, SPRITE, etc.)
- Tests für Sound-Befehle (SOUND, PLAY, etc.)
- Tests für INPUT-Befehle
- Integration Tests mit echter VFS
- Performance- und Speicher-Tests
- Mehr Error-Case Testing
- Tests für Arrays und erweiterte Datenstrukturen
