package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
)

// LogLevel definiert die verschiedenen Log-Level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

var logLevelNames = map[LogLevel]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

// LogArea definiert die verschiedenen Log-Bereiche
type LogArea string

const (
	AreaWebSocket  LogArea = "websocket"
	AreaTerminal   LogArea = "terminal"
	AreaAuth       LogArea = "auth"
	AreaChat       LogArea = "chat"
	AreaEditor     LogArea = "editor"
	AreaFileSystem LogArea = "filesystem"
	AreaResources  LogArea = "resources"
	AreaSecurity   LogArea = "security"
	AreaBanSystem  LogArea = "bansystem"
	AreaTinyBasic  LogArea = "tinybasic"
	AreaDatabase   LogArea = "database"
	AreaSession    LogArea = "session"
	AreaConfig     LogArea = "config"
	AreaGeneral    LogArea = "general"
	AreaChess      LogArea = "chess"
)

// Logger ist das Hauptlogging-System
type Logger struct {
	enabled       int32              // atomic bool - performance critical
	level         int32              // atomic LogLevel
	areaEnabled   map[LogArea]*int32 // atomic bools per area
	file          *os.File
	mutex         sync.RWMutex
	logPath       string
	maxSizeMB     int64
	rotationCount int
	currentSize   int64
}

var (
	globalLogger *Logger
	initOnce     sync.Once
)

// Initialize initialisiert das globale Logging-System
func Initialize() error {
	var err error
	initOnce.Do(func() {
		globalLogger, err = newLogger()
	})
	return err
}

// newLogger erstellt einen neuen Logger
func newLogger() (*Logger, error) {
	l := &Logger{
		areaEnabled: make(map[LogArea]*int32),
	}
	// Initialisiere alle Bereiche mit atomic ints
	areas := []LogArea{
		AreaWebSocket, AreaTerminal, AreaAuth, AreaChat, AreaEditor,
		AreaFileSystem, AreaResources, AreaSecurity, AreaBanSystem,
		AreaTinyBasic, AreaDatabase, AreaSession, AreaConfig, AreaGeneral,
		AreaChess,
	}

	for _, area := range areas {
		l.areaEnabled[area] = new(int32)
	}

	// Lade Konfiguration
	if err := l.loadConfig(); err != nil {
		return nil, err
	}

	// Öffne Log-Datei
	if err := l.openLogFile(); err != nil {
		return nil, err
	}

	return l, nil
}

// loadConfig lädt die Logging-Konfiguration
func (l *Logger) loadConfig() error {
	// Basis-Konfiguration
	enabled := configuration.GetBool("Debug", "enable_debug_logging", true)
	atomic.StoreInt32(&l.enabled, boolToInt32(enabled))

	levelStr := configuration.GetString("Debug", "log_level", "INFO")
	level := parseLogLevel(levelStr)
	atomic.StoreInt32(&l.level, int32(level))

	l.logPath = configuration.GetString("Debug", "log_file", "debug.log")
	l.maxSizeMB = int64(configuration.GetInt("Debug", "max_log_size_mb", 10))
	l.rotationCount = configuration.GetInt("Debug", "log_rotation_count", 3)

	// Bereichs-spezifische Konfiguration
	for area, atomicBool := range l.areaEnabled {
		configKey := fmt.Sprintf("log_%s", string(area))
		enabled := configuration.GetBool("Debug", configKey, false)
		atomic.StoreInt32(atomicBool, boolToInt32(enabled))
	}

	return nil
}

// openLogFile öffnet die Log-Datei
func (l *Logger) openLogFile() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// Schließe alte Datei falls vorhanden
	if l.file != nil {
		l.file.Close()
	}

	// Erstelle Verzeichnis falls nötig
	dir := filepath.Dir(l.logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Öffne neue Datei
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	l.file = file

	// Aktuelle Dateigröße ermitteln
	if stat, err := file.Stat(); err == nil {
		l.currentSize = stat.Size()
	}

	return nil
}

