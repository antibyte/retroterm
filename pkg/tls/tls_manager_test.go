package tls

import (
	"testing"

	"github.com/antibyte/retroterm/pkg/configuration"
)

func TestTLSManagerCreation(t *testing.T) {
	// Initialize configuration for testing
	if err := configuration.Initialize("../../settings.cfg"); err != nil {
		t.Skipf("Configuration file not found: %v", err)
	}

	// Test TLS manager creation with TLS disabled
	manager, err := NewTLSManager()
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	// TLS should be disabled by default
	if manager.IsEnabled() {
		t.Error("TLS should be disabled by default")
	}

	// HTTP port should be available
	if manager.GetHTTPPort() == "" {
		t.Error("HTTP port should be set")
	}
}

func TestTLSConfigValidation(t *testing.T) {
	// Test validation logic directly by creating a manager with invalid config
	config := &TLSConfig{
		EnableTLS:         true,
		EnableLetsEncrypt: true,
		Domain:            "", // Empty domain should cause validation error
		LetsEncryptEmail:  "test@example.com",
		CertCacheDir:      "./test_certs",
	}

	manager := &TLSManager{
		config: config,
	}

	// Test validation - should fail due to empty domain
	err := manager.validateConfig()
	if err == nil {
		t.Error("Expected validation error for empty domain, but got none")
	} else {
		t.Logf("Got expected validation error: %v", err)
	}

	// Test with empty email
	config.Domain = "test.com"
	config.LetsEncryptEmail = ""
	err = manager.validateConfig()
	if err == nil {
		t.Error("Expected validation error for empty email, but got none")
	} else {
		t.Logf("Got expected validation error: %v", err)
	}
}

func TestTLSManagerMethods(t *testing.T) {
	// Initialize configuration for testing
	if err := configuration.Initialize("../../settings.cfg"); err != nil {
		t.Skipf("Configuration file not found: %v", err)
	}

	manager, err := NewTLSManager()
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	// Test various methods
	if manager.GetTLSConfig() != nil && !manager.IsEnabled() {
		t.Error("TLS config should be nil when TLS is disabled")
	}

	if manager.GetHTTPHandler() != nil && !manager.config.EnableLetsEncrypt {
		t.Error("HTTP handler should be nil when Let's Encrypt is disabled")
	}

	// Test port getters
	httpPort := manager.GetHTTPPort()
	httpsPort := manager.GetHTTPSPort()

	if httpPort == "" {
		t.Error("HTTP port should not be empty")
	}

	if httpsPort == "" {
		t.Error("HTTPS port should not be empty")
	}

	// Test certificate file paths
	certFile, keyFile := manager.GetCertFiles()
	if certFile == "" || keyFile == "" {
		t.Error("Certificate file paths should not be empty")
	}
}

func TestTLSRedirectHandler(t *testing.T) {
	// Initialize configuration for testing
	if err := configuration.Initialize("../../settings.cfg"); err != nil {
		t.Skipf("Configuration file not found: %v", err)
	}

	manager, err := NewTLSManager()
	if err != nil {
		t.Fatalf("Failed to create TLS manager: %v", err)
	}

	// Test redirect handler
	redirectHandler := manager.GetHTTPSRedirectHandler()
	if redirectHandler != nil && !manager.config.ForceHTTPSRedirect {
		t.Error("Redirect handler should be nil when HTTPS redirect is disabled")
	}

	// Test NeedsHTTPServer
	needsHTTP := manager.NeedsHTTPServer()
	expectedNeedsHTTP := manager.config.EnableTLS && (manager.config.EnableLetsEncrypt || manager.config.ForceHTTPSRedirect)

	if needsHTTP != expectedNeedsHTTP {
		t.Errorf("NeedsHTTPServer() = %v, expected %v", needsHTTP, expectedNeedsHTTP)
	}
}
