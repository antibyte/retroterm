package configuration

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config verwaltet die Anwendungskonfiguration
type Config struct {
	settings map[string]map[string]string
	filePath string
	mu       sync.RWMutex
}

var (
	globalConfig *Config
	once         sync.Once
)

// Initialize initialisiert die globale Konfiguration
func Initialize(configPath string) error {
	var err error
	once.Do(func() {
		globalConfig, err = loadConfig(configPath)
		if err != nil {
			return
		}
		// Versuche zusätzlich settings.local.cfg zu laden (falls vorhanden)
		localConfigPath := "settings.local.cfg"
		if _, err := os.Stat(localConfigPath); err == nil {
			err = globalConfig.loadLocalConfig(localConfigPath)
			if err != nil {
				// Silent error - config loading continues with base config
			}
		}
	})
	return err
}

// loadConfig lädt die Konfiguration aus einer Datei
func loadConfig(filePath string) (*Config, error) {
	config := &Config{
		settings: make(map[string]map[string]string),
		filePath: filePath,
	}
	// Prüfe, ob die Datei existiert
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		config.createDefaultConfig()
		if err := config.saveToFile(); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
		return config, nil
	}

	// Lade existierende Konfiguration
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Überspringe leere Zeilen und Kommentare
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Sektion
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if config.settings[currentSection] == nil {
				config.settings[currentSection] = make(map[string]string)
			}
			continue
		}

		// Key-Value Pair
		if strings.Contains(line, "=") && currentSection != "" {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			config.settings[currentSection][key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return config, nil
}

// loadLocalConfig lädt lokale Konfigurationsüberschreibungen
func (c *Config) loadLocalConfig(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Überspringe leere Zeilen und Kommentare
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Sektion
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			if c.settings[currentSection] == nil {
				c.settings[currentSection] = make(map[string]string)
			}
			continue
		}
		// Key-Value Pair - Überschreibt Werte aus der Basis-Konfiguration
		if strings.Contains(line, "=") && currentSection != "" {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Überschreibe den Wert in der lokalen Konfiguration
			c.settings[currentSection][key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// createDefaultConfig erstellt die Standard-Konfiguration mit nur den verwendeten Parametern
func (c *Config) createDefaultConfig() {
	// [System] Sektion
	c.settings["System"] = map[string]string{
		"max_concurrent_users":      "50",
		"system_reserved_cpu":       "25.0",
		"system_reserved_ram_mb":    "128",
		"min_system_ram_mb":         "512",
		"resource_monitor_interval": "5s",
		"session_cleanup_interval":  "30m",
		"max_inactive_time":         "30m",
	}

	// [Security] Sektion
	c.settings["Security"] = map[string]string{
		"max_message_length":              "1000",
		"max_sessions_per_user":           "3",
		"max_sessions_per_ip":             "5",
		"rate_limit_messages":             "60",
		"rate_limit_bandwidth":            "10240",
		"rate_limit_window":               "1m",
		"enable_sql_injection_filter":     "true",
		"enable_xss_filter":               "true",
		"enable_command_injection_filter": "true",
	}

	// [Authentication] Sektion
	c.settings["Authentication"] = map[string]string{
		"max_username_length":  "20",
		"min_username_length":  "3",
		"max_password_length":  "100",
		"min_password_length":  "6",
		"session_token_length": "32",
		"password_hash_cost":   "12",
		"enable_guest_access":  "true",
	}

	// [ChatRateLimit] Sektion
	c.settings["ChatRateLimit"] = map[string]string{
		"max_requests_per_minute":     "10",
		"max_requests_per_minute_ban": "20",
		"rate_limit_duration":         "2m",
		"rate_limit_reset_interval":   "1m",
		"enable_ip_based_limiting":    "true",
		"enable_user_based_limiting":  "true",
	}

	// [BanSystem] Sektion
	c.settings["BanSystem"] = map[string]string{
		"default_ban_duration":   "24h",
		"max_ban_duration":       "720h",
		"enable_automatic_bans":  "true",
		"ban_threshold_requests": "20",
		"ban_cleanup_interval":   "1h",
		"enable_ip_bans":         "true",
		"enable_persistent_bans": "true",
	}

	// [FileSystem] Sektion
	c.settings["FileSystem"] = map[string]string{
		"max_directories_per_user": "20",
		"max_files_per_directory":  "100",
		"max_file_size_kb":         "1024",
		"user_quota_kb":            "10240",
		"enable_guest_persistence": "false",
		"backup_interval":          "1h",
		"enable_file_compression":  "false",
	}

	// [Terminal] Sektion
	c.settings["Terminal"] = map[string]string{
		"max_session_requests_per_minute": "3",
		"session_request_time_window":     "1m",
		"ip_ban_duration":                 "24h",
	}

	// [Editor] Sektion
	c.settings["Editor"] = map[string]string{
		"max_lines": "5000",
	}

	// [Network] Sektion
	c.settings["Network"] = map[string]string{
		"pong_timeout":            "90s",
		"write_wait_timeout":      "10s",
		"max_message_size_kb":     "64",
		"max_messages_per_second": "50",
		"max_channel_buffer":      "10000",
		"client_timeout":          "30s",
	}
	// [Debug] Sektion
	c.settings["Debug"] = map[string]string{
		"enable_debug_logging":          "true",
		"log_level":                     "INFO",
		"log_file":                      "debug.log",
		"max_log_size_mb":               "10",
		"log_rotation_count":            "3",
		"enable_performance_monitoring": "false",
		"enable_request_logging":        "false",
		// Selektive Logging-Bereiche
		"log_websocket":  "false",
		"log_terminal":   "false",
		"log_auth":       "true",
		"log_chat":       "false",
		"log_editor":     "false",
		"log_filesystem": "false",
		"log_resources":  "true",
		"log_security":   "true",
		"log_bansystem":  "true",
		"log_tinybasic":  "false",
		"log_database":   "false",
		"log_session":    "false",
		"log_config":     "true",
		"log_general":    "true",
	}
}

// saveToFile speichert die aktuelle Konfiguration in die Datei
func (c *Config) saveToFile() error {
	// Erstelle Verzeichnis falls es nicht existiert
	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(c.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Schreibe Header
	file.WriteString("; TinyOS Configuration File\n")
	file.WriteString("; Generated automatically - modify with care\n")
	file.WriteString(";\n\n")

	// Schreibe alle Sektionen in einer definierten Reihenfolge
	sections := []string{"System", "Security", "Authentication", "ChatRateLimit", "BanSystem", "FileSystem", "Terminal", "Editor", "Network", "Debug"}

	for _, section := range sections {
		if settings, exists := c.settings[section]; exists {
			file.WriteString(fmt.Sprintf("[%s]\n", section))

			for key, value := range settings {
				file.WriteString(fmt.Sprintf("%s = %s\n", key, value))
			}

			file.WriteString("\n")
		}
	}

	return nil
}

// GetString gibt einen String-Wert aus der Konfiguration zurück
func GetString(section, key, defaultValue string) string {
	if globalConfig == nil {
		return defaultValue
	}

	globalConfig.mu.RLock()
	defer globalConfig.mu.RUnlock()

	if sectionMap, exists := globalConfig.settings[section]; exists {
		if value, exists := sectionMap[key]; exists {
			return value
		}
	}

	return defaultValue
}

// GetInt gibt einen Integer-Wert aus der Konfiguration zurück
func GetInt(section, key string, defaultValue int) int {
	str := GetString(section, key, "")
	if str == "" {
		return defaultValue
	}

	if value, err := strconv.Atoi(str); err == nil {
		return value
	}

	return defaultValue
}

// GetFloat gibt einen Float-Wert aus der Konfiguration zurück
func GetFloat(section, key string, defaultValue float64) float64 {
	str := GetString(section, key, "")
	if str == "" {
		return defaultValue
	}

	if value, err := strconv.ParseFloat(str, 64); err == nil {
		return value
	}

	return defaultValue
}

// GetBool gibt einen Boolean-Wert aus der Konfiguration zurück
func GetBool(section, key string, defaultValue bool) bool {
	str := GetString(section, key, "")
	if str == "" {
		return defaultValue
	}

	if value, err := strconv.ParseBool(str); err == nil {
		return value
	}

	return defaultValue
}

// GetDuration gibt einen Duration-Wert aus der Konfiguration zurück
func GetDuration(section, key string, defaultValue time.Duration) time.Duration {
	str := GetString(section, key, "")
	if str == "" {
		return defaultValue
	}

	if value, err := time.ParseDuration(str); err == nil {
		return value
	}

	return defaultValue
}

// GetSection returns all key-value pairs from a configuration section
func GetSection(sectionName string) map[string]string {
	if globalConfig == nil {
		return make(map[string]string)
	}

	globalConfig.mu.RLock()
	defer globalConfig.mu.RUnlock()

	if section, exists := globalConfig.settings[sectionName]; exists {
		// Return a copy to prevent external modifications
		result := make(map[string]string)
		for key, value := range section {
			result[key] = value
		}
		return result
	}

	return make(map[string]string)
}

// SetString setzt einen String-Wert in der Konfiguration
func SetString(section, key, value string) {
	if globalConfig == nil {
		return
	}

	globalConfig.mu.Lock()
	defer globalConfig.mu.Unlock()

	if globalConfig.settings[section] == nil {
		globalConfig.settings[section] = make(map[string]string)
	}

	globalConfig.settings[section][key] = value
}

// Save speichert die aktuelle Konfiguration in die Datei
func Save() error {
	if globalConfig == nil {
		return fmt.Errorf("configuration not initialized")
	}

	globalConfig.mu.RLock()
	defer globalConfig.mu.RUnlock()

	return globalConfig.saveToFile()
}
