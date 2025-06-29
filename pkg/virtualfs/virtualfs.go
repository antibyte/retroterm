package virtualfs

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
)

// Helper function for VFS debug logging that respects configuration
func vfsDebugLog(format string, args ...interface{}) {
	logger.Debug(logger.AreaFileSystem, format, args...)
}

// Filesystem-Limits für Benutzer-Sicherheit - werden jetzt aus der Konfiguration gelesen
// Siehe [FileSystem] Sektion in settings.cfg

// VirtualFile repräsentiert eine Datei oder ein Verzeichnis im VFS
type VirtualFile struct {
	Name     string
	IsDir    bool
	Content  []byte
	ModTime  time.Time
	Children map[string]*VirtualFile
	Parent   *VirtualFile
}

// TinyOSProvider ist eine Schnittstelle für den Zugriff auf TinyOS-Funktionen
type TinyOSProvider interface {
	Username() string // Gibt den aktuellen Benutzernamen zurück
}

// VFS repräsentiert das virtuelle Dateisystem
type VFS struct {
	root      *VirtualFile
	mu        sync.RWMutex
	db        *sql.DB                 // Datenbank-Verbindung hinzugefügt
	os        TinyOSProvider          // Referenz auf TinyOS für den Zugriff auf Systemfunktionen
	userRoots map[string]*VirtualFile // Map für benutzerspezifische Root-Verzeichnisse
	// Hier könnten später Datenbank-Hooks oder Persistenzlogik hinzukommen
}

// UserStats enthält Statistiken über die VFS-Nutzung eines Benutzers
type UserStats struct {
	Username           string
	DirectoryCount     int
	MaxDirectories     int
	HomeDirectoryFiles int
	MaxFilesPerDir     int
	TotalFiles         int
}

// UserStorageInfo contains storage usage information for a user
type UserStorageInfo struct {
	UsedKB  int
	TotalKB int
}

// New erstellt ein neues virtuelles Dateisystem
func New(db *sql.DB) *VFS {
	// Erstelle nur das Root-Verzeichnis und die Verzeichnisse für /home und /system
	root := &VirtualFile{
		Name:     "/",
		IsDir:    true,
		ModTime:  time.Now(),
		Children: make(map[string]*VirtualFile),
		Parent:   nil, // Root hat kein Parent
	}

	// Erstelle direkt das /home-Verzeichnis
	homeDir := &VirtualFile{
		Name:     "home",
		IsDir:    true,
		ModTime:  time.Now(),
		Children: make(map[string]*VirtualFile),
		Parent:   root,
	}
	root.Children["home"] = homeDir

	// Erstelle direkt das /system-Verzeichnis
	systemDir := &VirtualFile{
		Name:     "system",
		IsDir:    true,
		ModTime:  time.Now(),
		Children: make(map[string]*VirtualFile),
		Parent:   root,
	}
	root.Children["system"] = systemDir

	vfs := &VFS{
		root:      root,
		db:        db,
		os:        nil,                           // Wird später gesetzt
		userRoots: make(map[string]*VirtualFile), // Initialisierung der Benutzer-Roots Map
	}

	return vfs
}

// Filesystem-Limit-Überprüfungsfunktionen für Sicherheit

// CountUserDirectories zählt die Anzahl der Verzeichnisse für einen Benutzer (rekursiv) - PUBLIC
func (vfs *VFS) CountUserDirectories(username string) int {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	userHomePath := "/home/" + username
	userHome, _, err := vfs.resolvePathInternalWithoutLock(userHomePath)
	if err != nil {
		return 0
	}

	return vfs.countDirectoriesRecursive(userHome)
}

// countDirectoriesRecursive zählt alle Verzeichnisse rekursiv in einem Baum
func (vfs *VFS) countDirectoriesRecursive(node *VirtualFile) int {
	if node == nil || !node.IsDir {
		return 0
	}

	count := 0
	for _, child := range node.Children {
		if child.IsDir {
			count++                                       // Zähle das Verzeichnis selbst
			count += vfs.countDirectoriesRecursive(child) // Rekursive Zählung
		}
	}
	return count
}

// countFilesInDirectory zählt die Anzahl der Dateien (nicht Verzeichnisse) in einem Verzeichnis
func (vfs *VFS) countFilesInDirectory(dir *VirtualFile) int {
	if dir == nil || !dir.IsDir {
		return 0
	}

	count := 0
	for _, child := range dir.Children {
		if !child.IsDir {
			count++
		}
	}
	return count
}

// checkUserDirectoryLimit überprüft, ob ein Benutzer das Verzeichnis-Limit erreicht hat
func (vfs *VFS) checkUserDirectoryLimit(username string) error {
	currentDirCount := vfs.CountUserDirectories(username)
	maxUserDirectories := configuration.GetInt("FileSystem", "max_directories_per_user", 20)
	if currentDirCount >= maxUserDirectories {
		return fmt.Errorf("user directory limit exceeded: %d/%d directories (max %d allowed)",
			currentDirCount, maxUserDirectories, maxUserDirectories)
	}
	return nil
}

// checkUserDirectoryLimitWithoutLock überprüft, ob ein Benutzer das Verzeichnis-Limit erreicht hat (ohne Lock)
func (vfs *VFS) checkUserDirectoryLimitWithoutLock(username string) error {
	currentDirCount := vfs.countUserDirectoriesLocked(username)
	maxUserDirectories := configuration.GetInt("FileSystem", "max_directories_per_user", 20)
	if currentDirCount >= maxUserDirectories {
		return fmt.Errorf("user directory limit exceeded: %d/%d directories (max %d allowed)",
			currentDirCount, maxUserDirectories, maxUserDirectories)
	}
	return nil
}

// checkDirectoryFileLimit überprüft, ob ein Verzeichnis das Datei-Limit erreicht hat
func (vfs *VFS) checkDirectoryFileLimit(dir *VirtualFile) error {
	currentFileCount := vfs.countFilesInDirectory(dir)
	maxFilesPerDirectory := configuration.GetInt("FileSystem", "max_files_per_directory", 100)
	if currentFileCount >= maxFilesPerDirectory {
		return fmt.Errorf("directory file limit exceeded: %d/%d files (max %d allowed)",
			currentFileCount, maxFilesPerDirectory, maxFilesPerDirectory)
	}
	return nil
}

// checkFileSizeLimit überprüft, ob eine Datei das Größen-Limit überschreitet
func (vfs *VFS) checkFileSizeLimit(content string) error {
	contentSize := len([]byte(content))
	maxFileSize := configuration.GetInt("FileSystem", "max_file_size_kb", 1024) * 1024 // KB zu Bytes
	if contentSize > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes / %.1f KB allowed)",
			contentSize, maxFileSize, float64(maxFileSize)/1024)
	}
	return nil
}

// getUsernameFromPath extrahiert den Benutzernamen aus einem Pfad (z.B. /home/user/file -> user)
func (vfs *VFS) getUsernameFromPath(path string) string {
	if !strings.HasPrefix(path, "/home/") {
		return "" // Nicht in einem Benutzerverzeichnis
	}

	pathParts := strings.Split(strings.TrimPrefix(path, "/home/"), "/")
	if len(pathParts) > 0 && pathParts[0] != "" {
		return pathParts[0]
	}
	return ""
}

