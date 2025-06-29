package terminal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// JSONValidator validiert JSON-Eingaben auf Sicherheit
type JSONValidator struct {
	MaxDepth      int
	MaxKeys       int
	MaxStringLen  int
	MaxArraySize  int
	MaxObjectSize int
}

// Sicherheitskonstanten für JSON-Validierung
const (
	MaxJSONDepth      = 10    // Maximale Verschachtelungstiefe
	MaxJSONKeys       = 100   // Maximale Anzahl Keys pro Objekt
	MaxJSONStringLen  = 10000 // Maximale String-Länge
	MaxJSONArraySize  = 1000  // Maximale Array-Größe
	MaxJSONObjectSize = 100   // Maximale Objekt-Größe
)

var (
	ErrJSONTooDeep        = errors.New("JSON nesting too deep")
	ErrJSONTooManyKeys    = errors.New("too many keys in JSON object")
	ErrJSONStringTooLong  = errors.New("JSON string too long")
	ErrJSONArrayTooLarge  = errors.New("JSON array too large")
	ErrJSONObjectTooLarge = errors.New("JSON object too large")
	ErrJSONMalicious      = errors.New("potentially malicious JSON detected")
)

// NewJSONValidator erstellt einen neuen JSON-Validator
func NewJSONValidator() *JSONValidator {
	return &JSONValidator{
		MaxDepth:      MaxJSONDepth,
		MaxKeys:       MaxJSONKeys,
		MaxStringLen:  MaxJSONStringLen,
		MaxArraySize:  MaxJSONArraySize,
		MaxObjectSize: MaxJSONObjectSize,
	}
}

// ValidateJSON validiert JSON-Daten auf Sicherheitsrisiken
func (v *JSONValidator) ValidateJSON(data []byte) error {
	// Größencheck
	if len(data) > 1024*1024 { // 1MB Limit
		return errors.New("JSON payload too large")
	}

	// Decoder mit Sicherheitseinstellungen
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var obj interface{}
	if err := decoder.Decode(&obj); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Strukturvalidierung
	return v.validateStructure(obj, 0)
}

// ValidateAndSanitize validiert und bereinigt JSON-Eingabe
func (v *JSONValidator) ValidateAndSanitize(data []byte) ([]byte, error) {
	// Erst validieren
	if err := v.ValidateJSON(data); err != nil {
		return nil, err
	}

	// Dann bereinigen
	sanitized, err := v.sanitizeJSON(data)
	if err != nil {
		return nil, err
	}

	return sanitized, nil
}

// validateStructure validiert die JSON-Struktur rekursiv
func (v *JSONValidator) validateStructure(obj interface{}, depth int) error {
	// Tiefencheck
	if depth > v.MaxDepth {
		return ErrJSONTooDeep
	}

	switch val := obj.(type) {
	case map[string]interface{}:
		return v.validateObject(val, depth)
	case []interface{}:
		return v.validateArray(val, depth)
	case string:
		return v.validateString(val)
	default:
		return nil
	}
}

