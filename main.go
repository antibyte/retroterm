package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/auth"
	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/terminal"
	"github.com/antibyte/retroterm/pkg/tinyos"
	tlsmanager "github.com/antibyte/retroterm/pkg/tls"
	"github.com/antibyte/retroterm/pkg/virtualfs"
)

func main() { // Initialize configuration (before all other initializations)
	configPath := "settings.cfg"
	err := configuration.Initialize(configPath)
	if err != nil {
		fmt.Printf("Error initializing configuration: %v\n", err)
		return
	}

	// Initialize logger
	err = logger.Initialize()
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		return
	}
	defer logger.Close()
	// First log message
	logger.ConfigInfo("System started - Configuration loaded from: %s", configPath)

	// Read log file from configuration (for compatibility)
	logFilePath := configuration.GetString("Debug", "log_file", "debug.log")

	// On program start, delete all previous standard log entries (not the new structured logs)
	// os.Remove(logFilePath) // Entfernt - wir nutzen jetzt Log-Rotation

	// Open log file in overwrite mode
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer logFile.Close() // Check if legacy logging should be disabled (multiplatform solution)
	disableLegacyLogging := configuration.GetBool("Debug", "disable_legacy_logging", false)

	// TEMPORARY FIX: Disable the stdout/stderr redirection that's causing server crashes
	// TODO: Implement safer logging redirection later
	if disableLegacyLogging {
		// Only redirect log.Printf to discard, leave stdout/stderr alone
		log.SetOutput(io.Discard)
		// Confirmation message to terminal
		fmt.Println("Legacy logging disabled. Using structured logging only.")
	} else {
		// Normal logging to file - but don't redirect stdout/stderr for now
		log.SetOutput(logFile)

		// Confirmation message to terminal
		fmt.Println("Log outputs are redirected to debug.log.")
		// Better startup message with timestamp (now in log file)
		log.Printf("=== SERVER START %s ===", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("Log redirection activated. Terminal outputs are saved in debug.log.")
	}
	// Database initialization
	db, err := tinyos.InitDB("tinyos.db")
	if err != nil {
		logger.Fatal(logger.AreaDatabase, "Database initialization failed: %v", err)
	}
	defer db.Close()

	// Create tables if they don't exist
	if err := tinyos.CreateTables(db); err != nil {
		logger.Fatal(logger.AreaDatabase, "Table creation failed: %v", err)
	}
	logger.Info(logger.AreaDatabase, "Database tables successfully initialized")

	// Create default users (including Dyson account)
	if err := tinyos.CreateDefaultUsers(db); err != nil {
		logger.Fatal(logger.AreaDatabase, "Default user creation failed: %v", err)
	} // Initialize virtual filesystem
	vfs := virtualfs.New(db)
	logger.Info(logger.AreaFileSystem, "Virtual filesystem initialized: %p", vfs)

	// Initialize PromptManager (loads all prompts at startup)
	promptManager, err := shared.NewPromptManager()
	if err != nil {
		logger.Fatal(logger.AreaGeneral, "Error initializing PromptManager: %v", err)
	}
	logger.Info(logger.AreaGeneral, "PromptManager successfully initialized")

	// Initialize TinyOS
	tinyOSInstance := tinyos.NewTinyOS(vfs, promptManager)
	logger.Info(logger.AreaGeneral, "TinyOS initialized: %p", tinyOSInstance)

	// Create TerminalHandler without global TinyBASIC instance
	// Each session gets its own TinyBASIC instance
	handler := terminal.NewTerminalHandler(tinyOSInstance)
	logger.Info(logger.AreaTerminal, "TerminalHandler created with OS=%p (session-based TinyBASIC instances)", tinyOSInstance)

	// Ensure VFS knows about the TinyOS provider (for user context operations)
	vfs.SetTinyOSProvider(tinyOSInstance)
	logger.Info(logger.AreaFileSystem, "TinyOS provider set for VFS")

	// Set TinyOS instance for auth handlers
	auth.SetTinyOSInstance(tinyOSInstance)
	logger.Info(logger.AreaAuth, "TinyOS instance set for auth handlers")
	// Configure HTTP handlers
	// Authentication API routes
	http.HandleFunc("/api/auth/session", auth.HandleCreateSession)
	http.HandleFunc("/api/auth/login", auth.HandleLogin)
	http.HandleFunc("/api/auth/validate", auth.HandleTokenValidation)
	http.HandleFunc("/api/auth/logout", auth.HandleLogout)
	http.HandleFunc("/ws", handler.HandleWebSocket)
	http.HandleFunc("/chat", handler.HandleChatWebSocket) // Chat WebSocket Route	http.HandleFunc("/cleanup-guest", handler.CleanupGuestSession) // Guest VFS cleanup endpoint
	// API route for SID files
	http.HandleFunc("/api/file", serveUserFile(handler))

	// Static file handlers for assets
	http.HandleFunc("/floppy.mp3", serveFile("assets/floppy.mp3"))
	http.HandleFunc("/background.png", serveFile("assets/background.png"))

	// Add favicon handler to prevent 404 errors
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r) // Return 404 but don't log it as error
	})
	// Static file servers for directories
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))

	// Legacy individual file handlers for backwards compatibility (development)
	http.HandleFunc("/retroterminal.css", serveFile("css/retroterminal.css"))
	http.HandleFunc("/auth_tests.html", serveFile("auth_tests.html")) // Root-Route - MUST be registered LAST to not override specific routes
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Debug(logger.AreaGeneral, "ROOT REQUEST: %s %s", r.Method, r.URL.Path)

		if r.URL.Path == "/" {
			// Try production HTML file first (from build system)
			if _, err := os.Stat("index.html"); err == nil {
				http.ServeFile(w, r, "index.html")
			} else if _, err := os.Stat("retroterminal.html"); err == nil {
				// Fallback to development HTML file
				http.ServeFile(w, r, "retroterminal.html")
			} else {
				logger.Error(logger.AreaGeneral, "Neither index.html nor retroterminal.html found")
				http.Error(w, "Main HTML file not found", http.StatusNotFound)
			}
		} else {
			logger.Debug(logger.AreaGeneral, "ROOT ROUTE: 404 for path: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
	// Initialize TLS Manager
	tlsManager, err := tlsmanager.NewTLSManager()
	if err != nil {
		logger.Fatal(logger.AreaSecurity, "TLS manager initialization failed: %v", err)
		return
	}
	// Start servers based on TLS configuration
	if tlsManager.IsEnabled() {
		startTLSServers(tlsManager)
	} else {
		startHTTPServer(tlsManager.GetHTTPPort())
	}
}

// startHTTPServer starts the HTTP server
func startHTTPServer(port string) {
	logger.Info(logger.AreaGeneral, "Starting HTTP server on port %s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Error(logger.AreaGeneral, "HTTP server startup failed: %v", err)
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}

// startTLSServers starts both HTTP and HTTPS servers for TLS mode
func startTLSServers(tlsManager *tlsmanager.TLSManager) {
	httpPort := tlsManager.GetHTTPPort()
	httpsPort := tlsManager.GetHTTPSPort()

	logger.Info(logger.AreaSecurity, "Starting TLS-enabled servers - HTTP: %s, HTTPS: %s", httpPort, httpsPort)

	// Channel to receive errors from server goroutines
	errorChan := make(chan error, 2)

	// Start HTTP server for Let's Encrypt challenges and redirects (if needed)
	if tlsManager.NeedsHTTPServer() {
		go func() {
			httpHandler := tlsManager.GetHTTPHandler()
			if httpHandler == nil {
				// Use HTTPS redirect handler if no Let's Encrypt handler
				httpHandler = tlsManager.GetHTTPSRedirectHandler()
			}

			if httpHandler != nil {
				logger.Info(logger.AreaSecurity, "Starting HTTP server for Let's Encrypt challenges/redirects on port %s", httpPort)
				if err := http.ListenAndServe(":"+httpPort, httpHandler); err != nil {
					logger.Error(logger.AreaSecurity, "HTTP server error: %v", err)
					errorChan <- fmt.Errorf("HTTP server error: %v", err)
				}
			}
		}()
	} // Start HTTPS server
	go func() {
		httpsServer := &http.Server{
			Addr:      ":" + httpsPort,
			TLSConfig: tlsManager.GetTLSConfig(),
			Handler:   nil, // Use default mux with all registered handlers
		}

		logger.Info(logger.AreaSecurity, "Starting HTTPS server on port %s", httpsPort)

		var err error
		if tlsManager.GetTLSConfig() != nil {
			// Let's Encrypt mode
			logger.Info(logger.AreaSecurity, "HTTPS server using Let's Encrypt certificates")
			err = httpsServer.ListenAndServeTLS("", "")
		} else {
			// Manual certificate mode
			certFile, keyFile := tlsManager.GetCertFiles()
			logger.Info(logger.AreaSecurity, "HTTPS server using manual certificates: %s, %s", certFile, keyFile)
			err = httpsServer.ListenAndServeTLS(certFile, keyFile)
		}

		// This should only execute if ListenAndServeTLS returns (indicating an error)
		logger.Error(logger.AreaSecurity, "HTTPS server ListenAndServeTLS returned with error: %v", err)
		errorChan <- fmt.Errorf("HTTPS server stopped unexpectedly: %v", err)
	}() // Wait for either server to report an error
	select {
	case err := <-errorChan:
		logger.Fatal(logger.AreaSecurity, "Server startup failed: %v", err)
	case <-time.After(5 * time.Second):
		// If no errors after 5 seconds, consider startup successful
		logger.Info(logger.AreaSecurity, "TLS servers startup window completed - HTTP: %s, HTTPS: %s", httpPort, httpsPort) // Test HTTPS connectivity (only for manual TLS)
		go func() {
			time.Sleep(1 * time.Second) // Give server a moment to fully bind

			if tlsManager.GetTLSConfig() != nil {
				// Let's Encrypt mode - skip connectivity test as domain validation is handled by Let's Encrypt
				logger.Info(logger.AreaSecurity, "HTTPS server ready with Let's Encrypt certificates for domain: %s", tlsManager.GetDomain())
			} else {
				// Manual TLS mode - test localhost connectivity
				testURL := fmt.Sprintf("https://localhost:%s/", httpsPort)
				logger.Debug(logger.AreaSecurity, "Testing HTTPS connectivity to %s", testURL)

				client := &http.Client{
					Timeout: 10 * time.Second,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip cert verification for localhost test
					},
				}

				resp, err := client.Get(testURL)
				if err != nil {
					logger.Warn(logger.AreaSecurity, "HTTPS connectivity test failed: %v", err)
				} else {
					resp.Body.Close()
					logger.Info(logger.AreaSecurity, "HTTPS connectivity test successful (status: %s)", resp.Status)
				}
			}
		}()

		// Now wait indefinitely for errors (blocking the main thread)
		for {
			err := <-errorChan
			logger.Error(logger.AreaSecurity, "Server error during runtime: %v", err)
		}
	}
}

// serveFile serves a single file with the correct MIME type
func serveFile(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if file exists
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			logger.Debug(logger.AreaGeneral, "File not found: %s", filename)
			http.NotFound(w, r)
			return
		}

		// Determine MIME type based on file extension (safer string handling)
		var contentType string
		lowerFilename := strings.ToLower(filename)
		switch {
		case strings.HasSuffix(lowerFilename, ".mp3"):
			contentType = "audio/mpeg"
		case strings.HasSuffix(lowerFilename, ".ogg"):
			contentType = "audio/ogg"
		case strings.HasSuffix(lowerFilename, ".css"):
			contentType = "text/css; charset=utf-8"
		case strings.HasSuffix(lowerFilename, ".html"):
			contentType = "text/html; charset=utf-8"
		case strings.HasSuffix(lowerFilename, ".js"):
			contentType = "application/javascript; charset=utf-8"
		case strings.HasSuffix(lowerFilename, ".png"):
			contentType = "image/png"
		case strings.HasSuffix(lowerFilename, ".jpg") || strings.HasSuffix(lowerFilename, ".jpeg"):
			contentType = "image/jpeg"
		default:
			contentType = "application/octet-stream"
		}

		// Set Content-Type header
		w.Header().Set("Content-Type", contentType)

		// Serve file
		http.ServeFile(w, r, filename)
	}
}