// InitializeUserVFS initialisiert das VFS für einen bestimmten Benutzer
// Lädt nur die Dateien des angegebenen Benutzers aus der Datenbank
func (vfs *VFS) InitializeUserVFS(username string) error {
	if username == "" {
		return fmt.Errorf("Username may not be empty")
	}

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Create the home directory for the user
	homePath := "/home/" + username

	// Check if the user already has a home directory
	_, exists := vfs.userRoots[username]
	if exists {
		vfsDebugLog("User VFS is already initialized: %s", username)
		return nil
	}

	// Create the user's home directory in the directory structure
	err := vfs.createDirectoryInMemory(homePath)
	if err != nil {
		return fmt.Errorf("error creating home directory: %v", err)
	}
	// Store the user's home directory in the userRoots map
	homeDir := vfs.root.Children["home"].Children[username]
	vfs.userRoots[username] = homeDir

	// Create basic directory FIRST - before loading any files that might be in it
	basicPath := homePath + "/basic"
	vfsDebugLog("Creating basic directory for user: %s", username)
	err = vfs.createDirectoryInMemory(basicPath)
	if err != nil {
		vfsDebugLog("Error creating basic directory %s: %v", basicPath, err)
	} else {
		vfsDebugLog("Basic directory created: %s", basicPath)
	}

	// Load the user's files and directories from the database
	if vfs.db != nil {
		vfsDebugLog("Loading filesystem for user: %s", username)

		// Load all directories first
		rows, err := vfs.db.Query(
			`SELECT path, mod_time FROM virtual_files 
			 WHERE username = ? AND is_dir = 1 
			 ORDER BY length(path)`, // Sort by path length to respect hierarchy
			username)

		if err != nil {
			return fmt.Errorf("database error loading directories: %v", err)
		}

		// Process all directories
		for rows.Next() {
			var path string
			var modTime int64
			if err := rows.Scan(&path, &modTime); err != nil {
				rows.Close()
				return fmt.Errorf("error scanning directory data: %v", err)
			}

			// Skip the user's home directory since we already created it
			if path == homePath {
				continue
			}

			// Create the directory in the in-memory structure
			err := vfs.createDirectoryInMemory(path)
			if err != nil {
				vfsDebugLog("Error creating directory %s: %v", path, err)
				// We continue even if an error occurs
			} else {
				vfsDebugLog("Directory created: %s", path)
			}
		}
		rows.Close()
		// Load all files next
		vfsDebugLog("Starting to load files for user: %s", username)
		fileRows, err := vfs.db.Query(
			`SELECT path, content, mod_time FROM virtual_files 
			 WHERE username = ? AND is_dir = 0`,
			username)

		if err != nil {
			vfsDebugLog("Database error loading files for user: %s", username)
			return fmt.Errorf("database error loading files: %v", err)
		} // Process all files
		fileCount := 0
		for fileRows.Next() {
			fileCount++
			var path string
			var content []byte
			var modTime int64
			if err := fileRows.Scan(&path, &content, &modTime); err != nil {
				vfsDebugLog("Error scanning file data for user %s: %v", username, err)
				fileRows.Close()
				return fmt.Errorf("error scanning file data: %v", err)
			}

			vfsDebugLog("Processing file from DB: %s (content length: %d)", path, len(content))

			// Create the file in the in-memory structure
			t := time.Unix(modTime, 0)
			err := vfs.createFileInMemory(path, string(content), t)
			if err != nil {
				vfsDebugLog("Error creating file %s: %v", path, err)
				// We continue even if an error occurs
			} else {
				vfsDebugLog("File loaded: %s", path)
			}
		}
		vfsDebugLog("Finished loading %d files for user: %s", fileCount, username)
		fileRows.Close()
	}

	// Copy example files to basic directory
	vfs.copyExampleFilesToUser(username, basicPath, homePath)

	vfsDebugLog("VFS for user %s initialized", username)
	return nil
}

// copyExampleFilesToUser kopiert Beispieldateien in das Benutzerverzeichnis (ohne Lock, da bereits in InitializeUserVFS gelockt)
func (vfs *VFS) copyExampleFilesToUser(username, basicPath, homePath string) {
	vfsDebugLog("Copying example files for user: %s", username)

	// Skip example files for Dyson user - Dyson has special files managed separately
	if username == "dyson" {
		vfsDebugLog("Skipping example files for Dyson user - special files managed separately")
		return
	}

	examplesDir := "examples"
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		vfsDebugLog("Error reading examples directory: %v", err)
		return
	}

	fileCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)

		contentBytes, err := os.ReadFile(filepath.Join(examplesDir, name))
		if err != nil {
			vfsDebugLog("Error reading example file %s: %v", name, err)
			continue
		}

		var targetPath string
		// BASIC programs and SID files go to /basic subdirectory
		if strings.HasSuffix(lowerName, ".bas") || strings.HasSuffix(lowerName, ".sid") {
			targetPath = basicPath + "/" + name
		} else if strings.HasSuffix(lowerName, ".txt") {
			// Text files (like readme.txt) go to home directory
			targetPath = homePath + "/" + name
		} else {
			// Skip other file types
			continue
		}
		err = vfs.createFileInMemory(targetPath, string(contentBytes), time.Now())
		if err != nil {
			vfsDebugLog("Error creating %s: %v", name, err)
			continue
		}

		// Speichere die Datei auch in der Datenbank für persistente Benutzer
		if vfs.db != nil && username != "guest" {
			modTime := time.Now().Unix()
			_, dbErr := vfs.db.Exec(
				`INSERT OR REPLACE INTO virtual_files (username, path, content, is_dir, mod_time) VALUES (?, ?, ?, ?, ?)`,
				username, targetPath, string(contentBytes), 0, modTime,
			)
			if dbErr != nil {
				vfsDebugLog("Error saving %s to database: %v", name, dbErr)
			} else {
				vfsDebugLog("%s saved to database for user %s", name, username)
			}
		}

		vfsDebugLog("%s successfully created at %s", name, targetPath)
		fileCount++
	}

	vfsDebugLog("Copied %d example files for user %s", fileCount, username)
}

// createDirectoryInMemory creates a directory only in memory without database operation
func (vfs *VFS) createDirectoryInMemory(path string) error {
	// Convert Windows paths to Unix paths
	path = strings.ReplaceAll(path, "\\", "/")

	// Split the path into its components
	parts := strings.Split(path, "/")
	currentNode := vfs.root
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Update current path
		if currentPath == "" || currentPath == "/" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		// Check if the node already exists
		child, exists := currentNode.Children[part]
		if !exists {
			// Create new directory node
			newNode := &VirtualFile{
				Name:     part,
				IsDir:    true,
				ModTime:  time.Now(),
				Children: make(map[string]*VirtualFile),
				Parent:   currentNode,
			}
			currentNode.Children[part] = newNode
			currentNode = newNode
		} else {
			if !child.IsDir {
				return fmt.Errorf("path component is not a directory: %s", part)
			}
			currentNode = child
		}
	}

	return nil
}

// createFileInMemory creates a file only in memory without database operation
func (vfs *VFS) createFileInMemory(path, content string, modTime time.Time) error {
	// Normalize the path to Unix format for internal consistency
	path = strings.ReplaceAll(path, "\\", "/")
	vfsDebugLog("createFileInMemory for path: %s", path)

	// Split the path into directory and file name
	// Use strings.Split instead of filepath.Dir to be OS-independent
	lastSlash := strings.LastIndex(path, "/")
	var dirPath, fileName string
	if lastSlash == -1 {
		// No directory structure, just file name
		dirPath = "/"
		fileName = path
	} else {
		dirPath = path[:lastSlash]
		if dirPath == "" {
			dirPath = "/"
		}
		fileName = path[lastSlash+1:]
	}

	vfsDebugLog("Split path: dirPath=%s, fileName=%s", dirPath, fileName)

	// Find the parent directory
	parentNode := vfs.root
	if dirPath != "/" {
		dirParts := strings.Split(strings.TrimPrefix(dirPath, "/"), "/")
		for _, part := range dirParts {
			if part == "" {
				continue
			}

			child, exists := parentNode.Children[part]
			if !exists {
				return fmt.Errorf("parent directory does not exist: %s", dirPath)
			}

			if !child.IsDir {
				return fmt.Errorf("path component is not a directory: %s", part)
			}

			parentNode = child
		}
	}

	// Create or update the file
	newNode := &VirtualFile{
		Name:     fileName,
		IsDir:    false,
		Content:  []byte(content),
		ModTime:  modTime,
		Children: nil,
		Parent:   parentNode,
	}

	parentNode.Children[fileName] = newNode
	vfsDebugLog("File successfully created in memory: %s", path)
	return nil
}

// SetTinyOSProvider sets the reference to TinyOS
func (vfs *VFS) SetTinyOSProvider(os TinyOSProvider) {
	vfs.os = os

	// If the provider is set, we immediately initialize the VFS for the current user
	if os != nil {
		username := os.Username()
		if username != "" {
			// Initialize the VFS for the user
			err := vfs.InitializeUserVFS(username)
			if err != nil {
				logger.Warn(logger.AreaFileSystem, "Error initializing VFS for user %s: %v", username, err)
			}
		}
	}
}

