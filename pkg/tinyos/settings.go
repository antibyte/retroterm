package tinyos

// Settings contains configuration constants for various functions

const (
	// MaxBasicSessions defines the maximum number of concurrent TinyBASIC sessions
	// When this limit is reached, no new BASIC sessions can be started
	MaxBasicSessions = 25

	// MaxGuestBasicSessions defines the maximum number of concurrent BASIC sessions for guest users
	// This is a sub-limit of the overall MaxBasicSessions limit
	MaxGuestBasicSessions = 5

	// SessionLimitMessage is the message displayed when the session limit is reached
	SessionLimitMessage = "Too many active sessions. Try later."
)
