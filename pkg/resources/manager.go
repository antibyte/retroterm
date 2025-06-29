package resources

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
)

// ResourceLimits definiert die Grenzen für einen Benutzer
type ResourceLimits struct {
	MaxCPUPercent    float64       `json:"max_cpu_percent"`    // CPU-Anteil in % (0-100)
	MaxMemoryMB      int64         `json:"max_memory_mb"`      // RAM in MB
	MaxGoroutines    int           `json:"max_goroutines"`     // Anzahl Goroutines
	MaxExecutionTime time.Duration `json:"max_execution_time"` // Max. Ausführungszeit für BASIC-Programme
	MaxFileSize      int64         `json:"max_file_size"`      // Max. Dateigröße in Bytes
	MaxTotalFiles    int           `json:"max_total_files"`    // Max. Anzahl Dateien
}

// UserResource verwaltet die Ressourcen eines einzelnen Benutzers
type UserResource struct {
	Username          string
	Limits            ResourceLimits
	CurrentCPU        float64
	CurrentMemoryMB   int64
	CurrentGoroutines int
	LastActivity      time.Time
	Context           context.Context
	Cancel            context.CancelFunc
	ExecutionMutex    sync.Mutex
}

// SystemResourceManager verwaltet alle Benutzerressourcen
type SystemResourceManager struct {
	mu                 sync.RWMutex
	users              map[string]*UserResource
	systemReservedCPU  float64 // % CPU reserved for system
	systemReservedRAM  int64   // MB RAM reserved for system
	totalSystemRAM     int64   // Total available RAM in MB
	maxConcurrentUsers int     // Max. concurrent users

	// Monitoring
	resourceTicker *time.Ticker
	stopMonitoring chan bool
}

// NewSystemResourceManager erstellt einen neuen Ressourcenmanager
func NewSystemResourceManager() *SystemResourceManager {
	// Systemressourcen ermitteln
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	totalRAM := int64(m.Sys / (1024 * 1024)) // Bytes zu MB
	minRAM := configuration.GetInt("System", "min_system_ram_mb", 512)
	if totalRAM < int64(minRAM) {
		totalRAM = int64(minRAM) // Minimum aus Konfiguration
	}

	// Werte aus Konfiguration lesen
	systemReservedCPU := configuration.GetFloat("System", "system_reserved_cpu", 25.0)
	systemReservedRAM := configuration.GetInt("System", "system_reserved_ram_mb", int(totalRAM/4))
	maxUsers := configuration.GetInt("System", "max_concurrent_users", 50)

	manager := &SystemResourceManager{
		users:              make(map[string]*UserResource),
		systemReservedCPU:  systemReservedCPU,
		systemReservedRAM:  int64(systemReservedRAM),
		totalSystemRAM:     totalRAM,
		maxConcurrentUsers: maxUsers,
		stopMonitoring:     make(chan bool),
	}
	// Start resource monitoring
	manager.startResourceMonitoring()

	logger.Info(logger.AreaResources, "Initialized - Total RAM: %d MB, Reserved: %d MB",
		totalRAM, manager.systemReservedRAM)

	return manager
}

// RegisterUser registriert einen neuen Benutzer im Ressourcenmanager
func (rm *SystemResourceManager) RegisterUser(username string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if len(rm.users) >= rm.maxConcurrentUsers {
		return fmt.Errorf("maximum concurrent users reached (%d)", rm.maxConcurrentUsers)
	}

	if _, exists := rm.users[username]; exists {
		return fmt.Errorf("user already registered: %s", username)
	}

	// Ressourcenlimits basierend auf aktueller Benutzerzahl berechnen
	activeUsers := len(rm.users) + 1 // +1 für den neuen Benutzer
	limits := rm.calculateUserLimits(activeUsers)

	// Kontext für Benutzer erstellen
	ctx, cancel := context.WithCancel(context.Background())

	userResource := &UserResource{
		Username:     username,
		Limits:       limits,
		LastActivity: time.Now(),
		Context:      ctx,
		Cancel:       cancel,
	}

	rm.users[username] = userResource
	// Recalculate limits for all existing users
	rm.recalculateAllUserLimits()

	logger.Info(logger.AreaResources, "User registered: %s (Active users: %d, CPU: %.1f%%, RAM: %d MB)",
		username, activeUsers, limits.MaxCPUPercent, limits.MaxMemoryMB)

	return nil
}

// UnregisterUser entfernt einen Benutzer aus dem Ressourcenmanager
func (rm *SystemResourceManager) UnregisterUser(username string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	userResource, exists := rm.users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}

	// Kontext abbrechen (stoppt alle laufenden Operationen)
	userResource.Cancel()

	delete(rm.users, username)

	// Limits für verbleibende Benutzer neu berechnen	rm.recalculateAllUserLimits()

	logger.Info(logger.AreaResources, "User unregistered: %s (Remaining users: %d)",
		username, len(rm.users))

	return nil
}

// GetUserContext gibt den Kontext für einen Benutzer zurück
func (rm *SystemResourceManager) GetUserContext(username string) (context.Context, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	userResource, exists := rm.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", username)
	}

	return userResource.Context, nil
}

// GetUserLimits gibt die aktuellen Ressourcenlimits für einen Benutzer zurück
func (rm *SystemResourceManager) GetUserLimits(username string) (ResourceLimits, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	userResource, exists := rm.users[username]
	if !exists {
		return ResourceLimits{}, fmt.Errorf("user not found: %s", username)
	}

	return userResource.Limits, nil
}