// ResolvePath resolves a path to a VirtualFile.
// Returns the found node and the remaining path component if not all could be resolved.
// Also returns an error if the path is invalid or a component does not exist.
func (vfs *VFS) ResolvePath(path string) (*VirtualFile, string, error) {
	node, remaining, err := vfs.resolvePathInternal(path)
	return node, remaining, err
}

func (vfs *VFS) resolvePathInternal(path string) (*VirtualFile, string, error) {
	// Simplified version - use the lock-free variant
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	return vfs.resolvePathInternalWithoutLock(path)
}

// resolvePathInternalWithoutLock is a version of resolvePathInternal,
// that does not use an additional mutex lock (for calls under existing locks)
func (vfs *VFS) resolvePathInternalWithoutLock(path string) (*VirtualFile, string, error) {
	// Convert Windows paths to Unix paths for consistent handling
	path = strings.ReplaceAll(path, "\\", "/")

	// Normalize path: remove double slashes and trailing slashes
	path = normalizePath(path)

	if !strings.HasPrefix(path, "/") {
		return nil, path, errors.New("relative paths not yet supported, use absolute paths starting with /")
	}

	if path == "/" {
		return vfs.root, "", nil
	}

	parts := strings.Split(path[1:], "/") // Split without leading "/"
	current := vfs.root
	resolvedPath := ""

	for i, part := range parts {
		if part == "" {
			continue // Ignore empty parts (e.g. for "//")
		}

		if !current.IsDir {
			// Cannot switch into a file
			return current, strings.Join(parts[i:], "/"), fmt.Errorf("not a directory: %s", resolvedPath)
		}
		child, exists := current.Children[part]
		if !exists {
			// Part of the path does not exist
			return current, strings.Join(parts[i:], "/"), fmt.Errorf("path not found: %s", path)
		}
		current = child
		if resolvedPath == "" {
			resolvedPath = "/" + part
		} else {
			resolvedPath = resolvedPath + "/" + part
		}
	}

	return current, "", nil
}

// normalizePath normalizes a Unix path by removing double slashes and trailing slashes
func normalizePath(path string) string {
	// Remove multiple consecutive slashes
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	// Remove trailing slash except for root
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

// initializeUserVFSWithoutLock is an internal helper method that initializes the VFS for a user
// without acquiring the mutex lock (for calls from methods that already hold the lock)
func (vfs *VFS) initializeUserVFSWithoutLock(username string) error {
	if username == "" {
		return fmt.Errorf("username must not be empty")
	}

	vfsDebugLog("Initializing VFS (without lock) for user: %s", username)

	// Create the home directory for the user
	homePath := "/home/" + username

	// Check if the user already has a home directory
	_, exists := vfs.userRoots[username]
	if exists {
		vfsDebugLog("User VFS is already initialized: %s", username)
		return nil
	}

	// Here the directory must already exist, as we are in ResolvePath
	homeDir, exists := vfs.root.Children["home"].Children[username]
	if !exists {
		// Create it if it doesn't exist
		err := vfs.createDirectoryWithoutLock(homePath)
		if err != nil {
			return fmt.Errorf("error creating home directory: %v", err)
		}
		homeDir = vfs.root.Children["home"].Children[username]
	}

	vfs.userRoots[username] = homeDir

	// Load the user's files and directories from the database
	if vfs.db != nil {
		vfsDebugLog("Loading filesystem for user: %s", username)

		// Load all directories first
		rows, err := vfs.db.Query(
			`SELECT path, mod_time FROM virtual_files 
			 WHERE username = ? AND is_dir = 1 
			 ORDER BY length(path)`, // Sort by path length to respect hierarchy
			username)

		if err != nil {
			return fmt.Errorf("database error loading directories: %v", err)
		}

		// Process all directories
		for rows.Next() {
			var path string
			var modTime int64
			if err := rows.Scan(&path, &modTime); err != nil {
				rows.Close()
				return fmt.Errorf("error scanning directory data: %v", err)
			}

			// Skip the user's home directory, as we have already created it
			if path == homePath {
				continue
			}

			// Create the directory in the in-memory structure
			err := vfs.createDirectoryWithoutLock(path)
			if err != nil {
				vfsDebugLog("Error creating directory %s: %v", path, err)
				// Continue even if there is an error
			} else {
				vfsDebugLog("Directory created: %s", path)
			}
		}
		rows.Close()

		// Load all files next
		fileRows, err := vfs.db.Query(
			`SELECT path, content, mod_time FROM virtual_files 
			 WHERE username = ? AND is_dir = 0`,
			username)

		if err != nil {
			return fmt.Errorf("database error loading files: %v", err)
		}

		// Process all files
		for fileRows.Next() {
			var path string
			var content []byte
			var modTime int64
			if err := fileRows.Scan(&path, &content, &modTime); err != nil {
				fileRows.Close()
				return fmt.Errorf("error scanning file data: %v", err)
			}

			// Create the file in the in-memory structure
			t := time.Unix(modTime, 0)
			err := vfs.createFileWithoutLock(path, string(content), t)
			if err != nil {
				vfsDebugLog("Error creating file %s: %v", path, err)
				// Continue even if there is an error
			} else {
				vfsDebugLog("File loaded: %s", path)
			}
		}
		fileRows.Close()
	}

	vfsDebugLog("VFS for user %s initialized", username)
	return nil
}

// createDirectoryWithoutLock is an internal helper method that creates a directory,
// without locking the mutex (for calls from methods that already have the lock)
func (vfs *VFS) createDirectoryWithoutLock(path string) error { // SICHERHEIT: Überprüfe Benutzer-Limits für Verzeichnisse
	username := vfs.getUsernameFromPath(path)
	if username != "" { // Nur für Benutzerverzeichnisse prüfen
		if err := vfs.checkUserDirectoryLimitWithoutLock(username); err != nil {
			log.Printf("[VFS-SECURITY] Directory creation blocked: %v", err)
			return err
		}
	}

	// Implementation like createDirectoryInMemory, but without lock
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	currentNode := vfs.root
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}

		if currentPath == "" || currentPath == "/" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		child, exists := currentNode.Children[part]
		if !exists {
			newNode := &VirtualFile{
				Name:     part,
				IsDir:    true,
				ModTime:  time.Now(),
				Children: make(map[string]*VirtualFile),
				Parent:   currentNode,
			}
			currentNode.Children[part] = newNode
			currentNode = newNode
		} else {
			if !child.IsDir {
				return fmt.Errorf("path component is not a directory: %s", part)
			}
			currentNode = child
		}
	}

	return nil
}

// createFileWithoutLock is an internal helper method that creates a file,
// without locking the mutex (for calls from methods that already have the lock)
func (vfs *VFS) createFileWithoutLock(path, content string, modTime time.Time) error {
	// Implementation like createFileInMemory, but without lock
	path = strings.ReplaceAll(path, "\\", "/")
	lastSlash := strings.LastIndex(path, "/")
	var dirPath, fileName string
	if lastSlash == -1 {
		// No directory structure, just file name
		dirPath = "/"
		fileName = path
	} else {
		dirPath = path[:lastSlash]
		if dirPath == "" {
			dirPath = "/"
		}
		fileName = path[lastSlash+1:]
	}

	// Find the parent directory
	parentNode := vfs.root
	if dirPath != "/" {
		dirParts := strings.Split(strings.TrimPrefix(dirPath, "/"), "/")
		for _, part := range dirParts {
			if part == "" {
				continue
			}

			child, exists := parentNode.Children[part]
			if !exists {
				return fmt.Errorf("parent directory does not exist: %s", dirPath)
			}

			if !child.IsDir {
				return fmt.Errorf("path component is not a directory: %s", part)
			}

			parentNode = child
		}
	}

	// Create or update the file
	newNode := &VirtualFile{
		Name:     fileName,
		IsDir:    false,
		Content:  []byte(content),
		ModTime:  modTime,
		Children: nil,
		Parent:   parentNode,
	}

	parentNode.Children[fileName] = newNode
	return nil
}