// rotateLogFile rotiert die Log-Datei wenn sie zu groß wird
func (l *Logger) rotateLogFile() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// Rotiere bestehende Dateien
	for i := l.rotationCount - 1; i >= 1; i-- {
		oldName := fmt.Sprintf("%s.%d", l.logPath, i)
		newName := fmt.Sprintf("%s.%d", l.logPath, i+1)

		if i == l.rotationCount-1 {
			// Lösche älteste Datei
			os.Remove(newName)
		}

		os.Rename(oldName, newName)
	}

	// Verschiebe aktuelle Datei zu .1
	os.Rename(l.logPath, l.logPath+".1")

	// Öffne neue Datei
	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.currentSize = 0

	return nil
}

// isEnabled prüft ob Logging aktiviert ist (atomic, sehr schnell)
func (l *Logger) isEnabled() bool {
	return atomic.LoadInt32(&l.enabled) != 0
}

// isAreaEnabled prüft ob ein Bereich aktiviert ist (atomic, sehr schnell)
func (l *Logger) isAreaEnabled(area LogArea) bool {
	if atomicBool, exists := l.areaEnabled[area]; exists {
		return atomic.LoadInt32(atomicBool) != 0
	}
	return false
}

// shouldLog prüft ob ein Log-Eintrag geschrieben werden soll
func (l *Logger) shouldLog(level LogLevel, area LogArea) bool {
	// Schnelle atomic Checks zuerst
	if !l.isEnabled() {
		return false
	}

	if atomic.LoadInt32(&l.level) > int32(level) {
		return false
	}

	return l.isAreaEnabled(area)
}

// writeLog schreibt den Log-Eintrag
func (l *Logger) writeLog(level LogLevel, area LogArea, format string, args ...interface{}) {
	// Formatiere Nachricht
	message := fmt.Sprintf(format, args...)

	// Hole Caller-Info
	_, file, line, _ := runtime.Caller(3)
	filename := filepath.Base(file)

	// Erstelle Log-Eintrag
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	logEntry := fmt.Sprintf("[%s] %s [%s:%d] [%s] %s\n",
		timestamp,
		logLevelNames[level],
		filename,
		line,
		strings.ToUpper(string(area)),
		message)

	// Schreibe in Datei
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.file != nil {
		n, err := l.file.WriteString(logEntry)
		if err == nil {
			l.currentSize += int64(n)
			l.file.Sync() // Sofort auf Disk schreiben

			// Prüfe ob Rotation nötig ist
			if l.currentSize > l.maxSizeMB*1024*1024 {
				l.rotateLogFile()
			}
		}
	}

	// Zusätzlich in Standard-Log für wichtige Meldungen
	if level >= WARN {
		log.Printf("[%s] [%s] %s", logLevelNames[level], strings.ToUpper(string(area)), message)
	}
}

// Public Logging-Funktionen für verschiedene Bereiche und Level

// Debug schreibt Debug-Logs
func Debug(area LogArea, format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.shouldLog(DEBUG, area) {
		globalLogger.writeLog(DEBUG, area, format, args...)
	}
}

// Info schreibt Info-Logs
func Info(area LogArea, format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.shouldLog(INFO, area) {
		globalLogger.writeLog(INFO, area, format, args...)
	}
}

// Warn schreibt Warning-Logs
func Warn(area LogArea, format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.shouldLog(WARN, area) {
		globalLogger.writeLog(WARN, area, format, args...)
	}
}

// Error schreibt Error-Logs
func Error(area LogArea, format string, args ...interface{}) {
	if globalLogger != nil && globalLogger.shouldLog(ERROR, area) {
		globalLogger.writeLog(ERROR, area, format, args...)
	}
}

// Fatal schreibt Fatal-Logs und beendet das Programm
func Fatal(area LogArea, format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.writeLog(FATAL, area, format, args...)
	}
	log.Fatalf("[FATAL] [%s] %s", strings.ToUpper(string(area)), fmt.Sprintf(format, args...))
}

