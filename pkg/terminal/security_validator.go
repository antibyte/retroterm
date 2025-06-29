package terminal

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/antibyte/retroterm/pkg/configuration"
)

// SecurityValidator bietet Sicherheitsvalidierung für Chat-Eingaben
type SecurityValidator struct{}

// NewSecurityValidator erstellt einen neuen SecurityValidator
func NewSecurityValidator() *SecurityValidator {
	return &SecurityValidator{}
}

// ValidateChatContent überprüft Chat-Nachrichten auf schädliche Inhalte
func (sv *SecurityValidator) ValidateChatContent(content string) error {
	if len(content) == 0 {
		return nil
	}

	// Länge aus Konfiguration begrenzen
	maxLength := configuration.GetInt("Security", "max_message_length", 1000)
	if len(content) > maxLength {
		return fmt.Errorf("message too long: maximum %d characters allowed", maxLength)
	}

	// SQL-Injection-Pattern prüfen (wenn aktiviert)
	if configuration.GetBool("Security", "enable_sql_injection_filter", true) {
		if err := sv.checkSQLInjection(content); err != nil {
			return err
		}
	}

	// XSS-Pattern prüfen (wenn aktiviert)
	if configuration.GetBool("Security", "enable_xss_filter", true) {
		if err := sv.checkXSSPatterns(content); err != nil {
			return err
		}
	}

	// Command-Injection-Pattern prüfen (wenn aktiviert)
	if configuration.GetBool("Security", "enable_command_injection_filter", true) {
		if err := sv.checkCommandInjection(content); err != nil {
			return err
		}
	}

	// AI-Prompt-Injection prüfen
	if err := sv.checkAIPromptInjection(content); err != nil {
		return err
	}

	return nil
}

// checkSQLInjection prüft auf SQL-Injection-Versuche
func (sv *SecurityValidator) checkSQLInjection(content string) error {
	// Nur sehr verdächtige SQL-Patterns prüfen, die wahrscheinlich Injection-Versuche sind
	suspiciousPatterns := []string{
		"' OR '1'='1", "' OR 1=1", "\" OR \"1\"=\"1", "\" OR 1=1",
		"'; DROP TABLE", "'; DELETE FROM", "'; INSERT INTO", "'; UPDATE",
		"UNION SELECT", "UNION ALL SELECT",
		"xp_cmdshell", "sp_executesql",
		"@@version", "@@servername",
		"CHAR(", "CAST(", "CONVERT(",
		"--", "/**/", "/*!",
		"' UNION", "\" UNION",
		"OR SLEEP(", "AND SLEEP(",
		"BENCHMARK(", "EXTRACTVALUE(",
		"UPDATEXML(", "NAME_CONST(",
	}

	contentUpper := strings.ToUpper(content)

	// Prüfe nur sehr spezifische SQL-Injection-Patterns
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(contentUpper, strings.ToUpper(pattern)) {
			return fmt.Errorf("potentially malicious SQL pattern detected: %s", pattern)
		}
	}

	// Prüfe auf verdächtige Kombinationen von Zeichen
	if strings.Contains(content, "'") && strings.Contains(contentUpper, " OR ") {
		if strings.Contains(content, "=") {
			return fmt.Errorf("potentially malicious SQL injection pattern detected")
		}
	}

	return nil
}

// containsWholeWord prüft, ob ein Wort als ganzes Wort vorkommt (nicht als Teil eines anderen Wortes)
// checkXSSPatterns prüft auf XSS-Versuche
func (sv *SecurityValidator) checkXSSPatterns(content string) error {
	xssPatterns := []string{
		"<script", "</script>", "javascript:", "onclick=", "onerror=",
		"onload=", "onmouseover=", "eval(", "document.cookie",
		"window.location", "innerHTML", "document.write",
		"<iframe", "<object", "<embed", "<form", "vbscript:",
	}

	contentLower := strings.ToLower(content)
	for _, pattern := range xssPatterns {
		if strings.Contains(contentLower, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially malicious XSS pattern detected: %s", pattern)
		}
	}
	return nil
}