// ReadFile reads the content of a file in the VFS
func (vfs *VFS) ReadFile(path string, sessionID string) (string, error) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()
	vfsDebugLog("ReadFile: path=%s, sessionID=%s", path, sessionID) // If path is relative, convert to current working directory
	if !strings.HasPrefix(path, "/") && vfs.os != nil {
		username := "guest"
		currentPath := "/home/guest" // Default

		if sessionID != "" {
			if sessionProvider, ok := vfs.os.(interface{ UsernameFromSession(string) string }); ok {
				uname := sessionProvider.UsernameFromSession(sessionID)
				if uname != "" {
					username = uname
				}
			}

			// Try to get current working directory from session
			if pathProvider, ok := vfs.os.(interface{ CurrentPathFromSession(string) string }); ok {
				workingDir := pathProvider.CurrentPathFromSession(sessionID)
				if workingDir != "" {
					currentPath = workingDir
				} else {
					currentPath = "/home/" + username
				}
			} else {
				currentPath = "/home/" + username
			}
		}

		path = currentPath + "/" + path
		vfsDebugLog("ReadFile: Relative path detected, converted to: %s", path)
	}
	node, remaining, err := vfs.ResolvePath(path)
	vfsDebugLog("After ResolvePath: node=%v, remaining=%s, err=%v", node, remaining, err)
	if err != nil && remaining == "" { // Error resolving exact path
		vfsDebugLog("Error resolving exact path: %v", err)
		return "", err
	}
	if remaining != "" { // Path could not be fully resolved
		vfsDebugLog("Path not fully resolved: remaining=%s", remaining)
		return "", fmt.Errorf("path not found: %s", path)
	}
	if node.IsDir {
		vfsDebugLog("Node is a directory: %s", path)
		return "", fmt.Errorf("is a directory: %s", path)
	}
	vfsDebugLog("File content is being returned: %s", path)
	return string(node.Content), nil
}

// WriteFile writes content to a file in the VFS. Creates the file if it does not exist.
func (vfs *VFS) WriteFile(path, content string, sessionID string) error {
	vfsDebugLog("WriteFile Start for path: %s (with SessionID: %s)", path, sessionID)

	// SICHERHEIT: Überprüfe Dateigröße vor dem Speichern
	if err := vfs.checkFileSizeLimit(content); err != nil {
		log.Printf("[VFS-SECURITY] File size limit exceeded: %v", err)
		return err
	}

	// Normalize the path to a Unix path before any operations
	path = strings.ReplaceAll(path, "\\", "/")
	vfsDebugLog("WriteFile - Normalized path: %s", path)

	// If a session ID is provided, try to determine the associated user
	var username string
	if sessionID != "" && vfs.os != nil {
		// Assumption: There is a method to get the username from a session ID
		if sessionProvider, ok := vfs.os.(interface{ UsernameFromSession(string) string }); ok {
			username = sessionProvider.UsernameFromSession(sessionID)
			vfsDebugLog("WriteFile with SessionID %s, determined user: %s", sessionID, username)
		}
	}
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Determine the currently logged-in user first
	if username == "" && vfs.os != nil {
		username = vfs.os.Username()
		vfsDebugLog("WriteFile - Logged-in user: %s", username)
	}

	// If no user is logged in, use a default value
	if username == "" {
		username = "guest" // Guest user as default
		vfsDebugLog("WriteFile - Using default user: %s", username)
	}
	// Handle relative paths by converting them to absolute paths
	if !strings.HasPrefix(path, "/") {
		// This is a relative path, convert it to absolute based on user's current directory
		currentPath := "/home/" + username // Default to home directory

		// Try to get current working directory from session
		if sessionID != "" && vfs.os != nil {
			if pathProvider, ok := vfs.os.(interface{ CurrentPathFromSession(string) string }); ok {
				workingDir := pathProvider.CurrentPathFromSession(sessionID)
				if workingDir != "" {
					currentPath = workingDir
				}
			}
		}

		path = currentPath + "/" + path
		vfsDebugLog("WriteFile - Converted relative path to absolute: %s", path)
	}

	// Extract directory and file name, ensure we are using Unix paths
	dirPath := filepath.Dir(path)
	dirPath = strings.ReplaceAll(dirPath, "\\", "/")
	fileName := filepath.Base(path)

	if fileName == "." || fileName == "/" {
		return errors.New("invalid file name")
	}
	vfsDebugLog("WriteFile - Base path: %s, File name: %s", dirPath, fileName)

	// Optimization for known paths (direct path resolution)
	// For guest user directory or home directories - use generic path resolution
	if dirPath == "/home/guest" || strings.HasPrefix(dirPath, "/home/guest/") {
		vfsDebugLog("WriteFile - Direct processing for guest directory/subdirectory")
		// Resolve the target directory
		targetDir, remaining, err := vfs.resolvePathInternalWithoutLock(dirPath)
		if err != nil || remaining != "" {
			vfsDebugLog("WriteFile - Target directory could not be resolved: %s (err: %v, remaining: %s)", dirPath, err, remaining)
			return fmt.Errorf("target directory not found: %s", dirPath)
		}

		if !targetDir.IsDir {
			vfsDebugLog("WriteFile - Target is not a directory: %s", dirPath)
			return fmt.Errorf("target is not a directory: %s", dirPath)
		}

		// Create or update the file in the target directory
		if existingFile, exists := targetDir.Children[fileName]; exists {
			if existingFile.IsDir {
				vfsDebugLog("WriteFile - Cannot overwrite directory")
				return fmt.Errorf("cannot overwrite directory with file: %s", fileName)
			}
			existingFile.Content = []byte(content)
			existingFile.ModTime = time.Now()
			vfsDebugLog("WriteFile - Updated existing file: %s", fileName)
		} else {
			// SICHERHEIT: Überprüfe Datei-Limit im Verzeichnis vor dem Erstellen einer neuen Datei
			if err := vfs.checkDirectoryFileLimit(targetDir); err != nil {
				log.Printf("[VFS-SECURITY] File creation blocked in guest directory: %v", err)
				return err
			}

			// Create new file
			newFile := &VirtualFile{
				Name:     fileName,
				IsDir:    false,
				Content:  []byte(content),
				ModTime:  time.Now(),
				Children: nil,
				Parent:   targetDir,
			}
			targetDir.Children[fileName] = newFile
			vfsDebugLog("WriteFile - Created new file: %s", fileName)
		}
		// For guest users, we do NOT save to database - only RAM storage
		vfsDebugLog("WriteFile - Guest user file saved only in RAM, no database persistence")

		vfsDebugLog("WriteFile - Operation completed successfully (guest directory): %s", path)
		return nil
	} else if strings.HasPrefix(dirPath, "/home/") && dirPath != "/home" {
		// Direct processing for user directories with resolvePathInternalWithoutLock
		vfsDebugLog("WriteFile - Direct processing for user directory")

		// Resolve the target directory
		targetDir, remaining, err := vfs.resolvePathInternalWithoutLock(dirPath)
		if err != nil || remaining != "" {
			vfsDebugLog("WriteFile - Target directory could not be resolved: %s (err: %v, remaining: %s)", dirPath, err, remaining)
			return fmt.Errorf("target directory not found: %s", dirPath)
		}

		if !targetDir.IsDir {
			vfsDebugLog("WriteFile - Target is not a directory: %s", dirPath)
			return fmt.Errorf("target is not a directory: %s", dirPath)
		}

		// Create or update the file in the target directory
		if existingFile, exists := targetDir.Children[fileName]; exists {
			if existingFile.IsDir {
				vfsDebugLog("WriteFile - Cannot overwrite directory")
				return fmt.Errorf("cannot overwrite directory with file: %s", fileName)
			}
			existingFile.Content = []byte(content)
			existingFile.ModTime = time.Now()
			vfsDebugLog("WriteFile - Updated existing file: %s", fileName)
		} else {
			// SICHERHEIT: Überprüfe Datei-Limit im Verzeichnis vor dem Erstellen einer neuen Datei
			if err := vfs.checkDirectoryFileLimit(targetDir); err != nil {
				log.Printf("[VFS-SECURITY] File creation blocked in user directory: %v", err)
				return err
			}

			// Create new file
			newFile := &VirtualFile{
				Name:     fileName,
				IsDir:    false,
				Content:  []byte(content),
				ModTime:  time.Now(),
				Children: nil,
				Parent:   targetDir,
			}
			targetDir.Children[fileName] = newFile
			vfsDebugLog("WriteFile - Created new file: %s", fileName)
		}

		// Save asynchronously to the database
		pathCopy := path
		contentCopy := content
		usernameCopy := username
		go func() {
			if vfs.db != nil {
				isDir := 0
				modTime := time.Now().Unix()

				vfsDebugLog("WriteFile[async] - Executing DB save for: %s", pathCopy)

				_, err := vfs.db.Exec(
					`INSERT OR REPLACE INTO virtual_files (username, path, content, is_dir, mod_time) VALUES (?, ?, ?, ?, ?)`,
					usernameCopy, pathCopy, contentCopy, isDir, modTime,
				)

				if err != nil {
					vfsDebugLog("WriteFile[async] - DB error: %v", err)
				} else {
					vfsDebugLog("WriteFile[async] - DB save successful for: %s", pathCopy)
				}
			}
		}()

		vfsDebugLog("WriteFile - Operation completed successfully (user directory): %s", path)
		return nil
	}

	// Standard processing for other paths
	// Find the directory where the file should be created
	vfsDebugLog("WriteFile - Searching directory: %s", dirPath)

	dirNode, remaining, err := vfs.resolvePathInternalWithoutLock(dirPath)
	if err != nil || remaining != "" {
		return fmt.Errorf("target directory not found: %s", dirPath)
	}

	if !dirNode.IsDir {
		return fmt.Errorf("target is not a directory: %s", dirPath)
	}

	// Create or update the file
	if existingFile, exists := dirNode.Children[fileName]; exists {
		if existingFile.IsDir {
			return fmt.Errorf("cannot overwrite directory with file: %s", fileName)
		}
		existingFile.Content = []byte(content)
		existingFile.ModTime = time.Now()
		vfsDebugLog("WriteFile - Updated existing file: %s", fileName)
	} else {
		// Check file limit before creating new file
		if err := vfs.checkDirectoryFileLimit(dirNode); err != nil {
			log.Printf("[VFS-SECURITY] File creation blocked: %v", err)
			return err
		}

		// Create new file
		vfsDebugLog("WriteFile - Created new file: %s", path)
		newNode := &VirtualFile{
			Name:     fileName,
			IsDir:    false,
			Content:  []byte(content),
			ModTime:  time.Now(),
			Children: nil,
			Parent:   dirNode,
		}
		dirNode.Children[fileName] = newNode
	}

	vfsDebugLog("WriteFile - File successfully written to memory: %s", path)

	// Save the change to the database as well, but only if a DB connection exists
	// Avoid timeout by saving asynchronously
	if vfs.db != nil {
		vfsDebugLog("WriteFile - Starting asynchronous DB save: %s", path)

		// Create copies of the values for the goroutine
		pathCopy := path
		contentCopy := content
		usernameCopy := username

		// Save asynchronously without waiting
		go func() {
			isDir := 0 // File, not directory
			modTime := time.Now().Unix()

			vfsDebugLog("WriteFile[async] - Executing DB save for: %s", pathCopy)

			// SQL to update or insert into the database
			_, err := vfs.db.Exec(
				`INSERT OR REPLACE INTO virtual_files (username, path, content, is_dir, mod_time) VALUES (?, ?, ?, ?, ?)`,
				usernameCopy, pathCopy, contentCopy, isDir, modTime,
			)

			if err != nil {
				vfsDebugLog("WriteFile[async] - DB error: %v", err)
			} else {
				vfsDebugLog("WriteFile[async] - DB save successful for: %s", pathCopy)
			}
		}()
	} else {
		vfsDebugLog("WriteFile - No DB connection, only saved in memory: %s", path)
	}

	vfsDebugLog("WriteFile - Operation completed successfully (standard directory): %s", path)
	return nil
}

