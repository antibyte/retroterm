package tls

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"

	"golang.org/x/crypto/acme/autocert"
)

// TLSManager handles TLS certificate management including Let's Encrypt
type TLSManager struct {
	config      *TLSConfig
	autocertMgr *autocert.Manager
	tlsConfig   *tls.Config
	initialized bool
}

// TLSConfig holds TLS configuration options
type TLSConfig struct {
	EnableTLS          bool
	EnableLetsEncrypt  bool
	Domain             string
	LetsEncryptEmail   string
	CertCacheDir       string
	ForceHTTPSRedirect bool
	CertFile           string
	KeyFile            string
	HTTPPort           string
	HTTPSPort          string
}

// NewTLSManager creates a new TLS manager with configuration
func NewTLSManager() (*TLSManager, error) {
	config := &TLSConfig{
		EnableTLS:          configuration.GetBool("TLS", "enable_tls", false),
		EnableLetsEncrypt:  configuration.GetBool("TLS", "enable_letsencrypt", false),
		Domain:             configuration.GetString("TLS", "domain", ""),
		LetsEncryptEmail:   configuration.GetString("TLS", "letsencrypt_email", ""),
		CertCacheDir:       configuration.GetString("TLS", "cert_cache_dir", "./certs"),
		ForceHTTPSRedirect: configuration.GetBool("TLS", "force_https_redirect", false),
		CertFile:           configuration.GetString("TLS", "cert_file", "./certs/server.crt"),
		KeyFile:            configuration.GetString("TLS", "key_file", "./certs/server.key"),
		HTTPPort:           configuration.GetString("TLS", "http_port", "8080"),
		HTTPSPort:          configuration.GetString("TLS", "https_port", "8443"),
	}

	manager := &TLSManager{
		config: config,
	}

	if err := manager.validateConfig(); err != nil {
		return nil, fmt.Errorf("TLS configuration validation failed: %v", err)
	}

	if config.EnableTLS {
		if err := manager.initializeTLS(); err != nil {
			return nil, fmt.Errorf("TLS initialization failed: %v", err)
		}
	}

	return manager, nil
}

// validateConfig validates the TLS configuration
func (tm *TLSManager) validateConfig() error {
	if !tm.config.EnableTLS {
		return nil // No validation needed if TLS is disabled
	}
	if tm.config.EnableLetsEncrypt {
		if strings.TrimSpace(tm.config.Domain) == "" {
			return fmt.Errorf("domain is required when Let's Encrypt is enabled")
		}
		if strings.TrimSpace(tm.config.LetsEncryptEmail) == "" {
			return fmt.Errorf("letsencrypt_email is required when Let's Encrypt is enabled")
		}
		// Validate domain format
		if strings.Contains(tm.config.Domain, "example.com") {
			logger.SecurityWarn("Using example domain - change this in production!")
		}
	} else {
		// Manual certificate mode - check if files exist
		if _, err := os.Stat(tm.config.CertFile); os.IsNotExist(err) {
			logger.SecurityWarn("TLS certificate file not found: %s", tm.config.CertFile)
		}
		if _, err := os.Stat(tm.config.KeyFile); os.IsNotExist(err) {
			logger.SecurityWarn("TLS key file not found: %s", tm.config.KeyFile)
		}
	}

	return nil
}

// initializeTLS sets up TLS configuration
func (tm *TLSManager) initializeTLS() error {
	if tm.config.EnableLetsEncrypt {
		return tm.initializeLetsEncrypt()
	}
	return tm.initializeManualTLS()
}

// initializeLetsEncrypt sets up Let's Encrypt automatic certificate management
func (tm *TLSManager) initializeLetsEncrypt() error {
	logger.Info(logger.AreaSecurity, "Initializing Let's Encrypt for domain: %s", tm.config.Domain)

	// Ensure certificate cache directory exists
	if err := os.MkdirAll(tm.config.CertCacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create certificate cache directory: %v", err)
	}

	// Create autocert manager
	tm.autocertMgr = &autocert.Manager{
		Cache:      autocert.DirCache(tm.config.CertCacheDir),
		Prompt:     autocert.AcceptTOS,
		Email:      tm.config.LetsEncryptEmail,
		HostPolicy: autocert.HostWhitelist(tm.config.Domain, "www."+tm.config.Domain),
	}

	// Configure TLS with better error handling
	tm.tlsConfig = &tls.Config{
		GetCertificate: func(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// Log the server name for debugging
			logger.Debug("TLS certificate requested for: %s", clientHello.ServerName)

			// If no server name is provided, use the configured domain
			serverName := clientHello.ServerName
			if serverName == "" {
				logger.SecurityWarn("TLS handshake without SNI from %s, using default domain", clientHello.Conn.RemoteAddr())
				serverName = tm.config.Domain
			}

			// Validate the server name against our whitelist
			if serverName != tm.config.Domain && serverName != "www."+tm.config.Domain {
				logger.SecurityWarn("TLS request for unauthorized domain: %s from %s", serverName, clientHello.Conn.RemoteAddr())
				return nil, fmt.Errorf("unauthorized domain: %s", serverName)
			}

			// Use autocert manager to get certificate
			cert, err := tm.autocertMgr.GetCertificate(clientHello)
			if err != nil {
				logger.SecurityWarn("Failed to get certificate for %s: %v", serverName, err)
				return nil, fmt.Errorf("certificate error for %s: %v", serverName, err)
			}

			logger.Debug("Successfully provided certificate for: %s", serverName)
			return cert, nil
		},
		NextProtos: []string{"h2", "http/1.1"}, // Enable HTTP/2
		MinVersion: tls.VersionTLS12,
		// Prefer server cipher suites for better security
		PreferServerCipherSuites: true,
		// Disable session tickets for forward secrecy
		SessionTicketsDisabled: false,
	}

	tm.initialized = true
	logger.Info(logger.AreaSecurity, "Let's Encrypt TLS manager initialized successfully")
	return nil
}