// validateObject validiert JSON-Objekte
func (v *JSONValidator) validateObject(obj map[string]interface{}, depth int) error {
	// Anzahl Keys prüfen
	if len(obj) > v.MaxKeys {
		return ErrJSONTooManyKeys
	}

	// Objektgröße prüfen
	if len(obj) > v.MaxObjectSize {
		return ErrJSONObjectTooLarge
	}

	// Jedes Feld validieren
	for key, value := range obj {
		// Key-Länge prüfen
		if len(key) > v.MaxStringLen {
			return ErrJSONStringTooLong
		}

		// Malicious Patterns in Keys prüfen
		if v.isMaliciousKey(key) {
			return ErrJSONMalicious
		}

		// Wert rekursiv validieren
		if err := v.validateStructure(value, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// validateArray validiert JSON-Arrays
func (v *JSONValidator) validateArray(arr []interface{}, depth int) error {
	// Array-Größe prüfen
	if len(arr) > v.MaxArraySize {
		return ErrJSONArrayTooLarge
	}

	// Jedes Element validieren
	for _, item := range arr {
		if err := v.validateStructure(item, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// validateString validiert JSON-Strings
func (v *JSONValidator) validateString(str string) error {
	// String-Länge prüfen
	if len(str) > v.MaxStringLen {
		return ErrJSONStringTooLong
	}

	// Malicious Patterns prüfen
	if v.isMaliciousString(str) {
		return ErrJSONMalicious
	}

	return nil
}

// isMaliciousKey prüft auf verdächtige JSON-Keys
func (v *JSONValidator) isMaliciousKey(key string) bool {
	maliciousPatterns := []string{
		"__proto__",
		"constructor",
		"prototype",
		"valueOf",
		"toString",
		"../",
		"..\\",
		"<script",
		"javascript:",
		"data:",
		"vbscript:",
	}

	keyLower := strings.ToLower(key)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(keyLower, pattern) {
			return true
		}
	}

	return false
}

// isMaliciousString prüft auf verdächtige String-Inhalte
func (v *JSONValidator) isMaliciousString(str string) bool {
	maliciousPatterns := []string{
		"<script",
		"</script>",
		"javascript:",
		"data:text/html",
		"vbscript:",
		"onload=",
		"onerror=",
		"eval(",
		"Function(",
		"setTimeout(",
		"setInterval(",
		"document.cookie",
		"localStorage",
		"sessionStorage",
		"../",
		"..\\",
		"/etc/passwd",
		"/windows/system32",
		"cmd.exe",
		"powershell",
		"bash",
		"sh -c",
		"$((",
		"`",
		"${",
	}

	strLower := strings.ToLower(str)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(strLower, pattern) {
			return true
		}
	}

	return false
}

// sanitizeJSON bereinigt JSON-Daten
func (v *JSONValidator) sanitizeJSON(data []byte) ([]byte, error) {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, err
	}

	sanitized := v.sanitizeValue(obj)

	return json.Marshal(sanitized)
}

// sanitizeValue bereinigt einen JSON-Wert rekursiv
func (v *JSONValidator) sanitizeValue(obj interface{}) interface{} {
	switch val := obj.(type) {
	case map[string]interface{}:
		return v.sanitizeObject(val)
	case []interface{}:
		return v.sanitizeArray(val)
	case string:
		return v.sanitizeString(val)
	default:
		return val
	}
}

// sanitizeObject bereinigt JSON-Objekte
func (v *JSONValidator) sanitizeObject(obj map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range obj {
		// Malicious Keys ausschließen
		if v.isMaliciousKey(key) {
			continue
		}

		// Key bereinigen
		cleanKey := v.sanitizeString(key)
		if cleanKey == "" {
			continue
		}

		// Wert rekursiv bereinigen
		sanitized[cleanKey] = v.sanitizeValue(value)
	}

	return sanitized
}

// sanitizeArray bereinigt JSON-Arrays
func (v *JSONValidator) sanitizeArray(arr []interface{}) []interface{} {
	sanitized := make([]interface{}, 0, len(arr))

	for _, item := range arr {
		sanitizedItem := v.sanitizeValue(item)
		if sanitizedItem != nil {
			sanitized = append(sanitized, sanitizedItem)
		}
	}

	return sanitized
}

// sanitizeString bereinigt Strings
func (v *JSONValidator) sanitizeString(str string) string {
	// Entferne gefährliche Zeichen
	dangerous := []string{
		"<script",
		"</script>",
		"javascript:",
		"data:",
		"vbscript:",
		"`",
		"${",
		"$(",
		"../",
		"..\\",
	}

	result := str
	for _, pattern := range dangerous {
		result = strings.ReplaceAll(result, pattern, "")
	}

	// Begrenzen auf maximale Länge
	if len(result) > v.MaxStringLen {
		result = result[:v.MaxStringLen]
	}

	return result
}

// GetValidationStats gibt Validierungs-Statistiken zurück
func (v *JSONValidator) GetValidationStats(obj interface{}) map[string]interface{} {
	stats := make(map[string]interface{})

	depth, keys, strings, arrays := v.analyzeStructure(obj, 0)

	stats["max_depth"] = depth
	stats["total_keys"] = keys
	stats["total_strings"] = strings
	stats["total_arrays"] = arrays
	stats["limits"] = map[string]interface{}{
		"max_depth":      v.MaxDepth,
		"max_keys":       v.MaxKeys,
		"max_string_len": v.MaxStringLen,
		"max_array_size": v.MaxArraySize,
	}

	return stats
}

// analyzeStructure analysiert die JSON-Struktur
func (v *JSONValidator) analyzeStructure(obj interface{}, depth int) (int, int, int, int) {
	maxDepth := depth
	totalKeys := 0
	totalStrings := 0
	totalArrays := 0

	switch val := obj.(type) {
	case map[string]interface{}:
		totalKeys += len(val)
		for _, value := range val {
			d, k, s, a := v.analyzeStructure(value, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
			totalKeys += k
			totalStrings += s
			totalArrays += a
		}
	case []interface{}:
		totalArrays++
		for _, item := range val {
			d, k, s, a := v.analyzeStructure(item, depth+1)
			if d > maxDepth {
				maxDepth = d
			}
			totalKeys += k
			totalStrings += s
			totalArrays += a
		}
	case string:
		totalStrings++
	}

	return maxDepth, totalKeys, totalStrings, totalArrays
}
