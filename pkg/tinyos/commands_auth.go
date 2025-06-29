package tinyos

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"

	"golang.org/x/crypto/bcrypt"
)

// cmdPasswd changes the password for the current user
func (os *TinyOS) cmdPasswd(args []string) []shared.Message { // Extract sessionID from args if present
	sessionID := ""
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Check if user is logged in
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "You are not logged in.")
	}

	// Get current user
	username := os.GetUsernameBySessionID(sessionID)
	if username == "" {
		return os.CreateWrappedTextMessage(sessionID, "Session not found. Please log in again.")
	}

	// Check if user is a guest - guests cannot change passwords
	if os.isGuestSession(sessionID) {
		return os.CreateWrappedTextMessage(sessionID, "You are not logged in.")
	}

	// Start new password change process
	os.passwordChangeMutex.Lock()
	os.passwordChangeStates[sessionID] = &PasswordChangeState{
		Stage:     "current",
		Username:  username,
		CreatedAt: time.Now(),
	}
	os.passwordChangeMutex.Unlock()
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "=== CHANGE PASSWORD ==="},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter your current password:"},
		{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
		{Type: shared.MessageTypePrompt, Content: "Current password: "},
	}
}

// isInPasswordChangeProcess checks if a session is in a password change process
func (os *TinyOS) isInPasswordChangeProcess(sessionID string) bool {
	os.passwordChangeMutex.RLock()
	defer os.passwordChangeMutex.RUnlock()
	_, exists := os.passwordChangeStates[sessionID]
	return exists
}

// handlePasswordChangeInput processes input during the password change process
func (os *TinyOS) handlePasswordChangeInput(input string, sessionID string) []shared.Message {
	os.passwordChangeMutex.RLock()
	state, exists := os.passwordChangeStates[sessionID]
	os.passwordChangeMutex.RUnlock()

	if !exists {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: No active password change process found."},
		}
	}

	switch state.Stage {
	case "current":
		return os.handleCurrentPasswordInput(input, sessionID, state)
	case "new":
		return os.handleNewPasswordInput(input, sessionID, state)
	case "confirm":
		return os.handleConfirmNewPasswordInput(input, sessionID, state)
	default:
		// Unknown stage - reset password change
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Unknown password change status. Please start again with 'passwd'."},
		}
	}
}