// GetUserStats sammelt Statistiken über die VFS-Nutzung eines Benutzers
func (vfs *VFS) GetUserStats(username string) (*UserStats, error) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	stats := &UserStats{
		Username:       username,
		MaxDirectories: configuration.GetInt("FileSystem", "max_directories_per_user", 20),
		MaxFilesPerDir: configuration.GetInt("FileSystem", "max_files_per_directory", 100),
	}

	// Zähle Verzeichnisse
	stats.DirectoryCount = vfs.countUserDirectoriesLocked(username)

	// Hole das Benutzerverzeichnis und zähle Dateien
	userHomePath := "/home/" + username
	userHome, _, err := vfs.resolvePathInternalWithoutLock(userHomePath)
	if err != nil {
		return stats, nil // Benutzerverzeichnis existiert noch nicht
	}

	if userHome != nil && userHome.IsDir {
		stats.HomeDirectoryFiles = vfs.countFilesInDirectory(userHome)
		stats.TotalFiles = vfs.countFilesRecursive(userHome)
	}

	return stats, nil
}

// countUserDirectoriesLocked zählt Verzeichnisse (Lock muss bereits gehalten werden)
func (vfs *VFS) countUserDirectoriesLocked(username string) int {
	userHomePath := "/home/" + username
	userHome, _, err := vfs.resolvePathInternalWithoutLock(userHomePath)
	if err != nil || userHome == nil {
		return 0
	}
	return vfs.countDirectoriesRecursive(userHome)
}

// countFilesRecursive zählt alle Dateien rekursiv in einem Verzeichnis
func (vfs *VFS) countFilesRecursive(node *VirtualFile) int {
	if !node.IsDir {
		return 1
	}

	count := 0
	for _, child := range node.Children {
		count += vfs.countFilesRecursive(child)
	}
	return count
}

// Exists checks if a file or directory exists
func (vfs *VFS) Exists(path string, sessionID string) bool {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	// If a session ID is provided, try to determine the associated user
	if sessionID != "" && vfs.os != nil {
		// This function would need to be implemented in TinyOSProvider
		if sessionProvider, ok := vfs.os.(interface{ UsernameFromSession(string) string }); ok {
			username := sessionProvider.UsernameFromSession(sessionID)
			vfsDebugLog("Exists with SessionID %s, determined user: %s", sessionID, username)
		}
	}

	_, remaining, err := vfs.ResolvePath(path)
	return err == nil && remaining == ""
}

// ListDirBasFiles lists all .bas files in the user's directory
// DEPRECATED: Use ListDirProgramFiles instead
// This version implements the FileSystem interface for TinyBASIC
func (vfs *VFS) ListDirBasFiles(sessionID string) ([]string, error) {
	return vfs.ListDirProgramFiles(sessionID)
}

// ListDirBasFilesForUser returns all .bas files in a user's home directory
// DEPRECATED: Use ListDirProgramFilesForUser instead
func (v *VFS) ListDirBasFilesForUser(username string) ([]string, error) {
	return v.ListDirProgramFilesForUser(username)
}

// ListDirProgramFiles lists all program files (.bas and .sid) in the user's directory
func (vfs *VFS) ListDirProgramFiles(sessionID string) ([]string, error) {
	username := "guest"

	// Versuche, den Benutzernamen über die Session zu ermitteln
	if sessionID != "" {
		if vfs.db != nil {
			vfsDebugLog("ListDirProgramFiles with SessionID %s, querying for username", sessionID)
			row := vfs.db.QueryRow("SELECT username FROM sessions WHERE session_id = ? AND expires_at > ?", sessionID, time.Now().Unix())
			var foundUsername string
			if err := row.Scan(&foundUsername); err == nil && foundUsername != "" {
				username = foundUsername
			}
			vfsDebugLog("ListDirProgramFiles with SessionID %s, determined user: %s", sessionID, username)
		}
	}

	return vfs.ListDirProgramFilesForUser(username)
}