// initializeManualTLS sets up manual certificate management
func (tm *TLSManager) initializeManualTLS() error {
	logger.Info(logger.AreaSecurity, "Initializing manual TLS with cert: %s, key: %s", tm.config.CertFile, tm.config.KeyFile)

	// Check if certificate files exist
	if _, err := os.Stat(tm.config.CertFile); os.IsNotExist(err) {
		return fmt.Errorf("certificate file not found: %s", tm.config.CertFile)
	}
	if _, err := os.Stat(tm.config.KeyFile); os.IsNotExist(err) {
		return fmt.Errorf("key file not found: %s", tm.config.KeyFile)
	}
	tm.initialized = true
	logger.Info(logger.AreaSecurity, "Manual TLS manager initialized successfully")
	return nil
}

// GetTLSConfig returns the TLS configuration for the HTTP server
func (tm *TLSManager) GetTLSConfig() *tls.Config {
	if !tm.initialized || !tm.config.EnableTLS {
		return nil
	}
	return tm.tlsConfig
}

// GetHTTPHandler returns an HTTP handler for Let's Encrypt challenges
func (tm *TLSManager) GetHTTPHandler() http.Handler {
	if tm.autocertMgr != nil {
		return tm.autocertMgr.HTTPHandler(nil)
	}
	return nil
}

// NeedsHTTPServer returns true if HTTP server is needed (for Let's Encrypt challenges or redirects)
func (tm *TLSManager) NeedsHTTPServer() bool {
	return tm.config.EnableTLS && (tm.config.EnableLetsEncrypt || tm.config.ForceHTTPSRedirect)
}

// GetHTTPSRedirectHandler returns a handler that redirects HTTP to HTTPS
func (tm *TLSManager) GetHTTPSRedirectHandler() http.Handler {
	if !tm.config.ForceHTTPSRedirect {
		return nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the host without port
		host := r.Host
		if strings.Contains(host, ":") {
			host = strings.Split(host, ":")[0]
		}

		// Construct HTTPS URL
		httpsURL := fmt.Sprintf("https://%s", host)
		if tm.config.HTTPSPort != "443" {
			httpsURL = fmt.Sprintf("https://%s:%s", host, tm.config.HTTPSPort)
		}
		httpsURL += r.RequestURI

		logger.Debug("Redirecting HTTP to HTTPS: %s -> %s", r.URL.String(), httpsURL)
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})
}

// IsEnabled returns true if TLS is enabled
func (tm *TLSManager) IsEnabled() bool {
	return tm.config.EnableTLS
}

// GetHTTPPort returns the HTTP port
func (tm *TLSManager) GetHTTPPort() string {
	return tm.config.HTTPPort
}

// GetHTTPSPort returns the HTTPS port
func (tm *TLSManager) GetHTTPSPort() string {
	return tm.config.HTTPSPort
}

// GetCertFiles returns the certificate and key file paths (for manual TLS)
func (tm *TLSManager) GetCertFiles() (string, string) {
	return tm.config.CertFile, tm.config.KeyFile
}

// GetDomain returns the configured domain (for connectivity testing)
func (tm *TLSManager) GetDomain() string {
	return tm.config.Domain
}

// GenerateSelfSignedCert generates a self-signed certificate for development
func (tm *TLSManager) GenerateSelfSignedCert() error {
	if tm.config.EnableLetsEncrypt {
		return fmt.Errorf("cannot generate self-signed certificate when Let's Encrypt is enabled")
	}

	logger.Info(logger.AreaSecurity, "Generating self-signed certificate for development")

	// This would be implemented with crypto/x509 for development use
	// For now, we log a warning that manual certificates are needed
	logger.SecurityWarn("Self-signed certificate generation not yet implemented")
	logger.SecurityWarn("Please provide manual certificates at: %s, %s", tm.config.CertFile, tm.config.KeyFile)

	return fmt.Errorf("self-signed certificate generation not yet implemented")
}