// Convenience-Funktionen für häufig verwendete Bereiche

// WebSocket Logging
func WebSocketDebug(format string, args ...interface{}) { Debug(AreaWebSocket, format, args...) }
func WebSocketInfo(format string, args ...interface{})  { Info(AreaWebSocket, format, args...) }
func WebSocketWarn(format string, args ...interface{})  { Warn(AreaWebSocket, format, args...) }
func WebSocketError(format string, args ...interface{}) { Error(AreaWebSocket, format, args...) }

// Auth Logging
func AuthDebug(format string, args ...interface{}) { Debug(AreaAuth, format, args...) }
func AuthInfo(format string, args ...interface{})  { Info(AreaAuth, format, args...) }
func AuthWarn(format string, args ...interface{})  { Warn(AreaAuth, format, args...) }
func AuthError(format string, args ...interface{}) { Error(AreaAuth, format, args...) }

// Security Logging
func SecurityDebug(format string, args ...interface{}) { Debug(AreaSecurity, format, args...) }
func SecurityInfo(format string, args ...interface{})  { Info(AreaSecurity, format, args...) }
func SecurityWarn(format string, args ...interface{})  { Warn(AreaSecurity, format, args...) }
func SecurityError(format string, args ...interface{}) { Error(AreaSecurity, format, args...) }

// Resources Logging
func ResourcesDebug(format string, args ...interface{}) { Debug(AreaResources, format, args...) }
func ResourcesInfo(format string, args ...interface{})  { Info(AreaResources, format, args...) }
func ResourcesWarn(format string, args ...interface{})  { Warn(AreaResources, format, args...) }
func ResourcesError(format string, args ...interface{}) { Error(AreaResources, format, args...) }

// Config Logging
func ConfigDebug(format string, args ...interface{}) { Debug(AreaConfig, format, args...) }
func ConfigInfo(format string, args ...interface{})  { Info(AreaConfig, format, args...) }
func ConfigWarn(format string, args ...interface{})  { Warn(AreaConfig, format, args...) }
func ConfigError(format string, args ...interface{}) { Error(AreaConfig, format, args...) }

// ReloadConfig lädt die Konfiguration neu
func ReloadConfig() error {
	if globalLogger != nil {
		return globalLogger.loadConfig()
	}
	return fmt.Errorf("logger not initialized")
}

// EnableArea aktiviert Logging für einen Bereich
func EnableArea(area LogArea) {
	if globalLogger != nil {
		if atomicBool, exists := globalLogger.areaEnabled[area]; exists {
			atomic.StoreInt32(atomicBool, 1)
		}
	}
}

// DisableArea deaktiviert Logging für einen Bereich
func DisableArea(area LogArea) {
	if globalLogger != nil {
		if atomicBool, exists := globalLogger.areaEnabled[area]; exists {
			atomic.StoreInt32(atomicBool, 0)
		}
	}
}

// GetAreaStatus gibt den Status eines Bereichs zurück
func GetAreaStatus(area LogArea) bool {
	if globalLogger != nil {
		return globalLogger.isAreaEnabled(area)
	}
	return false
}

// ListAreas gibt alle verfügbaren Bereiche zurück
func ListAreas() []LogArea {
	return []LogArea{
		AreaWebSocket, AreaTerminal, AreaAuth, AreaChat, AreaEditor,
		AreaFileSystem, AreaResources, AreaSecurity, AreaBanSystem,
		AreaTinyBasic, AreaDatabase, AreaSession, AreaConfig, AreaGeneral,
	}
}

// Hilfsfunktionen
func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN", "WARNING":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

// Close schließt das Logging-System
func Close() {
	if globalLogger != nil {
		globalLogger.mutex.Lock()
		defer globalLogger.mutex.Unlock()

		if globalLogger.file != nil {
			globalLogger.file.Close()
			globalLogger.file = nil
		}
	}
}