// ListDirProgramFilesForUser returns all program files (.bas and .sid) in a user's home directory
func (v *VFS) ListDirProgramFilesForUser(username string) ([]string, error) {
	// If no username is provided, we cannot proceed
	if username == "" {
		return nil, fmt.Errorf("no username provided")
	}
	// For guest users, use the RAM-VFS and look in the basic subdirectory
	if username == "guest" {
		log.Printf("[VFS] ListDirProgramFilesForUser for guest - using RAM-VFS instead of database")
		basicPath := "/home/guest/basic"
		v.mu.RLock()
		defer v.mu.RUnlock()
		basicDir, remaining, err := v.ResolvePath(basicPath)
		if err != nil || remaining != "" {
			return nil, fmt.Errorf("guest basic directory not found: %v", err)
		}

		if !basicDir.IsDir {
			return nil, fmt.Errorf("guest basic path is not a directory: %s", basicPath)
		}

		// Collect all .bas and .sid files from the basic directory
		var programFiles []string
		for name, file := range basicDir.Children {
			if !file.IsDir {
				lowerName := strings.ToLower(name)
				if strings.HasSuffix(lowerName, ".bas") || strings.HasSuffix(lowerName, ".sid") {
					programFiles = append(programFiles, name)
				}
			}
		}

		return programFiles, nil
	}

	// For regular users, query the database
	if v.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	// SQL query to find all .bas and .sid files in the user's basic directory
	query := `
		SELECT path FROM virtual_files 
		WHERE username = ? 
		  AND is_dir = 0 
		  AND (
		    LOWER(path) LIKE '%.bas'
		    OR LOWER(path) LIKE '%.sid'
		  )
		  AND path LIKE '/home/' || ? || '/basic/%'
	`
	rows, err := v.db.Query(query, username, username)
	if err != nil {
		return nil, fmt.Errorf("error querying program files: %v", err)
	}
	defer rows.Close()
	var programFiles []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		filename := filepath.Base(path)
		programFiles = append(programFiles, filename)
	}

	// Verhindere rekursive SyncExamplePrograms-Aufrufe
	if len(programFiles) == 0 {
		log.Printf("[VFS] No program files found for user %s, checking if sync already attempted", username)

		// Prüfe, ob basic-Verzeichnis existiert - wenn ja, dann wurde bereits sync versucht
		basicDirQuery := `SELECT COUNT(*) FROM virtual_files WHERE username = ? AND path = '/home/' || ? || '/basic' AND is_dir = 1`
		var count int
		err := v.db.QueryRow(basicDirQuery, username, username).Scan(&count)
		if err != nil {
			return nil, fmt.Errorf("error checking basic directory: %v", err)
		}

		if count == 0 {
			log.Printf("[VFS] Basic directory not found for user %s, trying SyncExamplePrograms", username)
			if err := v.SyncExamplePrograms(username); err != nil {
				return nil, fmt.Errorf("no program files found and sync failed: %v", err)
			}

			// Try querying again after sync - but only once
			rows, err := v.db.Query(query, username, username)
			if err != nil {
				return nil, fmt.Errorf("error querying program files after sync: %v", err)
			}
			defer rows.Close()

			for rows.Next() {
				var path string
				if err := rows.Scan(&path); err != nil {
					continue
				}
				filename := filepath.Base(path)
				programFiles = append(programFiles, filename)
			}
		} else {
			log.Printf("[VFS] Basic directory exists for user %s but no program files found - sync already attempted", username)
		}
	}

	return programFiles, nil
}

// ListDirAllFiles lists all files in the user's directory (not just .bas and .sid)
func (vfs *VFS) ListDirAllFiles(sessionID string) ([]string, error) {
	// If no session ID is provided, we cannot proceed
	if sessionID == "" {
		return nil, fmt.Errorf("no sessionID provided")
	}

	// Get username from session
	var username string
	if vfs.os != nil {
		if sessionProvider, ok := vfs.os.(interface{ UsernameFromSession(string) string }); ok {
			username = sessionProvider.UsernameFromSession(sessionID)
			vfsDebugLog("ListDirAllFiles with SessionID %s, querying for username", sessionID)
		}
	}

	if username == "" && vfs.os != nil {
		username = vfs.os.Username()
		vfsDebugLog("ListDirAllFiles with SessionID %s, determined user: %s", sessionID, username)
	}

	if username == "" {
		return nil, fmt.Errorf("no username provided")
	}

	return vfs.ListDirAllFilesForUser(username)
}

// ListDirAllFilesForUser returns all files in a user's home directory
func (v *VFS) ListDirAllFilesForUser(username string) ([]string, error) {
	// If no username is provided, we cannot proceed
	if username == "" {
		return nil, fmt.Errorf("no username provided")
	}
	// For guest users, use the RAM-VFS and look in the basic subdirectory
	if username == "guest" {
		log.Printf("[VFS] ListDirAllFilesForUser for guest - using RAM-VFS instead of database, looking in basic directory")
		basicPath := "/home/guest/basic"
		v.mu.RLock()
		defer v.mu.RUnlock()
		basicDir, remaining, err := v.ResolvePath(basicPath)
		if err != nil || remaining != "" {
			return nil, fmt.Errorf("guest basic directory not found: %v", err)
		}

		if !basicDir.IsDir {
			return nil, fmt.Errorf("guest basic path is not a directory: %s", basicPath)
		}

		// Collect all files from the basic directory (no filtering)
		var allFiles []string
		for name, file := range basicDir.Children {
			if !file.IsDir {
				allFiles = append(allFiles, name)
			}
		}

		return allFiles, nil
	}

	// For regular users, query the database
	if v.db == nil {
		return nil, fmt.Errorf("database not available")
	}
	// Query for all files in the user's basic directory
	query := `
		SELECT path FROM virtual_files 
		WHERE username = ? AND is_dir = 0 AND path LIKE '/home/' || ? || '/basic/%'
		ORDER BY path
	`

	rows, err := v.db.Query(query, username, username)
	if err != nil {
		return nil, fmt.Errorf("error querying all files: %v", err)
	}
	defer rows.Close()

	var allFiles []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		filename := filepath.Base(path)
		allFiles = append(allFiles, filename)
	}

	return allFiles, nil
}

// ListDir returns a list of all files and directories in the specified path
func (vfs *VFS) ListDir(path string) ([]string, error) {
	// IMPORTANT: User initialization must be done before the lock
	if strings.HasPrefix(path, "/home/") {
		pathParts := strings.Split(path, "/")
		if len(pathParts) >= 3 && pathParts[2] != "" {
			requestedUser := pathParts[2]
			currentUser := ""

			if vfs.os != nil {
				currentUser = vfs.os.Username()
			}

			// Security checks
			if requestedUser == "guest" {
				// Check if guest VFS is initialized
				vfs.mu.RLock()
				_, guestExists := vfs.userRoots["guest"]
				vfs.mu.RUnlock()

				if !guestExists {
					err := vfs.InitializeGuestVFS()
					if err != nil {
						return nil, fmt.Errorf("error initializing guest VFS: %v", err)
					}
				}
			} else {
				// Normal users
				if currentUser == "" {
					return nil, fmt.Errorf("access denied: you need to log in to access user directories")
				}
				if currentUser != requestedUser {
					return nil, fmt.Errorf("access denied: you can only access your own home directory")
				}
				// Check if user VFS is initialized
				err := vfs.safeInitializeUserVFS(currentUser)
				if err != nil {
					return nil, fmt.Errorf("error initializing user VFS: %v", err)
				}
			}
		}
	}

	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	node, remaining, err := vfs.resolvePathInternalWithoutLock(path)
	if err != nil {
		return nil, err
	}
	if remaining != "" {
		return nil, fmt.Errorf("Path not found: %s", path)
	}
	if !node.IsDir {
		return nil, fmt.Errorf("is not a directory: %s", path)
	}
	var result []string
	for name, child := range node.Children {
		if child.IsDir {
			// Add trailing slash for directories (retro UNIX style)
			result = append(result, name+"/")
		} else {
			// Files without modification
			result = append(result, name)
		}
	}
	return result, nil
}

