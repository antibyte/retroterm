package tinybasic

import (
	"strings"
)

// FileSystem defines the interface for interacting with a file system.
type FileSystem interface {
	ReadFile(path string, sessionID string) (string, error)
	WriteFile(path, content string, sessionID string) error
	Exists(path string, sessionID string) bool
	ListDirProgramFiles(sessionID string) ([]string, error) // Supports both .bas and .sid files
	ListDirAllFiles(sessionID string) ([]string, error)     // Returns all files
}

// OpenFile represents a file opened within the BASIC environment.
type OpenFile struct {
	Name     string   // Original filename.
	Mode     string   // "INPUT" or "OUTPUT".
	Lines    []string // Content split into lines (for INPUT mode).
	Pos      int      // Current line number index for reading (INPUT mode).
	WriteBuf []string // Buffer for lines written (OUTPUT mode).
}

// closeAllFiles iterates over open files and attempts to close them cleanly.
// Writes buffer for OUTPUT files. Called during cleanup actions.
// Assumes lock is held by the caller.
func (b *TinyBASIC) closeAllFiles() {
	if len(b.openFiles) == 0 {
		return // Nothing to close
	}
	closedCount := 0
	writeErrors := 0
	for _, of := range b.openFiles {
		// Attempt to write if output file has buffered data
		if of.Mode == "OUTPUT" && len(of.WriteBuf) > 0 {
			if b.fs == nil {
				writeErrors++
			} else {
				content := strings.Join(of.WriteBuf, "\n")
				// Add trailing newline standard for text files unless buffer was empty
				if content != "" || len(of.WriteBuf) > 0 { // Add if buffer wasn't technically empty (e.g., contained empty strings)
					content += "\n"
				}

				err := b.fs.WriteFile(of.Name, content, b.sessionID)
				if err != nil {
					// Log error but continue closing other files
					writeErrors++
				}
			}
		}
		closedCount++
		// No need to explicitly delete from map, it will be reset below
	}
	// Clear the map after iterating and attempting writes
	b.openFiles = make(map[int]*OpenFile)
	b.nextHandle = 1 // Reset handle counter
}
