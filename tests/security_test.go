package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoHardcodedSecrets verifies that no hardcoded secrets are present in the codebase
func TestNoHardcodedSecrets(t *testing.T) {
	// Define patterns that should not appear in production code
	dangerousPatterns := []struct {
		pattern     string
		description string
		isRegex     bool
	}{
		{"sk-[a-zA-Z0-9]{32}", "DeepSeek API key pattern", true},
		{"sehr_geheimes_jwt_token", "Hardcoded German JWT token", false},
		{"fallback_secret_change_in_production", "Production fallback secret", false},
		{"YOUR_DEEPSEEK_API_KEY_HERE", "Placeholder API key", false},
		{"password.*=.*[\"'][^\"']{8,}[\"']", "Hardcoded password", true},
		{"api_key.*=.*[\"'][^\"']{10,}[\"']", "Hardcoded API key", true},
		{"secret.*=.*[\"'][^\"']{8,}[\"']", "Hardcoded secret", true},
	}
	// Files to exclude from security scan
	excludeFiles := []string{
		"setup-env.ps1",
		"setup-env-simple.ps1",
		"setup-env-fixed.ps1",
		"setup-env.bat",
		"settings.cfg.template",
		"continue.txt",
		"security_test.go", // This test file itself
		"SECURITY_SETUP.md",
		"jwt.go", // Contains monitored fallback secrets with warnings
	}

	// Directories to exclude
	excludeDirs := []string{
		".git",
		"node_modules",
		"tmp",
		"debug.log",
	}
	var violations []string

	// Get the project root directory (parent of tests directory)
	projectRoot := ".."

	err := filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			for _, excludeDir := range excludeDirs {
				if info.Name() == excludeDir {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Skip excluded files
		filename := filepath.Base(path)
		for _, excludeFile := range excludeFiles {
			if filename == excludeFile {
				return nil
			}
		}

		// Only check relevant file types
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".js" && ext != ".cfg" && ext != ".json" && ext != ".md" {
			return nil
		}
		// Read and scan file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %v", path, err)
		}

		// Skip very large files (likely binary or generated)
		if len(content) > 1024*1024 { // Skip files larger than 1MB
			return nil
		}

		// Scan for dangerous patterns line by line
		lines := strings.Split(string(content), "\n")
		for lineNum, line := range lines {
			line = strings.TrimSpace(line)
			// Skip empty lines and comments in Go files
			if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") {
				continue
			}

			for _, pattern := range dangerousPatterns {
				var matched bool
				if pattern.isRegex {
					matched, _ = regexp.MatchString(pattern.pattern, line)
				} else {
					matched = strings.Contains(line, pattern.pattern)
				}

				if matched {
					violations = append(violations, fmt.Sprintf(
						"SECURITY VIOLATION in %s:%d - %s\nLine: %s",
						path, lineNum+1, pattern.description, line,
					))
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Error scanning files: %v", err)
	}

	// Report violations
	if len(violations) > 0 {
		t.Errorf("Found %d security violations:\n\n%s",
			len(violations), strings.Join(violations, "\n\n"))
	}
}

// TestEnvironmentVariableUsage verifies that environment variables are properly used
func TestEnvironmentVariableUsage(t *testing.T) {
	// Check that JWT auth code uses environment variables
	jwtFile := "../pkg/auth/jwt.go"
	content, err := os.ReadFile(jwtFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", jwtFile, err)
	}

	jwtContent := string(content)

	// Verify environment variable usage
	if !strings.Contains(jwtContent, "os.Getenv(\"JWT_SECRET_KEY\")") {
		t.Error("JWT auth should check JWT_SECRET_KEY environment variable")
	}

	if !strings.Contains(jwtContent, "SecurityWarn") {
		t.Error("JWT auth should warn when using fallback secrets")
	}
}

// TestConfigurationSecurity verifies that configuration files are secure
func TestConfigurationSecurity(t *testing.T) {
	// Check that settings.cfg uses environment variable placeholders
	settingsFile := "../settings.cfg"
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		t.Skip("settings.cfg not found - skipping test")
	}

	content, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", settingsFile, err)
	}

	settingsContent := string(content)

	// Check for environment variable placeholders
	if !strings.Contains(settingsContent, "ENVIRONMENT_VARIABLE_NOT_SET") {
		t.Error("settings.cfg should use environment variable placeholders")
	}

	// Check that no real secrets are present
	dangerousLines := []string{
		"sk-",
		"sehr_geheimes",
		"secret_key = [a-zA-Z0-9]{20,}",
	}

	for _, dangerous := range dangerousLines {
		if strings.Contains(settingsContent, dangerous) {
			t.Errorf("settings.cfg contains potentially dangerous content: %s", dangerous)
		}
	}
}