// handleCurrentPasswordInput processes current password input
func (os *TinyOS) handleCurrentPasswordInput(input string, sessionID string, state *PasswordChangeState) []shared.Message {
	// Handle Ctrl+C (break signal) - cancel password change
	if input == "__BREAK__" {
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		logger.Info(logger.AreaAuth, "Password change cancelled by user break signal for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Password change cancelled."},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}

	currentPassword := input // Don't trim password as spaces might be valid
	// Validate current password against database
	if os.db != nil {
		var storedPasswordHash string
		err := os.db.QueryRow("SELECT password FROM users WHERE username = ?", state.Username).Scan(&storedPasswordHash)
		if err != nil {
			logger.Error(logger.AreaAuth, "Error querying user password for %s: %v", state.Username, err)
			// Clean up password change state
			os.passwordChangeMutex.Lock()
			delete(os.passwordChangeStates, sessionID)
			os.passwordChangeMutex.Unlock()
			return []shared.Message{
				{Type: shared.MessageTypeText, Content: "Error validating current password."},
			}
		}
		// Validate current password against stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(storedPasswordHash), []byte(currentPassword)); err != nil {
			logger.Info(logger.AreaAuth, "Password change failed for %s: incorrect current password", state.Username)
			// Clean up password change state
			os.passwordChangeMutex.Lock()
			delete(os.passwordChangeStates, sessionID)
			os.passwordChangeMutex.Unlock()
			return []shared.Message{
				{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
				{Type: shared.MessageTypeText, Content: "Current password is incorrect."},
				{Type: shared.MessageTypeText, Content: "Password change cancelled."},
			}
		}
	} else {
		// Clean up password change state
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Database not available."},
		}
	}

	// Current password is correct, move to next stage
	os.passwordChangeMutex.Lock()
	state.CurrentPassword = currentPassword
	state.Stage = "new"
	os.passwordChangeStates[sessionID] = state
	os.passwordChangeMutex.Unlock()
	return []shared.Message{
		{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
		{Type: shared.MessageTypeText, Content: "Current password verified."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter your new password:"},
		{Type: shared.MessageTypeText, Content: "Min. 4 characters, max. 20 characters"},
		{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
		{Type: shared.MessageTypePrompt, Content: "New password: "},
	}
}

// handleNewPasswordInput processes new password input
func (os *TinyOS) handleNewPasswordInput(input string, sessionID string, state *PasswordChangeState) []shared.Message {
	// Handle Ctrl+C (break signal) - cancel password change
	if input == "__BREAK__" {
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		logger.Info(logger.AreaAuth, "Password change cancelled by user break signal during new password entry for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Password change cancelled."},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}

	newPassword := input // Don't trim password as spaces might be valid
	// Validate new password
	if err := validatePassword(newPassword); err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Error: " + err.Error()},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
			{Type: shared.MessageTypePrompt, Content: "New password: "},
		}
	}

	// Check if new password is different from current password
	if newPassword == state.CurrentPassword {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Error: New password must be different from current password."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
			{Type: shared.MessageTypePrompt, Content: "New password: "},
		}
	}

	// Store new password and move to confirmation stage
	os.passwordChangeMutex.Lock()
	state.NewPassword = newPassword
	state.Stage = "confirm"
	os.passwordChangeStates[sessionID] = state
	os.passwordChangeMutex.Unlock()
	return []shared.Message{
		{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
		{Type: shared.MessageTypeText, Content: "New password accepted."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please confirm your new password:"},
		{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
		{Type: shared.MessageTypePrompt, Content: "Confirm password: "},
	}
}

// handleConfirmNewPasswordInput processes password confirmation input
func (os *TinyOS) handleConfirmNewPasswordInput(input string, sessionID string, state *PasswordChangeState) []shared.Message {
	// Handle Ctrl+C (break signal) - cancel password change
	if input == "__BREAK__" {
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		logger.Info(logger.AreaAuth, "Password change cancelled by user break signal during confirmation for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Password change cancelled."},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}

	confirmPassword := input // Don't trim password as spaces might be valid
	// Check if passwords match
	if confirmPassword != state.NewPassword {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Error: Passwords do not match."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
			{Type: shared.MessageTypePrompt, Content: "Confirm password: "},
		}
	}

	// Passwords match - update in database
	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(state.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(logger.AreaAuth, "Error hashing new password for %s: %v", state.Username, err) // Clean up password change state
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Error processing new password."},
		}
	}

	// Update password in database
	if os.db != nil {
		_, err := os.db.Exec("UPDATE users SET password = ? WHERE username = ?", newPasswordHash, state.Username)
		if err != nil {
			logger.Error(logger.AreaAuth, "Error updating password for %s: %v", state.Username, err)
			// Clean up password change state
			os.passwordChangeMutex.Lock()
			delete(os.passwordChangeStates, sessionID)
			os.passwordChangeMutex.Unlock()
			return []shared.Message{
				{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
				{Type: shared.MessageTypeText, Content: "Error updating password."},
			}
		}
	} else {
		// Clean up password change state
		os.passwordChangeMutex.Lock()
		delete(os.passwordChangeStates, sessionID)
		os.passwordChangeMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Database not available."},
		}
	}

	// Success - clean up password change state
	os.passwordChangeMutex.Lock()
	delete(os.passwordChangeStates, sessionID)
	os.passwordChangeMutex.Unlock()
	logger.Info(logger.AreaAuth, "Password successfully changed for user %s", state.Username)

	return []shared.Message{
		{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
		{Type: shared.MessageTypeText, Content: "Password confirmation accepted."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "✓ Password changed successfully!"},
		{Type: shared.MessageTypeText, Content: ""},
	}
}

// handleRegistrationInput processes input during the registration process
func (os *TinyOS) handleRegistrationInput(input string, sessionID string) []shared.Message {
	// Handle break signal - cancel registration
	if strings.ToLower(strings.TrimSpace(input)) == "__break__" {
		os.registrationMutex.Lock()
		delete(os.registrationStates, sessionID)
		os.registrationMutex.Unlock()
		logger.Info(logger.AreaAuth, "Registration cancelled by user for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode if active
			{Type: shared.MessageTypeText, Content: "Registration cancelled."},
		}
	}

	os.registrationMutex.RLock()
	state, exists := os.registrationStates[sessionID]
	os.registrationMutex.RUnlock()

	if !exists {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: No active registration process found."},
		}
	}

	switch state.Stage {
	case "username":
		return os.handleUsernameInput(input, sessionID, state)
	case "password":
		return os.handlePasswordInput(input, sessionID, state)
	case "confirm_password":
		return os.handleConfirmPasswordInput(input, sessionID, state)
	default:
		// Unknown stage - reset registration
		os.registrationMutex.Lock()
		delete(os.registrationStates, sessionID)
		os.registrationMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Unknown registration status. Please start again with 'register'."},
		}
	}
}

