package shared

// MessageType definiert den Typ einer Nachricht für die WebSocket-Kommunikation.
type MessageType int

// Konstanten für MessageType, angepasst an Frontend-Erwartungen (retroconsole.js RESPONSE_TYPE_MAP)
const (
	MessageTypeText         MessageType = 0  // Textausgabe
	MessageTypeClear        MessageType = 1  // Bildschirm löschen
	MessageTypeBeep         MessageType = 2  // Beep-Ton
	MessageTypeSpeak        MessageType = 3  // Sprachausgabe
	MessageTypeGraphics     MessageType = 4  // Grafikbefehl
	MessageTypeSound        MessageType = 5  // Soundkommando (allgemein, z.B. für benutzerdefinierte Frequenzen)
	MessageTypeSpeakDone    MessageType = 6  // Sprachausgabe beendet
	MessageTypeMode         MessageType = 7  // Moduswechsel (z.B. "os", "basic")
	MessageTypeSession      MessageType = 8  // Session-ID Übermittlung
	MessageTypeInputControl MessageType = 9  // Eingabesteuerung (aktivieren/deaktivieren)
	MessageTypeSprite       MessageType = 10 // Sprite-Befehl
	MessageTypeVector       MessageType = 11 // Vektorgrafik-Befehl
	MessageTypePrompt       MessageType = 12 // Prompt-Informationen (Symbol, Eingabestatus)
	MessageTypeNoise        MessageType = 13 // Noise-Sound-Befehl
	MessageTypeInput        MessageType = 14 // Eingabezeile aktualisieren (vom Backend zum Frontend)
	MessageTypeChat         MessageType = 15 // Chat-Modus aktivieren (startet Chat-WebSocket-Verbindung)
	MessageTypeKeyDown      MessageType = 16 // Taste gedrückt (für INKEY$)
	MessageTypeKeyUp        MessageType = 17 // Taste losgelassen (für INKEY$)
	MessageTypeLocate       MessageType = 18 // Cursor-Position setzen (LOCATE x,y)
	MessageTypeInverse      MessageType = 19 // Inverser Text-Modus (INVERSE ON/OFF)
	MessageTypeEditor       MessageType = 20 // Editor-Modus aktivieren/steuern
	MessageTypePager        MessageType = 21 // Pager-Modus aktivieren/deaktivieren
	MessageTypeCursor       MessageType = 22 // Cursor-Steuerung (show/hide)
	MessageTypeTelnet       MessageType = 23 // Telnet-Modus aktivieren/steuern
	MessageTypeAutoExecute  MessageType = 24 // Automatische Eingabe-Ausführung (autorun)
	MessageTypeBitmap       MessageType = 25 // Bitmap-Übertragung (PNG) mit Platzierung/Skalierung/Rotation
	MessageTypeEvil         MessageType = 26 // Evil effect - dramatic noise increase for MCP
	MessageTypeAuthRefresh  MessageType = 27 // Auth token refresh required (for temporary users)
	MessageTypeImage        MessageType = 28 // Image commands (LOAD, SHOW, HIDE, ROTATE)
	MessageTypeParticle     MessageType = 29 // Particle system commands
	MessageTypeSFX          MessageType = 30 // Sound effects via sfxr.js
	MessageTypePhysics      MessageType = 31 // Physics commands via Planck.js

	// MessageTypeError könnte hier mit einem Wert außerhalb des Frontend-Bereichs definiert werden, falls benötigt
	// z.B. MessageTypeError MessageType = 100
)