// checkCommandInjection prüft auf Command-Injection-Versuche
func (sv *SecurityValidator) checkCommandInjection(content string) error {
	cmdPatterns := []string{
		"|", "&", ";", "`", "$(", "${", "||", "&&",
		">/", "</", ">>", "<<", "curl ", "wget ", "nc ",
		"netcat", "bash", "/bin/", "cmd.exe", "powershell",
	}

	for _, pattern := range cmdPatterns {
		if strings.Contains(content, pattern) {
			return fmt.Errorf("potentially malicious command pattern detected: %s", pattern)
		}
	}
	return nil
}

// checkAIPromptInjection prüft auf AI-Prompt-Injection-Versuche
func (sv *SecurityValidator) checkAIPromptInjection(content string) error {
	aiPatterns := []string{
		"IGNORE PREVIOUS INSTRUCTIONS", "IGNORE ALL PREVIOUS",
		"SYSTEM:", "ASSISTANT:", "USER:", "HUMAN:", "AI:",
		"JAILBREAK", "PRETEND YOU ARE", "ROLEPLAY AS",
		"FORGET EVERYTHING", "NEW INSTRUCTIONS:",
		"OVERRIDE", "ADMIN MODE", "DEBUG MODE",
		"```", "---END---", "---START---",
	}

	contentUpper := strings.ToUpper(content)
	for _, pattern := range aiPatterns {
		if strings.Contains(contentUpper, strings.ToUpper(pattern)) {
			return fmt.Errorf("potentially malicious AI prompt injection detected: %s", pattern)
		}
	}
	return nil
}

// SanitizeForDeepSeek bereinigt Eingaben für die DeepSeek-API
func (sv *SecurityValidator) SanitizeForDeepSeek(input string) string {
	if len(input) == 0 {
		return input
	}

	// Länge begrenzen
	if len(input) > 1000 {
		input = input[:1000] + "..."
	}

	// Gefährliche AI-Prompt-Injection-Pattern entfernen
	dangerous := map[string]string{
		"IGNORE PREVIOUS INSTRUCTIONS": "[FILTERED]",
		"IGNORE ALL PREVIOUS":          "[FILTERED]",
		"SYSTEM:":                      "[FILTERED]",
		"ASSISTANT:":                   "[FILTERED]",
		"USER:":                        "[FILTERED]",
		"HUMAN:":                       "[FILTERED]",
		"AI:":                          "[FILTERED]",
		"JAILBREAK":                    "[FILTERED]",
		"PRETEND YOU ARE":              "[FILTERED]",
		"ROLEPLAY AS":                  "[FILTERED]",
		"FORGET EVERYTHING":            "[FILTERED]",
		"NEW INSTRUCTIONS:":            "[FILTERED]",
		"OVERRIDE":                     "[FILTERED]",
		"ADMIN MODE":                   "[FILTERED]",
		"DEBUG MODE":                   "[FILTERED]",
		"```":                          "[CODE]",
		"---END---":                    "[FILTERED]",
		"---START---":                  "[FILTERED]",
	}

	// Case-insensitive Ersetzung
	for pattern, replacement := range dangerous {
		input = strings.ReplaceAll(strings.ToUpper(input), strings.ToUpper(pattern), replacement)
	}

	// Kontrollzeichen entfernen (außer normalen Whitespace)
	var result strings.Builder
	for _, r := range input {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}

// ValidateSessionID prüft die Gültigkeit einer Session-ID
func (sv *SecurityValidator) ValidateSessionID(sessionID string) error {
	if len(sessionID) == 0 {
		return fmt.Errorf("session ID is empty")
	}

	if len(sessionID) > 128 {
		return fmt.Errorf("session ID too long")
	}

	// Nur alphanumerische Zeichen und Bindestriche erlauben
	for _, r := range sessionID {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return fmt.Errorf("session ID contains invalid characters")
		}
	}

	return nil
}