// Mkdir creates a single directory
func (vfs *VFS) Mkdir(path string) error {
	vfs.mu.Lock()
	defer vfs.mu.Unlock() // Check if path is empty
	if path == "" {
		return fmt.Errorf("Empty path")
	}

	// If the directory already exists
	if vfs.existsWithoutLock(path) {
		// Check if it is a directory
		node, _, _ := vfs.resolvePathInternalWithoutLock(path)
		if !node.IsDir {
			return fmt.Errorf("Exists as file: %s", path)
		}
		return nil // Directory already exists, no error
	}

	// Create the directory
	return vfs.createDirectoryWithoutLock(path)
}

// MkdirAll creates a directory and all necessary parent directories
func (vfs *VFS) MkdirAll(path string) error {
	log.Printf("[VFS-MKDIRALL-DEBUG] MkdirAll aufgerufen für Pfad: %s", path)
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Check if path is empty
	if path == "" {
		log.Printf("[VFS-MKDIRALL-DEBUG] Fehler: Leerer Pfad")
		return fmt.Errorf("Empty path")
	}

	log.Printf("[VFS-MKDIRALL-DEBUG] Prüfe ob Pfad bereits existiert: %s", path)
	// If the directory already exists
	if vfs.existsWithoutLock(path) {
		log.Printf("[VFS-MKDIRALL-DEBUG] Pfad existiert bereits: %s", path)
		// Check if it is a directory
		node, _, _ := vfs.resolvePathInternalWithoutLock(path)
		if !node.IsDir {
			log.Printf("[VFS-MKDIRALL-DEBUG] Fehler: Pfad existiert als Datei: %s", path)
			return fmt.Errorf("Exists as file: %s", path)
		}
		log.Printf("[VFS-MKDIRALL-DEBUG] Verzeichnis existiert bereits, keine Aktion erforderlich: %s", path)
		return nil // Directory already exists, no error
	}

	log.Printf("[VFS-MKDIRALL-DEBUG] Erstelle Verzeichnisstruktur für: %s", path)
	// Create all parent directories
	components := strings.Split(path, "/")
	currentPath := ""

	for i, component := range components {
		if component == "" {
			if i == 0 { // Root path starts with "/"
				continue
			}
			continue // Ignore empty path components
		}

		// Build the path
		if i == 0 {
			currentPath = "/" + component
		} else {
			currentPath = currentPath + "/" + component
		}

		log.Printf("[VFS-MKDIRALL-DEBUG] Verarbeite Komponente: %s (aktueller Pfad: %s)", component, currentPath)

		// Check if the directory already exists
		if vfs.existsWithoutLock(currentPath) {
			log.Printf("[VFS-MKDIRALL-DEBUG] Komponente existiert bereits: %s", currentPath)
			node, _, _ := vfs.resolvePathInternalWithoutLock(currentPath)
			if !node.IsDir {
				log.Printf("[VFS-MKDIRALL-DEBUG] Fehler: Pfadkomponente ist eine Datei: %s", currentPath)
				return fmt.Errorf("Path contains a file instead of a directory: %s", currentPath)
			}
			continue
		}

		log.Printf("[VFS-MKDIRALL-DEBUG] Erstelle Verzeichnis: %s", currentPath)
		// Create the directory
		err := vfs.createDirectoryWithoutLock(currentPath)
		if err != nil {
			log.Printf("[VFS-MKDIRALL-DEBUG] Fehler beim Erstellen von %s: %v", currentPath, err)
			return fmt.Errorf("Error creating directory %s: %v", currentPath, err)
		}
		log.Printf("[VFS-MKDIRALL-DEBUG] Verzeichnis erfolgreich erstellt: %s", currentPath)
	}

	log.Printf("[VFS-MKDIRALL-DEBUG] MkdirAll erfolgreich abgeschlossen für: %s", path)
	return nil
}

// Remove deletes a file or an empty directory
func (vfs *VFS) Remove(path string) error {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Check if the path exists
	if !vfs.existsWithoutLock(path) {
		return fmt.Errorf("Path not found: %s", path)
	}
	// Resolve the path
	node, remaining, err := vfs.resolvePathInternalWithoutLock(path)
	if err != nil || remaining != "" {
		return fmt.Errorf("Path not found: %s", path)
	}

	// If it is a directory, check if it is empty
	if node.IsDir && len(node.Children) > 0 {
		return fmt.Errorf("Directory is not empty: %s", path)
	}

	// Determine parent directory and remove child node
	parentPath := filepath.Dir(path)
	if parentPath == path { // Root directory cannot be deleted
		return fmt.Errorf("Root directory cannot be deleted")
	}
	parentNode, _, err := vfs.resolvePathInternalWithoutLock(parentPath)
	if err != nil {
		return fmt.Errorf("Parent directory not found: %s", parentPath)
	}

	childName := filepath.Base(path)
	delete(parentNode.Children, childName)
	// If the database is available, delete there as well (but NOT for guest users)
	if vfs.db != nil {
		// Determine the username from the path
		username := ""
		if strings.HasPrefix(path, "/home/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 3 {
				username = parts[2]
			}
		}

		// Only delete from database for non-guest users
		if username != "" && username != "guest" {
			_, err := vfs.db.Exec(
				"DELETE FROM virtual_files WHERE username = ? AND path = ?",
				username, path)

			if err != nil {
				return fmt.Errorf("database error: %v", err)
			}
		} else if username == "guest" {
			vfsDebugLog("Remove - Guest user file deleted only from RAM, no database operation")
		}
	}

	return nil
}

// NotifyFileChanged marks a file as changed
// This method can be used to inform listeners about changes
func (vfs *VFS) NotifyFileChanged(path string) {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Check if the path exists
	if !vfs.existsWithoutLock(path) {
		log.Printf("NotifyFileChanged: Path not found: %s", path)
		return
	}

	// Resolve the path
	node, remaining, err := vfs.resolvePathInternalWithoutLock(path)
	if err != nil || remaining != "" {
		log.Printf("NotifyFileChanged: Path could not be resolved: %s (err: %v, remaining: %s)", path, err, remaining)
		return
	}

	// Mark the file as changed
	node.ModTime = time.Now()
	log.Printf("NotifyFileChanged: File changed: %s", path)
}

// existsWithoutLock checks if a path exists without acquiring the mutex
// (for calls from methods that already hold the lock)
func (vfs *VFS) existsWithoutLock(path string) bool {
	node, remaining, err := vfs.resolvePathInternalWithoutLock(path)
	if err != nil || remaining != "" || node == nil {
		return false
	}
	return true
}

// IsDir checks if a path is a directory
func (vfs *VFS) IsDir(path string) bool {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	node, remaining, err := vfs.ResolvePath(path)
	if err != nil || remaining != "" {
		return false
	}

	return node.IsDir
}

// initializingUsers keeps track of which users are currently being initialized
var initializingUsers = make(map[string]bool)
var initializingMutex = sync.Mutex{}

// safeInitializeUserVFS initializes a user safely across goroutines
func (vfs *VFS) safeInitializeUserVFS(username string) error {
	// First check if already initialized (without lock)
	vfs.mu.RLock()
	_, exists := vfs.userRoots[username]
	vfs.mu.RUnlock()

	if exists {
		return nil // Already initialized
	}

	// Thread-safe initialization
	initializingMutex.Lock()
	defer initializingMutex.Unlock()

	// Double-check after lock
	vfs.mu.RLock()
	_, exists = vfs.userRoots[username]
	vfs.mu.RUnlock()

	if exists {
		return nil // Already initialized
	}

	// Check if already being initialized
	if initializingUsers[username] {
		// Wait until initialization is complete
		for initializingUsers[username] {
			initializingMutex.Unlock()
			time.Sleep(10 * time.Millisecond)
			initializingMutex.Lock()
		}
		return nil
	}

	// Mark as "being initialized"
	initializingUsers[username] = true
	defer func() {
		delete(initializingUsers, username)
	}()

	// Release lock for actual initialization
	initializingMutex.Unlock()
	err := vfs.InitializeUserVFS(username)
	initializingMutex.Lock()

	return err
}