// Message repräsentiert eine Nachricht, die über WebSocket gesendet oder empfangen wird.
// Die Felder sind so strukturiert, dass sie den direkten Zugriffen im Frontend (retroconsole.js) entsprechen.
type Message struct {
	Type    MessageType `json:"type"` // Für TEXT, SPEAK, BEEP ("beep"), SOUND (freq,dur), NOISE ("noise:p,a,d")
	Content string      `json:"content"`
	// Für TEXT - verhindert automatischen Zeilenumbruch im Frontend
	NoNewline bool `json:"noNewline"`
	// Für PRINT/LOCATE: Inverser Text-Modus
	Inverse bool `json:"inverse"`

	// Für SESSION
	SessionID string `json:"sessionId,omitempty"` // Beibehaltung des Namens sessionId für Kompatibilität

	// Für SPEAK
	SpeechID int `json:"speechId,omitempty"` // Wird von speakTextWithID im Frontend erwartet

	// Für GRAPHICS, SPRITE, VECTOR
	Command string `json:"command,omitempty"` // z.B. "PLOT", "LINE", "DEFINE_SPRITE", "UPDATE_VECTOR"

	// Für GRAPHICS (wenn command vorhanden ist, werden diese als Parameter interpretiert)
	// Das Frontend (retroconsole.js) erwartet für GRAPHICS: response.command und response.params
	// Daher wird "Params" als map[string]interface{} verwendet.
	Params map[string]interface{} `json:"params,omitempty"`

	// Für SPRITE und VECTOR (und ggf. Grafikbefehle, die keine eigene "params" map nutzen)
	ID int `json:"id,omitempty"` // Sprite/Vector ID oder Definitions-ID

	// Sprite-spezifische Felder (Type == MessageTypeSprite)
	// Position (X, Y)
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`
	// Definitionsdaten
	PixelData   []int  `json:"pixelData,omitempty"`   // Für DEFINE_SPRITE
	SpriteData  string `json:"spriteData,omitempty"`  // Alternative für PixelData als String
	SpriteSheet string `json:"spriteSheet,omitempty"` // Für UPDATE_SPRITE (oft für definitionId missbraucht)
	// Update-Parameter
	DefinitionID int `json:"definitionId,omitempty"` // Explizit für UPDATE_SPRITE
	Rotation     int `json:"rotation,omitempty"`
	// Sichtbarkeit (Pointer, da optional und um zwischen false und nicht gesetzt zu unterscheiden)
	Visible *bool `json:"visible,omitempty"`
	// Für virtuelle Sprites
	Layout        string `json:"layout,omitempty"`
	BaseSpriteIDs []int  `json:"baseSpriteIds,omitempty"`

	// Vector-spezifische Felder (Type == MessageTypeVector)
	Shape string `json:"shape,omitempty"` // "cube", "pyramid", "sphere", "cylinder", "cone"
	// Position und Rotation für Vektoren als Maps, um {x,y,z} Strukturen abzubilden
	Position    map[string]float64 `json:"position,omitempty"`
	VecRotation map[string]float64 `json:"vecRotation,omitempty"` // Eigener Name, um Kollision mit Sprite-Rotation zu vermeiden
	Scale       interface{}        `json:"scale,omitempty"`       // Kann Zahl oder {x,y,z} Map sein
	Brightness  int                `json:"brightness,omitempty"`
	// Für erweiterte Shape-Parameter (Pyramid baseShape, Cylinder radius/height, etc.)
	CustomData map[string]interface{} `json:"customData,omitempty"`

	// Für INPUT (Type == MessageTypeInput)
	InputStr  string `json:"input,omitempty"` // "input" ist der Feldname im Frontend
	CursorPos int    `json:"cursorPos,omitempty"`
	// Für PROMPT (Type == MessageTypePrompt) oder INPUT_CONTROL (Type == MessageTypeInputControl)
	InputEnabled *bool  `json:"inputEnabled,omitempty"` // Pointer für optionale Booleans
	PromptSymbol string `json:"promptSymbol,omitempty"`
	// Für MODE (Type == MessageTypeMode)
	Mode string `json:"mode,omitempty"` // z.B. "os", "basic"

	// Für EDITOR (Type == MessageTypeEditor)
	EditorCommand string `json:"editorCommand,omitempty"` // "start", "stop", "render", "status", "key"
	EditorData    string `json:"editorData,omitempty"`    // Textiinhalt für "render", Tastencode für "key"
	CursorLine    int    `json:"cursorLine,omitempty"`    // Zeile für Cursor (0-based)
	CursorCol     int    `json:"cursorCol,omitempty"`     // Spalte für Cursor (0-based)
	ScrollLine    int    `json:"scrollLine,omitempty"`    // Scroll-Position (0-based)
	EditorRows    int    `json:"editorRows,omitempty"`    // Terminal-Höhe
	EditorCols    int    `json:"editorCols,omitempty"`    // Terminal-Breite
	EditorStatus  string `json:"editorStatus,omitempty"`  // Status-Text für Statuszeile
	EditorFile    string `json:"editorFile,omitempty"`    // Dateiname
	EditorMod     bool   `json:"editorMod,omitempty"`     // Datei wurde geändert

	// Für TELNET-Modus: Echo-Unterdrückung
	SuppressEcho bool   `json:"suppressEcho,omitempty"` // Unterdrückt lokales Echo in TELNET-Modus
	PagerPrompt  string `json:"pagerPrompt,omitempty"`  // Prompt-Text für Pager-Modus (z.B. cat)

	// Für BITMAP (Type == MessageTypeBitmap)
	BitmapData   string  `json:"bitmapData,omitempty"`   // Base64-encoded PNG data
	BitmapX      int     `json:"bitmapX,omitempty"`      // X position for bitmap placement
	BitmapY      int     `json:"bitmapY,omitempty"`      // Y position for bitmap placement
	BitmapScale  float64 `json:"bitmapScale,omitempty"`  // Scale factor (1.0 = original size)
	BitmapRotate float64 `json:"bitmapRotate,omitempty"` // Rotation in degrees
	BitmapID     string  `json:"bitmapId,omitempty"`     // Unique identifier for the bitmap
}

// TerminalMessage ist ein Alias, um alte Referenzen nicht sofort zu brechen.
// Es wird empfohlen, direkt Message zu verwenden.
// Deprecated: Verwende stattdessen Message.
type TerminalMessage = Message
