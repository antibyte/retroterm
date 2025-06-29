## Retro-Terminal-Projekt Dokumentation für KI-Agenten

## Projektübersicht
Dieses Projekt ist eine Web-Anwendung die ein cooles 80er Jahre Retro-Terminal im Browser simuliert.
Die Darstellung ist ein Grün-Monitor, es werden keine Farben unterstützt aber 16 Helligkeitstufen für Grafikbefehle.
Es implemetiert im Backend im Golang ein virtuelles Multiuser-Betriebsystem namens TinyOs.
Es gibt ein Session basiertes Authentifikationssystem.
Zudem existiert ein Basic Interpreter names TinyBasic der von TinyOs aus gestartet werden kann durch Eingabe des Befehls "basic".
Ein weiteres feature ist ein KI-Chat der sich mit der Deepseek API verbindet und den user mit Deepseek chatten lässt.

## Projektziele:

## TinyOs
TinyOs simuliert ein Retro Betriebsystem das sich an Betriebsysteme aus dieser Epoche anlehnt und Befehle für die Arbeit mit
Dateien und Verzeichnissen bereitstellt, Dateien anzeigen kann und einige weitere nützliche Befehle bereitstellt.
Als Programme stehen vorerst "basic" und "chat" zur Verfügung.
Nutzer können sich mit den Befehlen "register <username> <Password>" und "login <username> <password> registrieren und anmelden.
Wenn von der selben IP Adresse aus versucht wird mehr als einen benutzernamen innerhalb 24 stunden zu registrieren, wird dies unterbunden.
Dazu muss jeder Anmeldeversuch in der Datenbank mit IP gespeichert werden.
hat sich der user angemeldet, wird ein JWT cookie erstellt mit dem sich der user für 24 stunden nicht erneut anmelden muss.
Mit dem befehl logout wird der user abgemeldet und sein cookie ungültig.
Benutzername und verschlüsseltes Passwort werden in der sqlite Datenbank des Projektes gespeichert (tinyos.db).
Dort werden auch andere wichtige Informationen zum Nutzer erfasst (letzer login, ist eingelogged, ist gebannt etc)
Wird ein Nutzer angelegt, wird auch ein virtuelles Dateisystem für ihn in der Datenbank erstellt.
Ein regulärer Nutzer hat ein Disk-Quota von 10MB, alle Schreibvorgänge im virtuellen Dateisystem prüfen dies.
Es muss unbedingt darauf geachtet werden, dass das VFS auf Windows und Linux funktioniert (hier gab es bereits Probleme mit den unterschiedlich Pfadformaten)
Nicht angemeldete Gast Nutzer haben ein temporäres VFS im RAM. Ihre Daten werden nicht in der Datenbank gespeichert !


## TinyBasic
Der Basic Interpreter implementiert ein klassisches, zeilenbasiertes Basic das allerdings sehr umfangreich und komfortabel sein soll.
Es stehen neben den elementaren Basic-Befehlen auch Befehle für Sound, Grafik und Sprachausgabe zu Verfügung.
Der Parser sollte auch ungewöhnlich formatierte Zeilen erkennen können (z.b. doppelte leerzeichen) solange die grundsätzliche syntax gültig bleibt.
In dem Fall wird die Eingabe einheitlich formatiert gespeichert.
Dateien könne mit "load" und "save" im virtuellen Dateisystems gesichert und geladen werden. Wenn nicht vorhanden wird immer die endung .bas angehängt.
der Befehl dir listet die vorhandenen Dateien mit der Endung .bas auf.
Ist ein Benutzer nicht angemeldet kann er load und save nicht nutzen und erhält beim Versuch eine Fehlermeldung "Log in to use the filesystem"
Ins Dateisystem aller Anwender werden bei dessen Erstellung auch einige Beispielprogramme zur Demonstration der Möglichkeiten kopiert.
Diese Beispielprogramme liegen im Anwendungsverzeichnis im Unterordner ./examples
Meldet sich ein Nutzer an, wird geprüft ob alle Beispielprogramme in seinem vfs sind, wenn nicht werden diese hinein kopiert.
Werden dem Basic neue Befehle hinzugefügt, werden auch diese Beispiele aktualisiert.
Es wurde ein stark verbessertes handling für die basic fehlerbehandlung in errors.go implemetiert. Dies muss überall berücksichtigt und verwendet werden !
die tinybasic dateistructur ist wie folgt:
constants.go Konstanten
errors.go zentrale fehlerbehndlung für basic befehle
file_commands.go implementierung basic datei befehle
filesystem.go dateisystem methoden
flow_commands.go basic befehle für Programmablauf
gfx_commands.go basic befehle für Grafikfunktionen
help.go basic Hilfesystem
interface.go kommunikation mit anderen programmteilen
io_commands.go I/O Basic Befehle
media_commands.go Sound und Sprachausgabe Basic Befehle
parser.go der Basic Parser
programm_commands.go Befehle zur steuerung von Basic
tinybasic.go der Interpreter
utilities.go Hilfsfunktionen
var_commands.go Basic Befehle für den Umgang mit Variablen