// CheckResourceUsage überprüft, ob ein Benutzer seine Limits einhält
func (rm *SystemResourceManager) CheckResourceUsage(username string) error {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	userResource, exists := rm.users[username]
	if !exists {
		return fmt.Errorf("user not found: %s", username)
	}

	// CPU-Check (würde echte CPU-Messung benötigen)
	if userResource.CurrentCPU > userResource.Limits.MaxCPUPercent {
		return fmt.Errorf("CPU limit exceeded: %.1f%% > %.1f%%",
			userResource.CurrentCPU, userResource.Limits.MaxCPUPercent)
	}

	// Memory-Check
	if userResource.CurrentMemoryMB > userResource.Limits.MaxMemoryMB {
		return fmt.Errorf("memory limit exceeded: %d MB > %d MB",
			userResource.CurrentMemoryMB, userResource.Limits.MaxMemoryMB)
	}

	return nil
}

// calculateUserLimits berechnet die Ressourcenlimits pro Benutzer
func (rm *SystemResourceManager) calculateUserLimits(activeUsers int) ResourceLimits {
	if activeUsers <= 0 {
		activeUsers = 1
	}

	// Verfügbare Ressourcen (nach Systemreservierung)
	availableCPU := 100.0 - rm.systemReservedCPU
	availableRAM := rm.totalSystemRAM - rm.systemReservedRAM

	// Pro Benutzer aufteilen
	cpuPerUser := availableCPU / float64(activeUsers)
	ramPerUser := availableRAM / int64(activeUsers)
	// Minimum-Garantien
	if cpuPerUser < 1.0 {
		cpuPerUser = 1.0
	}
	if ramPerUser < 32 {
		ramPerUser = 32 // Min. 32 MB pro Benutzer
	}

	return ResourceLimits{
		MaxCPUPercent:    cpuPerUser,
		MaxMemoryMB:      ramPerUser,
		MaxGoroutines:    configuration.GetInt("BasicPrograms", "max_goroutines", 20),
		MaxExecutionTime: configuration.GetDuration("BasicPrograms", "max_execution_time", 24*time.Hour),
		MaxFileSize:      int64(configuration.GetInt("BasicPrograms", "max_file_size_bytes", 1024*1024)),
		MaxTotalFiles:    configuration.GetInt("BasicPrograms", "max_total_files", 100),
	}
}

// recalculateAllUserLimits berechnet die Limits für alle Benutzer neu
func (rm *SystemResourceManager) recalculateAllUserLimits() {
	activeUsers := len(rm.users)
	if activeUsers == 0 {
		return
	}

	newLimits := rm.calculateUserLimits(activeUsers)

	for username, userResource := range rm.users {
		userResource.Limits = newLimits
		logger.Info(logger.AreaResources, "Updated limits for %s: CPU %.1f%%, RAM %d MB",
			username, newLimits.MaxCPUPercent, newLimits.MaxMemoryMB)
	}
}

// startResourceMonitoring startet das Ressourcen-Monitoring
func (rm *SystemResourceManager) startResourceMonitoring() {
	interval := configuration.GetDuration("System", "resource_monitor_interval", 5*time.Second)
	rm.resourceTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-rm.resourceTicker.C:
				rm.monitorResources()
			case <-rm.stopMonitoring:
				rm.resourceTicker.Stop()
				return
			}
		}
	}()
}

// monitorResources überwacht die Ressourcennutzung
func (rm *SystemResourceManager) monitorResources() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentSystemRAM := int64(m.Alloc / (1024 * 1024))

	logger.Info(logger.AreaResources, "System RAM: %d MB, Active users: %d, Goroutines: %d",
		currentSystemRAM, len(rm.users), runtime.NumGoroutine())

	// Check each user for activity
	for username, userResource := range rm.users {
		// Detect inactive users (more than 30 minutes without activity)
		if time.Since(userResource.LastActivity) > 30*time.Minute {
			logger.Info(logger.AreaResources, "User %s inactive for %v",
				username, time.Since(userResource.LastActivity))
		}
	}
}

// UpdateUserActivity aktualisiert die letzte Aktivität eines Benutzers
func (rm *SystemResourceManager) UpdateUserActivity(username string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if userResource, exists := rm.users[username]; exists {
		userResource.LastActivity = time.Now()
	}
}

// GetSystemStats gibt Systemstatistiken zurück
func (rm *SystemResourceManager) GetSystemStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"total_ram_mb":     rm.totalSystemRAM,
		"reserved_ram_mb":  rm.systemReservedRAM,
		"current_ram_mb":   int64(m.Alloc / (1024 * 1024)),
		"active_users":     len(rm.users),
		"max_users":        rm.maxConcurrentUsers,
		"total_goroutines": runtime.NumGoroutine(),
		"cpu_reserved":     rm.systemReservedCPU,
	}
}

// Stop stoppt den Ressourcenmanager
func (rm *SystemResourceManager) Stop() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Alle Benutzer-Kontexte abbrechen
	for username, userResource := range rm.users {
		userResource.Cancel()
		logger.Info(logger.AreaResources, "Cancelled context for user: %s", username)
	}

	// Stop monitoring
	close(rm.stopMonitoring)

	logger.Info(logger.AreaResources, "Resource manager stopped")
}