// handleUsernameInput processes username input
func (os *TinyOS) handleUsernameInput(input string, sessionID string, state *RegistrationState) []shared.Message {
	username := strings.TrimSpace(input)

	// Validate username
	if err := validateUsername(username); err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: " + err.Error()},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypePrompt, Content: "Username: "},
		}
	}

	// Check if user already exists
	if os.UserExists(username) {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: This username is already taken."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypePrompt, Content: "Username: "},
		}
	}

	// Username accepted, proceed to next step
	os.registrationMutex.Lock()
	state.Username = username
	state.Stage = "password"
	os.registrationMutex.Unlock()

	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "Username '" + username + "' is available."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter a password (minimum 4 characters):"},
		{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
		{Type: shared.MessageTypePrompt, Content: "Password: "},
	}
}

// handlePasswordInput processes password input
func (os *TinyOS) handlePasswordInput(input string, sessionID string, state *RegistrationState) []shared.Message {
	password := input // Don't trim password as spaces might be valid

	// Validate password
	if err := validatePassword(password); err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: " + err.Error()},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypePrompt, Content: "Password: "},
		}
	}

	// Password accepted, proceed to next step
	os.registrationMutex.Lock()
	state.Password = password
	state.Stage = "confirm_password"
	os.registrationMutex.Unlock()

	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "Password accepted."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please confirm your password:"},
		{Type: shared.MessageTypePrompt, Content: "Confirm password: "},
	}
}

// handleConfirmPasswordInput processes password confirmation
func (os *TinyOS) handleConfirmPasswordInput(input string, sessionID string, state *RegistrationState) []shared.Message {
	confirmPassword := input

	// Compare passwords
	if confirmPassword != state.Password {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Passwords do not match."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeText, Content: "Please enter the password again:"},
			{Type: shared.MessageTypePrompt, Content: "Password: "},
		}
	}

	// Perform registration
	os.AddRegistrationAttempt(state.IPAddress)
	err := os.RegisterUser(state.Username, state.Password, state.IPAddress)

	// Delete registration status
	os.registrationMutex.Lock()
	delete(os.registrationStates, sessionID)
	os.registrationMutex.Unlock()

	if err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Registration error: " + err.Error()},
		}
	}

	// Automatic login after successful registration
	messages, newSessionID, loginErr := os.LoginUser(state.Username, state.Password, state.IPAddress)
	if loginErr != nil {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "User successfully registered, but automatic login failed: " + loginErr.Error()},
			{Type: shared.MessageTypeText, Content: "Please log in manually with 'login " + state.Username + "'."},
		}
	}

	// Update SessionID for messages
	for i := range messages {
		messages[i].SessionID = newSessionID
	}
	// Disable password mode
	messages = append([]shared.Message{
		{Type: shared.MessageTypeInputControl, Content: "password_mode_off", SessionID: newSessionID},
	}, messages...)

	// Add session message for client to update session ID
	messages = append(messages, shared.Message{
		Type:      shared.MessageTypeSession,
		Content:   newSessionID,
		SessionID: newSessionID,
	})

	return messages
}

// isInRegistrationProcess checks if a session is in a registration process
func (os *TinyOS) isInRegistrationProcess(sessionID string) bool {
	os.registrationMutex.RLock()
	defer os.registrationMutex.RUnlock()
	_, exists := os.registrationStates[sessionID]
	return exists
}

// isInLoginProcess checks if a session is in a login process
func (os *TinyOS) isInLoginProcess(sessionID string) bool {
	os.loginMutex.RLock()
	defer os.loginMutex.RUnlock()
	_, exists := os.loginStates[sessionID]
	return exists
}

// handleLoginInput processes input during the login process
func (os *TinyOS) handleLoginInput(input string, sessionID string) []shared.Message {
	os.loginMutex.RLock()
	state, exists := os.loginStates[sessionID]
	os.loginMutex.RUnlock()

	if !exists {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: No active login process found."},
		}
	}

	switch state.Stage {
	case "username":
		return os.handleLoginUsernameInput(input, sessionID, state)
	case "password":
		return os.handleLoginPasswordInput(input, sessionID, state)
	default:
		// Unknown stage - reset login
		os.loginMutex.Lock()
		delete(os.loginStates, sessionID)
		os.loginMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Unknown login status. Please start again with 'login'."},
		}
	}
}