// serveUserFile creates a handler for user files
func serveUserFile(handler *terminal.TerminalHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		} // Read filename from query parameter
		filename := r.URL.Query().Get("path")
		if filename == "" {
			http.Error(w, "Missing path parameter", http.StatusBadRequest)
			return
		}
		logger.Info(logger.AreaGeneral, "API File request for: %s", filename)

		// Try to read file from virtual filesystem
		content, err := os.ReadFile(filename)
		if err != nil {
			logger.Debug(logger.AreaGeneral, "File not found in user directory: %s, error: %v", filename, err)
			// Fallback: Try to read from examples directory
			if strings.HasSuffix(strings.ToLower(filename), ".sid") || strings.HasSuffix(strings.ToLower(filename), ".bas") { // Try first with exact name
				examplePath := "examples/" + filename
				if data, readErr := os.ReadFile(examplePath); readErr == nil {
					logger.Debug(logger.AreaGeneral, "Found file in examples directory: %s", examplePath)
					// Set MIME type for SID files
					if strings.HasSuffix(strings.ToLower(filename), ".sid") {
						w.Header().Set("Content-Type", "application/octet-stream")
					} else {
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
					}
					w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))
					w.Write(data)
					return
				}

				// Try with .sid extension if not present
				if !strings.HasSuffix(strings.ToLower(filename), ".sid") {
					examplePathSid := "examples/" + filename + ".sid"
					if data, readErr := os.ReadFile(examplePathSid); readErr == nil {
						logger.Debug(logger.AreaGeneral, "Found file in examples directory with .sid extension: %s", examplePathSid)
						w.Header().Set("Content-Type", "application/octet-stream")
						w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename+".sid"))
						w.Write(data)
						return
					}
				}

				logger.Debug(logger.AreaGeneral, "File not found in examples directory either: %s", examplePath)
			}

			logger.Debug(logger.AreaGeneral, "Error reading file %s: %v", filename, err)
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Determine MIME type based on file extension
		var contentType string
		lowerName := strings.ToLower(filename)
		switch {
		case strings.HasSuffix(lowerName, ".sid"):
			contentType = "application/octet-stream"
		case strings.HasSuffix(lowerName, ".bas"):
			contentType = "text/plain; charset=utf-8"
		case strings.HasSuffix(lowerName, ".txt"):
			contentType = "text/plain; charset=utf-8"
		default:
			contentType = "application/octet-stream"
		}

		// Headers setzen
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filename))

		// Serve file
		w.Write([]byte(content))
	}
}