// initializeGuestVFSWithoutLock initializes the VFS for guest users without mutex lock
// This function is called by resolvePathInternal, where a mutex lock is already held
func (vfs *VFS) initializeGuestVFSWithoutLock() error {

	// Create /home if it doesn't exist
	if vfs.root.Children["home"] == nil {
		vfs.root.Children["home"] = &VirtualFile{
			Name:     "home",
			IsDir:    true,
			Children: make(map[string]*VirtualFile),
			Parent:   vfs.root,
		}
	}
	// Create a completely new /home/guest directory (overwrites old one)
	guestDir := &VirtualFile{
		Name:     "guest",
		IsDir:    true,
		Children: make(map[string]*VirtualFile),
		Parent:   vfs.root.Children["home"],
		ModTime:  time.Now(),
	}
	vfs.root.Children["home"].Children["guest"] = guestDir
	vfs.userRoots["guest"] = guestDir

	// Create basic subdirectory for BASIC programs
	basicDir := &VirtualFile{
		Name:     "basic",
		IsDir:    true,
		Children: make(map[string]*VirtualFile),
		Parent:   guestDir,
		ModTime:  time.Now(),
	}
	guestDir.Children["basic"] = basicDir

	// Debug: Check if the home directory was created correctly
	if vfs.root.Children["home"].Children["guest"] == nil {
		return fmt.Errorf("Guest home directory was not created")
	}

	examplesDir := "examples"
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		return fmt.Errorf("Error reading examples directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)

		contentBytes, err := os.ReadFile(filepath.Join(examplesDir, name))
		if err != nil {
			vfsDebugLog("[GUEST] Error reading example file %s: %v", name, err)
			continue
		}

		var targetDir *VirtualFile
		// BASIC programs and SID files go to /basic subdirectory
		if strings.HasSuffix(lowerName, ".bas") || strings.HasSuffix(lowerName, ".sid") {
			targetDir = basicDir
		} else if strings.HasSuffix(lowerName, ".txt") {
			// Text files (like readme.txt) go to home directory
			targetDir = guestDir
		} else {
			// Skip other file types
			continue
		}

		// Create the file in the appropriate directory
		file := &VirtualFile{
			Name:     name,
			IsDir:    false,
			Content:  contentBytes,
			ModTime:  time.Now(),
			Parent:   targetDir,
			Children: nil}
		targetDir.Children[name] = file
	}

	// Check created files in both directories
	vfsDebugLog("[GUEST] Checking created files in guest home and basic directories")
	homeFileCount := 0
	basicFileCount := 0

	for name, file := range guestDir.Children {
		if file.IsDir {
			if name == "basic" {
				// Count files in basic directory
				for basicName, basicFile := range file.Children {
					if !basicFile.IsDir {
						basicFileCount++
						vfsDebugLog("[GUEST] Found in basic directory: %s", basicName)
					}
				}
			}
		} else {
			homeFileCount++
			vfsDebugLog("[GUEST] Found in guest home directory: %s", name)
		}
	}

	vfsDebugLog("[GUEST] Number of files in guest home directory: %d", homeFileCount)
	vfsDebugLog("[GUEST] Number of files in basic directory: %d", basicFileCount)

	vfsDebugLog("[GUEST] Guest VFS successfully initialized with example programs")
	return nil
}

func (vfs *VFS) InitializeGuestVFS() error {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()
	return vfs.initializeGuestVFSWithoutLock()
}

// CleanupGuestVFS deletes all files in the guest VFS
// This function should be called when a guest session ends
func (vfs *VFS) CleanupGuestVFS() error {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	// Delete files from memory
	if home, exists := vfs.root.Children["home"]; exists {
		if guestDir, exists := home.Children["guest"]; exists {
			// Clear the directory but keep the directory itself
			guestDir.Children = make(map[string]*VirtualFile)
			vfsDebugLog("[GUEST] Guest files successfully deleted from memory")
		}
	}

	// Resetting the entry in userRoots is optional, as InitializeGuestVFS recreates it if needed
	delete(vfs.userRoots, "guest")
	vfsDebugLog("[GUEST] Guest removed from userRoots")

	return nil
}

// SyncExamplePrograms synchronizes the example programs into a user's home directory
// Similar to InitializeGuestVFS, but for regular users
func (vfs *VFS) SyncExamplePrograms(username string) error {
	if username == "" || username == "guest" {
		return fmt.Errorf("invalid or empty username")
	}

	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	vfsDebugLog("[USER] SyncExamplePrograms called for user %s", username)
	// Create the home directory for the user if it doesn't exist
	homePath := "/home/" + username

	// Check if the home directory already exists
	homeDir, exists := vfs.root.Children["home"].Children[username]
	if !exists {
		vfsDebugLog("[USER] Home directory for %s does not exist, creating", username)
		var err error
		err = vfs.createDirectoryWithoutLock(homePath)
		if err != nil {
			return fmt.Errorf("error creating home directory: %v", err)
		}
		homeDir = vfs.root.Children["home"].Children[username]
	}

	// Create basic subdirectory for BASIC programs if it doesn't exist
	basicPath := homePath + "/basic"
	_, exists = homeDir.Children["basic"]
	if !exists {
		vfsDebugLog("[USER] Basic directory for %s does not exist, creating", username)
		err := vfs.createDirectoryWithoutLock(basicPath)
		if err != nil {
			return fmt.Errorf("error creating basic directory: %v", err)
		}
	}

	// Copy example programs into the appropriate directories
	examplesDir := "examples"
	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		return fmt.Errorf("error reading examples directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)

		contentBytes, err := os.ReadFile(filepath.Join(examplesDir, name))
		if err != nil {
			vfsDebugLog("[USER] Error reading example file %s: %v", name, err)
			continue
		}

		var targetPath string
		// BASIC programs and SID files go to /basic subdirectory
		if strings.HasSuffix(lowerName, ".bas") || strings.HasSuffix(lowerName, ".sid") {
			targetPath = basicPath + "/" + name
		} else if strings.HasSuffix(lowerName, ".txt") {
			// Text files (like readme.txt) go to home directory
			targetPath = homePath + "/" + name
		} else {
			// Skip other file types
			continue
		}

		err = vfs.createFileWithoutLock(targetPath, string(contentBytes), time.Now())
		if err != nil {
			vfsDebugLog("[USER] Error creating %s: %v", name, err)
			continue
		}
		vfsDebugLog("[USER] %s successfully created at %s", name, targetPath)
	}

	// Check created files
	vfsDebugLog("[USER] Checking created files in directory of %s", username)
	fileCount := len(homeDir.Children)
	vfsDebugLog("[USER] Number of files in directory: %d", fileCount)

	for name := range homeDir.Children {
		vfsDebugLog("[USER] Found in directory: %s", name)
	}

	vfsDebugLog("[USER] Example programs for %s successfully synchronized", username)
	return nil
}

// GetUserStorageInfo returns storage usage information for a user
func (vfs *VFS) GetUserStorageInfo(username string) (*UserStorageInfo, error) {
	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	// Calculate total used storage by traversing user's directory
	userHomePath := "/home/" + username
	userHome, remaining, err := vfs.resolvePathInternalWithoutLock(userHomePath)
	if err != nil || remaining != "" {
		return nil, fmt.Errorf("user home directory not found: %v", err)
	}

	if !userHome.IsDir {
		return nil, fmt.Errorf("user home path is not a directory: %s", userHomePath)
	}

	// Calculate used storage recursively
	usedBytes := vfs.calculateDirectorySize(userHome)
	usedKB := usedBytes / 1024
	if usedBytes%1024 > 0 {
		usedKB++ // Round up
	} // Calculate total available storage - use configured user quota
	totalKB := configuration.GetInt("FileSystem", "user_quota_kb", 10240) // Default 10MB

	return &UserStorageInfo{
		UsedKB:  usedKB,
		TotalKB: totalKB,
	}, nil
}

// calculateDirectorySize recursively calculates the total size of a directory
func (vfs *VFS) calculateDirectorySize(dir *VirtualFile) int {
	if dir == nil {
		return 0
	}

	totalSize := 0

	if !dir.IsDir {
		// If it's a file, return its content size
		return len(dir.Content)
	}

	// If it's a directory, sum up all children
	for _, child := range dir.Children {
		if child.IsDir {
			totalSize += vfs.calculateDirectorySize(child)
		} else {
			totalSize += len(child.Content)
		}
	}

	return totalSize
}