// handleLoginUsernameInput processes username input during login
func (os *TinyOS) handleLoginUsernameInput(input string, sessionID string, state *LoginState) []shared.Message {
	username := strings.TrimSpace(input)
	// Handle Ctrl+C (break signal) - cancel login
	if username == "__BREAK__" {
		os.loginMutex.Lock()
		delete(os.loginStates, sessionID)
		os.loginMutex.Unlock()
		logger.Info(logger.AreaAuth, "Login cancelled by user break signal for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Login cancelled."},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}

	if username == "" {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Username cannot be empty."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypePrompt, Content: "Username: "},
		}
	}

	// Store username and move to password stage
	os.loginMutex.Lock()
	state.Username = username
	state.Stage = "password"
	os.loginMutex.Unlock()
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter password:"},
		{Type: shared.MessageTypeInputControl, Content: "password_mode_on"}, // Enable password mode
		{Type: shared.MessageTypePrompt, Content: "Password: "},
	}
}

// handleLoginPasswordInput processes password input during login
func (os *TinyOS) handleLoginPasswordInput(input string, sessionID string, state *LoginState) []shared.Message {
	password := strings.TrimSpace(input)

	// Handle Ctrl+C (break signal) - cancel login
	if password == "__BREAK__" {
		os.loginMutex.Lock()
		delete(os.loginStates, sessionID)
		os.loginMutex.Unlock()
		logger.Info(logger.AreaAuth, "Login cancelled by user break signal during password entry for session %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Login cancelled."},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}

	if password == "" {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Password cannot be empty."},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypePrompt, Content: "Password: "},
		}
	}

	// Attempt login
	username := state.Username
	ipAddress := state.IPAddress

	// Clean up login state
	os.loginMutex.Lock()
	delete(os.loginStates, sessionID)
	os.loginMutex.Unlock()
	// Perform login via the LoginUser method
	messages, newSessionID, err := os.LoginUser(username, password, ipAddress)
	if err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
			{Type: shared.MessageTypeText, Content: "Login error: " + err.Error()},
			{Type: shared.MessageTypeText, Content: ""},
		}
	} // Disable password mode and add session ID in response for client-side storage
	messages = append([]shared.Message{
		{Type: shared.MessageTypeInputControl, Content: "password_mode_off"}, // Disable password mode
	}, messages...)
	messages = append(messages, shared.Message{
		Type:      shared.MessageTypeSession,
		SessionID: newSessionID,
	})

	return messages
}

// cmdLoginNew starts the interactive login process
func (os *TinyOS) cmdLoginNew(args []string, sessionID string) []shared.Message {
	// Get IP address - use default since we don't have request context here
	ipAddress := "127.0.0.1" // Default value

	// Check if already in login process
	if os.isInLoginProcess(sessionID) {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Login process already in progress. Please complete it or restart."},
		}
	}

	// Start new login process
	loginState := &LoginState{
		Stage:     "username",
		Username:  "",
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}
	os.loginMutex.Lock()
	os.loginStates[sessionID] = loginState
	os.loginMutex.Unlock()

	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "Please enter your login credentials."},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter username:"},
		{Type: shared.MessageTypePrompt, Content: "Username: "},
	}
}

// validateUsername validates the username
func validateUsername(username string) error {
	minLen := configuration.GetInt("Authentication", "min_username_length", 3)
	maxLen := configuration.GetInt("Authentication", "max_username_length", 20)

	if len(username) == 0 {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) < minLen {
		return fmt.Errorf("username must be at least %d characters long", minLen)
	}
	if len(username) > maxLen {
		return fmt.Errorf("username cannot be longer than %d characters", maxLen)
	}

	// Allowed characters: letters, numbers, underscore, hyphen
	for _, r := range username {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("username may only contain letters, numbers, underscores and hyphens")
		}
	}
	// Check reserved names and control signals
	reservedNames := []string{"admin", "root", "system", "guest", "user", "test", "demo", "api", "www", "mail", "ftp", "__break__", "__ctrl__", "__cancel__"}
	lowerUsername := strings.ToLower(username)
	for _, reserved := range reservedNames {
		if lowerUsername == reserved {
			return fmt.Errorf("this username is reserved")
		}
	}

	return nil
}

