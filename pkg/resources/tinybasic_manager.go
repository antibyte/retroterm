package resources

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// TinyBASICResourceManager erweitert den SystemResourceManager für TinyBASIC-spezifische Limits
type TinyBASICResourceManager struct {
	*SystemResourceManager
	basicExecutions map[string]*BasicExecution // Aktive BASIC-Ausführungen pro Benutzer
	basicMutex      sync.RWMutex
}

// BasicExecution verwaltet eine laufende BASIC-Programmausführung
type BasicExecution struct {
	Username     string
	ProgramName  string
	StartTime    time.Time
	Context      context.Context
	Cancel       context.CancelFunc
	MemoryUsage  int64 // Geschätzte Speichernutzung in Bytes
	LoopCount    int64 // Anzahl der ausgeführten Schleifen-Iterationen
	MaxLoopCount int64 // Maximum erlaubte Schleifen-Iterationen
	CommandCount int64 // Anzahl ausgeführter Befehle
	MaxCommands  int64 // Maximum erlaubte Befehle
}

// NewTinyBASICResourceManager erstellt einen neuen TinyBASIC-Ressourcenmanager
func NewTinyBASICResourceManager() *TinyBASICResourceManager {
	return &TinyBASICResourceManager{
		SystemResourceManager: NewSystemResourceManager(),
		basicExecutions:       make(map[string]*BasicExecution),
	}
}

// StartBasicExecution startet eine neue BASIC-Programmausführung mit Ressourcenlimits
func (trm *TinyBASICResourceManager) StartBasicExecution(username, programName string) (*BasicExecution, error) {
	trm.basicMutex.Lock()
	defer trm.basicMutex.Unlock()

	// Prüfe, ob bereits eine Ausführung läuft
	if execution, exists := trm.basicExecutions[username]; exists {
		return nil, fmt.Errorf("user %s already has a running program: %s", username, execution.ProgramName)
	}

	// Hole Benutzer-Limits
	limits, err := trm.GetUserLimits(username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user limits: %v", err)
	}

	// Erstelle Kontext mit Timeout
	ctx, cancel := context.WithTimeout(context.Background(), limits.MaxExecutionTime)
	execution := &BasicExecution{
		Username:     username,
		ProgramName:  programName,
		StartTime:    time.Now(),
		Context:      ctx,
		Cancel:       cancel,
		MaxLoopCount: 3000000,  // 3 Millionen Schleifen-Iterationen
		MaxCommands:  20000000, // 20 Millionen Befehle
	}

	trm.basicExecutions[username] = execution

	// Aktivität aktualisieren
	trm.UpdateUserActivity(username)

	return execution, nil
}

// StopBasicExecution stoppt eine BASIC-Programmausführung
func (trm *TinyBASICResourceManager) StopBasicExecution(username string) error {
	trm.basicMutex.Lock()
	defer trm.basicMutex.Unlock()

	execution, exists := trm.basicExecutions[username]
	if !exists {
		return fmt.Errorf("no running program for user: %s", username)
	}

	execution.Cancel()
	delete(trm.basicExecutions, username)

	runtime := time.Since(execution.StartTime)

	// Log der Ausführungsstatistiken
	fmt.Printf("[BASIC-RESOURCE] Program stopped - User: %s, Program: %s, Runtime: %v, Commands: %d, Loops: %d\n",
		username, execution.ProgramName, runtime, execution.CommandCount, execution.LoopCount)

	return nil
}

// CheckBasicExecution überprüft, ob eine BASIC-Ausführung ihre Limits einhält
func (trm *TinyBASICResourceManager) CheckBasicExecution(username string, commandType string) error {
	trm.basicMutex.RLock()
	execution, exists := trm.basicExecutions[username]
	trm.basicMutex.RUnlock()

	if !exists {
		return fmt.Errorf("no running program for user: %s", username)
	}

	// Prüfe Kontext (Timeout)
	select {
	case <-execution.Context.Done():
		trm.StopBasicExecution(username)
		return fmt.Errorf("program execution timeout for user %s", username)
	default:
	}

	// Aktualisiere Zähler
	trm.basicMutex.Lock()
	execution.CommandCount++

	// Schleifenzähler für bestimmte Befehle
	if commandType == "FOR" || commandType == "NEXT" || commandType == "GOTO" || commandType == "GOSUB" {
		execution.LoopCount++
	}
	trm.basicMutex.Unlock()

	// Prüfe Limits
	if execution.CommandCount > execution.MaxCommands {
		trm.StopBasicExecution(username)
		return fmt.Errorf("command limit exceeded for user %s: %d > %d",
			username, execution.CommandCount, execution.MaxCommands)
	}

	if execution.LoopCount > execution.MaxLoopCount {
		trm.StopBasicExecution(username)
		return fmt.Errorf("loop limit exceeded for user %s: %d > %d",
			username, execution.LoopCount, execution.MaxLoopCount)
	}

	// Aktivität aktualisieren
	trm.UpdateUserActivity(username)

	return nil
}