## Frontend
Das Frontend übernimmt die realistische Darstellung des Retro-Terminal unter Verwendung von three.js Shadern.
Es ist über einen Websocket mit dem Backend verbunden und erhält über ein Protokoll (siehe messages.go) Befehle um Text auszugeben,
Grafikbefehle zu realisieren, Töne und Sprache auszugeben. Die Sprachausgabe verwendet SAM als Synthesizer.
Eingaben des Nutzers werden ebenfalls an das Backend geschickt und dort verarbeitet.
Das Backend kann per command ausserdem einen floppy sound "floppy.mp3" abspielen lassen.
Zur Sicherheit ist die Frontend Backend mit CORS und csrf Token abgesichert, Token werden rotiert.
Diese Sichrheitsmaßnahmen können in der lokalen Entwicklungsumgebung deaktiviert werden, da diese nur in einer Produktivumgebung funktionieren.
Das Frontend implemetiert mehr oder weniger ein "dummes" Terminal, es enthält keine Business logik !! Diese ist vollständig im Backend.

## Chat
Gibt der User in TinyOS "chat" ein, startet eine Chatsession. Dies ist nur möglich wenn der User angemeldet ist.
Der Chat wird mit deepseek ai realisiert. Der api key wird aus einer Umgebungsvariale gelesen, kann aber zu testzwecken auch
direkt im Quellcode hinterlegt werden. Über den Prompt erhält deepseek die Möglichkeit die codes *beeep* und *talk:<text>* zu senden
um einen piepston oder eine Sprachausgabe mit SAM zu veranlassen. diese Steuercodes werden entfernt ehe der Text dem User
angezeigt wird und der Text erscheint gemainsam mit dem Piepston / der Sprachausgabe.
Da deepseek gelegentlich farbige Emojis sendet, müssen diese vor der Übermittlung an das Frontend entfernt werden weil dies
sonst den retro look stört.
Um Missbrauch der kosten produzierenden Chatfunktion zu verhindern, werden saveguards und rate limits implementiert.
Die Chat Nutzung ist auf 5 Minuten pro Stunde und kummuliert 15 Minuten pro Tag begrenzt.
Werden diese Werte überschritten wird der nutzer informiert und der chat beendet und kann erst nach Ablauf von einer Stunde
bzw. am nächsten Tag wieder genutzt werden.
Zusätzlich gilt pro Chat session ein rate limit von maximal 10 Anfragen an deepseek pro Minute. Wird dies überschritten,
ist der Chat für 2 Minuten gesperrt und der User wird darüber informiert und der chat beendet.
Versucht ein Nutzer mehr als 20 anfragen pro Minute an deepseek zu senden wird er für 24 Stunden von jeglicher Nutzung gebannt.
Der Bann wird mit IP Adreasse in der datenbank gespeichert und durchgesetzt. Der User wird informiert und ausgeloggt.
Der User kann den Chat jederzeit durch Eingabe von "exit" beenden.

## Sonstiges

Im finalen Produkt werden alle Textausgaben in Englisch sein, in der Entwicklungsphase kann dies anders sein.
Alle Ausgaben für den User (Fehlermeldungen etc) sollen immer kurz und trotzdem verständlich sein.

Analyse des TinyBASIC-Interpreters
Gesamtarchitektur
Der TinyBASIC-Interpreter ist Teil eines größeren Retro-Terminal-Projekts, das ein 80er-Jahre-Computersystem im Browser simuliert. Das System besteht aus mehreren Hauptkomponenten:

TinyOS - Ein virtuelles Multiuser-Betriebssystem mit Nutzerauthentifizierung
TinyBASIC - Ein vollständiger BASIC-Interpreter, der in TinyOS integriert ist
Virtuelles Dateisystem (VFS) - Verwaltet Dateien für angemeldete und Gastbenutzer
Frontend - Ein Retro-Terminal mit Shader-Grafiken und Sound-Unterstützung
TinyBASIC-Interpreter
Der BASIC-Interpreter ist modular aufgebaut und hat folgende Schlüsselkomponenten:

Hauptstruktur (TinyBASIC)
Verwaltet den Programmzustand (Code, Variablen, Ausführungskontext)
Kommuniziert über Kanäle mit dem Frontend
Unterstützt sowohl direkten Modus als auch Programmausführung
Kernfunktionen
Programmverwaltung: Laden, Speichern und Auflisten von BASIC-Programmen
Ausdrucksauswertung: Parser für mathematische und String-Ausdrücke
Ausführung: Zeilenweise Ausführung von BASIC-Programmen
Fehlerbehandlung: Strukturierte Fehlerbehandlung mit hilfreichen Meldungen
Sprachelemente
Der Interpreter unterstützt klassische BASIC-Funktionen:

Programmflusssteuerung: GOTO, GOSUB/RETURN, IF/THEN, FOR/NEXT
Variablen: Numerische Variablen und Stringvariablen (mit $-Suffix)
Ein-/Ausgabe: PRINT, INPUT
Datenverwaltung: DATA, READ, RESTORE
Dateisystem: LOAD, SAVE, DIR für Dateioperationen
Multimedia: BEEP, SOUND, SAY/SPEAK für Audio-Ausgabe
Grafik: Diverse Grafikbefehle (noch nicht vollständig implementiert)
Integration mit TinyOS
TinyOS startet den BASIC-Interpreter mit dem Befehl basic
Der Interpreter greift auf das Dateisystem des angemeldeten Benutzers zu
Beispielprogramme werden beim ersten Start in das Benutzerverzeichnis kopiert
Bei Gastbenutzern wird ein temporäres VFS im RAM verwendet
Kommunikation mit dem Frontend
Der Interpreter sendet Nachrichten über einen Kanal (OutputChan) an das Frontend
Verschiedene Nachrichtentypen unterstützen Text, Grafik, Sound und Sprachausgabe
Das Frontend stellt diese Ausgaben im Retro-Terminal-Stil dar
Besonderheiten und Einschränkungen
Sicherheit: Nutzer müssen angemeldet sein, um Dateisystemfunktionen zu nutzen
Quota: Angemeldete Nutzer haben ein Disk-Quota von 10MB
Synchronisation: Bei Sprachausgabe mit WAIT-Parameter wird die Ausführung pausiert
Plattformunabhängigkeit: Das VFS funktioniert auf Windows und Linux
Technische Details
Implementiert in Go
Verwendet SQLite für persistente Datenspeicherung
WebSockets für die Echtzeit-Kommunikation mit dem Frontend
Unterstützt authentifizierte Sessions mit JWT-Tokens

## Anylyse durck KI (teilweise eventuell veraltet):
Analyse des TinyBASIC-Interpreters
Gesamtarchitektur
Der TinyBASIC-Interpreter ist Teil eines größeren Retro-Terminal-Projekts, das ein 80er-Jahre-Computersystem im Browser simuliert. Das System besteht aus mehreren Hauptkomponenten:

TinyOS - Ein virtuelles Multiuser-Betriebssystem mit Nutzerauthentifizierung
TinyBASIC - Ein vollständiger BASIC-Interpreter, der in TinyOS integriert ist
Virtuelles Dateisystem (VFS) - Verwaltet Dateien für angemeldete und Gastbenutzer
Frontend - Ein Retro-Terminal mit Shader-Grafiken und Sound-Unterstützung
TinyBASIC-Interpreter
Der BASIC-Interpreter ist modular aufgebaut und hat folgende Schlüsselkomponenten:

Hauptstruktur (TinyBASIC)
Verwaltet den Programmzustand (Code, Variablen, Ausführungskontext)
Kommuniziert über Kanäle mit dem Frontend
Unterstützt sowohl direkten Modus als auch Programmausführung
Kernfunktionen
Programmverwaltung: Laden, Speichern und Auflisten von BASIC-Programmen
Ausdrucksauswertung: Parser für mathematische und String-Ausdrücke
Ausführung: Zeilenweise Ausführung von BASIC-Programmen
Fehlerbehandlung: Strukturierte Fehlerbehandlung mit hilfreichen Meldungen
Sprachelemente
Der Interpreter unterstützt klassische BASIC-Funktionen:

Programmflusssteuerung: GOTO, GOSUB/RETURN, IF/THEN, FOR/NEXT
Variablen: Numerische Variablen und Stringvariablen (mit $-Suffix)
Ein-/Ausgabe: PRINT, INPUT
Datenverwaltung: DATA, READ, RESTORE
Dateisystem: LOAD, SAVE, DIR für Dateioperationen
Multimedia: BEEP, SOUND, SAY/SPEAK für Audio-Ausgabe
Grafik: Diverse Grafikbefehle (noch nicht vollständig implementiert)
Integration mit TinyOS
TinyOS startet den BASIC-Interpreter mit dem Befehl basic
Der Interpreter greift auf das Dateisystem des angemeldeten Benutzers zu
Beispielprogramme werden beim ersten Start in das Benutzerverzeichnis kopiert
Bei Gastbenutzern wird ein temporäres VFS im RAM verwendet
Kommunikation mit dem Frontend
Der Interpreter sendet Nachrichten über einen Kanal (OutputChan) an das Frontend
Verschiedene Nachrichtentypen unterstützen Text, Grafik, Sound und Sprachausgabe
Das Frontend stellt diese Ausgaben im Retro-Terminal-Stil dar
Besonderheiten und Einschränkungen
Sicherheit: Nutzer müssen angemeldet sein, um Dateisystemfunktionen zu nutzen
Quota: Angemeldete Nutzer haben ein Disk-Quota von 10MB
Synchronisation: Bei Sprachausgabe mit WAIT-Parameter wird die Ausführung pausiert
Plattformunabhängigkeit: Das VFS funktioniert auf Windows und Linux
Technische Details
Implementiert in Go
Verwendet SQLite für persistente Datenspeicherung
WebSockets für die Echtzeit-Kommunikation mit dem Frontend
Unterstützt authentifizierte Sessions mit JWT-Tokens