// validatePassword validates the password
func validatePassword(password string) error {
	if len(password) < 4 {
		return fmt.Errorf("password must be at least 4 characters long")
	}
	if len(password) > 50 {
		return fmt.Errorf("password cannot be longer than 50 characters")
	}
	return nil
}

// cmdLogin logs in a user and uses the SQLite database
func (os *TinyOS) cmdLogin(args []string) []shared.Message {
	// For login, no session is active yet, so empty sessionID
	sessionID := ""

	if len(args) < 2 {
		return os.CreateWrappedTextMessage(sessionID, "Usage: login <username> <password>")
	}
	username := args[0]
	password := args[1]
	ipAddress := "127.0.0.1" // Default value, should be replaced by actual client IP

	// Log in user via the LoginUser method of the TinyOS structure
	messages, newSessionID, err := os.LoginUser(username, password, ipAddress)
	if err != nil {
		return os.CreateWrappedTextMessage(sessionID, "Login error: "+err.Error())
	}

	// Update sessionID after successful login
	sessionID = newSessionID

	// Add session ID in response for client-side storage
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeSession,
		Content: sessionID,
	})
	// Use the returned messages instead of creating new ones

	// Use the returned messages instead of creating new ones
	return messages
}

// cmdLogout logs out the user
func (os *TinyOS) cmdLogout(args []string) []shared.Message {
	// Check if a SessionID was passed in the argument
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}
	// If no session ID was provided, we can't log out
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "Session ID not provided. Please log in again.")
	}
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		return os.CreateWrappedTextMessage(sessionID, "No valid session found. Please log in again.")
	}

	// Retrieve IP address from the session for the new guest session
	var ipAddress string = "127.0.0.1" // Default value
	os.sessionMutex.RLock()
	if session, exists := os.sessions[sessionID]; exists {
		ipAddress = session.IPAddress
	}
	os.sessionMutex.RUnlock()

	// Abmeldung durchführen
	os.sessionMutex.Lock()
	delete(os.sessions, sessionID)
	os.sessionMutex.Unlock()

	// Aktualisiere den Login-Status in der Datenbank
	if os.db != nil {
		_, err := os.db.Exec("UPDATE users SET is_logged_in = 0 WHERE username = ?", username)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Aktualisieren des Logout-Status: %v", err)
		}

		// Lösche die Session aus der Datenbank
		_, err = os.db.Exec("DELETE FROM user_sessions WHERE session_id = ?", sessionID)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Löschen der Session aus der Datenbank: %v", err)
		}
	}

	// Erstelle automatisch eine neue Gast-Session mit derselben SessionID
	// Dadurch kann der Benutzer nahtlos als Gast weiterarbeiten
	_, err := os.CreateGuestSession(sessionID, ipAddress)
	if err != nil {
		// Trotzdem erfolgreich abmelden, auch wenn Gast-Session-Erstellung fehlschlägt
		return os.CreateWrappedTextMessage("", "User "+username+" has been logged out.")
	}

	logMessage("[TINYOS] User %s logged out and automatically continued as guest session %s", username, sessionID)

	// Rückgabe mit mehreren Nachrichten: Abmeldebestätigung und neuer Session-Status
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "User " + username + " has been logged out."},
		{Type: shared.MessageTypeText, Content: "You can continue to use the terminal as a guest."},
		{Type: shared.MessageTypeSession, Content: sessionID}, // SessionID bleibt gleich, aber jetzt als Gast
	}
}

// cmdWhoAmI zeigt den aktuellen Benutzernamen an
func (os *TinyOS) cmdWhoAmI(args []string) []shared.Message {
	// Überprüfen, ob eine SessionID im Argument übergeben wurde
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Wenn eine Session-ID angegeben wurde, verwenden wir diese
	if sessionID != "" {
		username := os.GetUsernameForSession(sessionID)
		if username == "" {
			return os.CreateWrappedTextMessage(sessionID, "No valid session found.")
		}
		return os.CreateWrappedTextMessage(sessionID, "Logged in as: "+username)
	}

	// Wenn keine Session-ID angegeben wurde
	return os.CreateWrappedTextMessage("", "No session ID provided. Please log in.")
}