// GetBasicExecutionStats gibt Statistiken über eine laufende BASIC-Ausführung zurück
func (trm *TinyBASICResourceManager) GetBasicExecutionStats(username string) (map[string]interface{}, error) {
	trm.basicMutex.RLock()
	execution, exists := trm.basicExecutions[username]
	trm.basicMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no running program for user: %s", username)
	}

	runtime := time.Since(execution.StartTime)

	return map[string]interface{}{
		"username":     execution.Username,
		"program":      execution.ProgramName,
		"runtime_ms":   runtime.Milliseconds(),
		"commands":     execution.CommandCount,
		"loops":        execution.LoopCount,
		"max_commands": execution.MaxCommands,
		"max_loops":    execution.MaxLoopCount,
		"memory_usage": execution.MemoryUsage,
	}, nil
}

// CreateResourceLimitedContext erstellt einen Kontext mit Ressourcenlimits für einen Benutzer
func (trm *TinyBASICResourceManager) CreateResourceLimitedContext(username string) (context.Context, error) {
	limits, err := trm.GetUserLimits(username)
	if err != nil {
		return nil, err
	}

	// Erstelle Kontext mit Timeout basierend auf Benutzerlimits
	ctx, _ := context.WithTimeout(context.Background(), limits.MaxExecutionTime)

	return ctx, nil
}

// MemoryGuard überwacht den Speicherverbrauch und stoppt Programme bei Überschreitung
func (trm *TinyBASICResourceManager) MemoryGuard() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			currentMemMB := int64(m.Alloc / (1024 * 1024))

			trm.basicMutex.RLock()
			activeExecutions := len(trm.basicExecutions)
			trm.basicMutex.RUnlock()

			// Wenn Speicherverbrauch zu hoch, stoppe Programme
			maxAllowedMemMB := trm.totalSystemRAM - trm.systemReservedRAM
			if currentMemMB > maxAllowedMemMB && activeExecutions > 0 {
				fmt.Printf("[MEMORY-GUARD] High memory usage: %d MB, stopping oldest programs\n", currentMemMB)
				trm.stopOldestExecutions(1)
			}
		}
	}()
}

// stopOldestExecutions stoppt die ältesten laufenden Programme
func (trm *TinyBASICResourceManager) stopOldestExecutions(count int) {
	trm.basicMutex.Lock()
	defer trm.basicMutex.Unlock()

	type executionInfo struct {
		username  string
		startTime time.Time
	}

	var executions []executionInfo
	for username, exec := range trm.basicExecutions {
		executions = append(executions, executionInfo{
			username:  username,
			startTime: exec.StartTime,
		})
	}

	// Sortiere nach Startzeit (älteste zuerst)
	for i := 0; i < len(executions)-1; i++ {
		for j := i + 1; j < len(executions); j++ {
			if executions[i].startTime.After(executions[j].startTime) {
				executions[i], executions[j] = executions[j], executions[i]
			}
		}
	}

	// Stoppe die ältesten Programme
	stopped := 0
	for _, exec := range executions {
		if stopped >= count {
			break
		}

		if execution, exists := trm.basicExecutions[exec.username]; exists {
			execution.Cancel()
			delete(trm.basicExecutions, exec.username)
			fmt.Printf("[MEMORY-GUARD] Stopped program for user %s due to memory pressure\n", exec.username)
			stopped++
		}
	}
}

// GetAllBasicExecutions gibt alle aktiven BASIC-Ausführungen zurück
func (trm *TinyBASICResourceManager) GetAllBasicExecutions() map[string]*BasicExecution {
	trm.basicMutex.RLock()
	defer trm.basicMutex.RUnlock()

	result := make(map[string]*BasicExecution)
	for username, execution := range trm.basicExecutions {
		result[username] = execution
	}

	return result
}

// EstimateMemoryUsage schätzt den Speicherverbrauch für ein BASIC-Programm
func (trm *TinyBASICResourceManager) EstimateMemoryUsage(username string, programSize int) error {
	trm.basicMutex.Lock()
	defer trm.basicMutex.Unlock()

	execution, exists := trm.basicExecutions[username]
	if !exists {
		return fmt.Errorf("no running program for user: %s", username)
	}

	// Einfache Schätzung: Programmgröße * 10 + Basis-Overhead
	estimatedMemory := int64(programSize*10 + 1024*1024) // 1MB Basis + 10x Programmgröße
	execution.MemoryUsage = estimatedMemory

	// Prüfe Speicherlimit
	limits, err := trm.GetUserLimits(username)
	if err != nil {
		return err
	}

	if estimatedMemory > limits.MaxMemoryMB*1024*1024 {
		return fmt.Errorf("estimated memory usage %d MB exceeds limit %d MB for user %s",
			estimatedMemory/(1024*1024), limits.MaxMemoryMB, username)
	}

	return nil
}