// cmdRegisterNew starts the new multi-step registration process
func (os *TinyOS) cmdRegisterNew(args []string, r *http.Request, sessionID string) []shared.Message {
	// Registration requires a valid session - reject empty sessionID
	if sessionID == "" {
		logger.Warn(logger.AreaAuth, "Registration attempt without valid session ID - potential security issue")
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Invalid session. Please refresh and try again."},
		}
	}

	// If old parameters were passed, use the old process
	if len(args) >= 2 {
		return os.cmdRegister(args, r, sessionID)
	}

	// Check if user is already logged in
	if sessionID != "" {
		username := os.GetUsernameBySessionID(sessionID)
		if username != "" && !os.isGuestSession(sessionID) {
			return os.CreateWrappedTextMessage(sessionID, "You are already logged in as '"+username+"'. Please logout first if you want to register a new account.")
		}
	}
	// Default IP address for local registrations
	ipAddress := "127.0.0.1"
	if r != nil {
		ipAddress = r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ipAddress = forwardedFor
		}
	}
	// Check if IP address is restricted
	if os.IsIPRestricted(ipAddress) {
		return os.CreateWrappedTextMessage(sessionID, "Registration is not allowed twice within 24 hours.")
	}

	// Start new registration
	os.registrationMutex.Lock()
	os.registrationStates[sessionID] = &RegistrationState{
		Stage:     "username",
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}
	os.registrationMutex.Unlock()

	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "=== USER REGISTRATION ==="},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Please enter a username (max. 20 characters):"},
		{Type: shared.MessageTypeText, Content: "Allowed: letters, numbers, underscores, hyphens"},
		{Type: shared.MessageTypePrompt, Content: "Username: "},
	}
}

// cmdRegister registers a new user and uses the SQLite database
func (os *TinyOS) cmdRegister(args []string, r *http.Request, sessionID string) []shared.Message {
	// Registration requires a valid session - reject empty sessionID
	if sessionID == "" {
		logger.Warn(logger.AreaAuth, "Registration attempt without valid session ID - potential security issue")
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Invalid session. Please refresh and try again."},
		}
	}

	// Check if user is already logged in
	if sessionID != "" {
		username := os.GetUsernameBySessionID(sessionID)
		if username != "" && !os.isGuestSession(sessionID) {
			return os.CreateWrappedTextMessage(sessionID, "You are already logged in as '"+username+"'. Please logout first if you want to register a new account.")
		}
	}

	if len(args) < 2 {
		return os.CreateWrappedTextMessage(sessionID, "Usage: register <username> <password>")
	}
	username := args[0]
	password := args[1]
	logger.Debug(logger.AreaAuth, "Register command called for username: %s", username)

	// Default IP address for local registrations
	ipAddress := "127.0.0.1"

	// If an HTTP request is present, extract the actual IP address
	if r != nil {
		ipAddress = r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ipAddress = forwardedFor
		}
	}
	// Check if the IP address is restricted
	if os.IsIPRestricted(ipAddress) {
		return os.CreateWrappedTextMessage(sessionID, "Registration from this IP address is not allowed again within 24 hours.")
	}
	os.AddRegistrationAttempt(ipAddress)
	// Register user via the RegisterUser method of the TinyOS structure
	logger.Debug(logger.AreaAuth, "Calling RegisterUser for: %s", username)
	err := os.RegisterUser(username, password, ipAddress)
	if err != nil {
		logger.Error(logger.AreaAuth, "RegisterUser failed for %s: %v", username, err)
		return os.CreateWrappedTextMessage(sessionID, "Error while registering user: "+err.Error())
	}
	logger.Info(logger.AreaAuth, "User %s successfully registered", username)
	// Nach erfolgreicher Registrierung automatisch einloggen
	logger.Debug(logger.AreaAuth, "Attempting automatic login for: %s", username)
	messages, newSessionID, loginErr := os.LoginUser(username, password, ipAddress)
	if loginErr != nil {
		logger.Error(logger.AreaAuth, "Automatic login failed for %s: %v", username, loginErr)
		return os.CreateWrappedTextMessage(sessionID, "User successfully registered, but automatic login failed: "+loginErr.Error())
	}
	logger.Info(logger.AreaAuth, "Automatic login successful for %s, new sessionID: %s", username, newSessionID)

	// Aktualisiere die sessionID mit der vom Login erhaltenen
	sessionID = newSessionID

	// Session-ID in der Antwort hinzufügen für Client-seitige Speicherung
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeSession,
		Content: sessionID,
	})

	return messages
}
