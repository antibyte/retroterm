package tinybasic

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Sprite-Befehle für TinyBASIC

// Standard-Sprite-Größe: 32x32 Pixel, 16 Helligkeitswerte pro Pixel (0-15), 0 bedeutet transparent
const (
	SpriteSize              = 32
	MaxSpriteID             = 255
	MaxSpriteInstances      = 256 // erhöht von 100 auf 256 für Space Invaders
	MaxVirtualSpriteEntries = 16
	SpriteBatchMaxSize      = 32 // Maximum sprites per batch
	SpriteBatchTimeout      = 5  // Milliseconds to wait before auto-sending batch
)

// Hilfsfunktion zum Senden von Sprite-Kommandos
// Die Struktur 'cmd' wird nun direkt in die Felder von shared.Message gemappt.
// Die genaue Struktur von 'cmd' hängt vom jeweiligen Sprite-Befehl ab.
func (b *TinyBASIC) sendSpriteCommand(commandType string, instanceID int, details map[string]interface{}) {
	msg := shared.Message{
		Type:      shared.MessageTypeSprite,
		Command:   commandType, // z.B. "DEFINE_SPRITE", "UPDATE_SPRITE"
		ID:        instanceID,  // Korrigiert von SpriteID zu ID (für Instanz-ID)
		SessionID: b.sessionID, // SessionID hinzufügen
	}
	// Setze spezifische Felder basierend auf 'details'
	if pixelData, ok := details["pixelData"].([]int); ok {
		msg.PixelData = pixelData // Direkt []int zuweisen, wenn Message.PixelData []int ist
	}
	if defID, ok := details["definitionId"].(int); ok {
		msg.DefinitionID = defID // Verwende das korrekte Feld DefinitionID
	}
	if x, ok := details["x"].(int); ok {
		msg.X = x // Korrigiert von X1 zu X
	}
	if y, ok := details["y"].(int); ok {
		msg.Y = y // Korrigiert von Y1 zu Y
	}
	if rot, ok := details["rotation"].(int); ok {
		msg.Rotation = rot // Korrigiert von X2 zu Rotation
	}
	if vis, ok := details["visible"].(bool); ok {
		visiblePtr := new(bool)
		*visiblePtr = vis
		msg.Visible = visiblePtr // Korrigiert von Fill zu Visible (*bool)
	}
	if layout, ok := details["layout"].(string); ok {
		msg.Layout = layout
	}
	if baseIDs, ok := details["baseSpriteIds"].([]int); ok {
		msg.BaseSpriteIDs = baseIDs
	}

	// Debug-Ausgabe deaktiviert um Log-Größe zu reduzieren
	// fmt.Printf("[SPRITE-DEBUG] Sende %s Nachricht: ID=%d, SessionID=%s\n", commandType, instanceID, b.sessionID)
	b.addToBatch(msg)
}

// cmdDefineSprite implementiert den SPRITE-Befehl: SPRITE id, pixelData
// Definiert oder aktualisiert eine Sprite-Definition
func (b *TinyBASIC) cmdDefineSprite(args string) error {

	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 2 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint("SPRITE requires at least 2 parameters: SPRITE id, pixelData")
	}

	// ID auswerten
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint("Sprite ID parameter must be a valid numeric expression")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint("Sprite ID must be numeric")
	}
	if id < 0 || id > MaxSpriteID {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint(fmt.Sprintf("Sprite ID must be between 0 and %d", MaxSpriteID))
	}
	// Pixel-Daten auswerten - muss ein String-Array sein
	pixelDataExpr := strings.TrimSpace(params[1])
	pixelDataVal, err := b.evalExpression(pixelDataExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_PIXELDATA_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint("PixelData parameter must be a valid string expression")
	}

	if pixelDataVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_PIXELDATA_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint("PixelData must be a string (e.g., A$), not a numeric value")
	}
	// fmt.Printf("[Go DEBUG] cmdDefineSprite: pixelDataVal is a string. Content preview (up to 50 chars): '%.50s...'\\n", pixelDataVal.StrValue)

	// Pixel-Daten in Uint8Array umwandeln (erwartet wird ein String im Format "0,1,2,3,...")
	pixelDataStr := pixelDataVal.StrValue
	pixelDataStr = strings.Trim(pixelDataStr, "\"' \t")
	pixelDataStrings := strings.Split(pixelDataStr, ",")
	// Überprüfen der Datengröße
	expectedSize := SpriteSize * SpriteSize
	if len(pixelDataStrings) != expectedSize {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_DATA_SIZE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE").
			WithUsageHint(fmt.Sprintf("Sprite data must contain exactly %d values (%dx%d), but received %d", expectedSize, SpriteSize, SpriteSize, len(pixelDataStrings)))
	} // Umwandlung in numerische Werte
	pixelDataNum := make([]int, expectedSize)
	uniqueValues := make(map[int]int) // value -> count
	for i, sVal := range pixelDataStrings {
		trimmedSVal := strings.TrimSpace(sVal)
		val, errConv := parseBasicInt(trimmedSVal)
		if errConv != nil {
			return NewBASICError(ErrCategoryEvaluation, "SPRITE_PIXEL_VALUE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE").
				WithUsageHint(fmt.Sprintf("Invalid pixel value at position %d: '%s'", i, trimmedSVal))
		}
		if val < 0 || val > 15 {
			return NewBASICError(ErrCategoryEvaluation, "SPRITE_PIXEL_RANGE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE").
				WithUsageHint(fmt.Sprintf("Pixel values must be between 0 and 15, got %d", val))
		}
		pixelDataNum[i] = val
		uniqueValues[val]++
	}

	// Debug: Zeige verwendete Helligkeitswerte an

	// Sprite-Definition an das Frontend senden
	b.sendSpriteCommand("DEFINE_SPRITE", id, map[string]interface{}{
		"pixelData": pixelDataNum, // pixelDataNum ist []int
	})

	// Cache Pixel-Daten für Kollisionserkennung
	cacheSpritePixelData(id, pixelDataNum)

	// Kleine Verzögerung nach DEFINE_SPRITE, damit das Frontend Zeit hat, die Definition zu verarbeiten
	// Dies löst das Timing-Problem bei schnell aufeinanderfolgenden Definitionen wie in invaders.bas
	time.Sleep(10 * time.Millisecond)

	return nil
}

// cmdUpdateSprite implementiert den SPRITE UPDATE-Befehl: SPRITE UPDATE id, definitionId, x, y, [rotation], [visible]
// Aktualisiert die Position und andere Parameter einer Sprite-Instanz
func (b *TinyBASIC) cmdUpdateSprite(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 4 || len(params) > 6 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE UPDATE").
			WithUsageHint("SPRITE UPDATE requires 4-6 parameters: SPRITE UPDATE id, definitionId, x, y, [rotation], [visible]")
	}

	valueNames := []string{"id", "definitionId", "x", "y", "rotation", "visible"}
	values := make([]int, len(params))
	for i, param := range params {
		expr := strings.TrimSpace(param)
		val, err := b.evalExpression(expr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "SPRITE_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE UPDATE").
				WithUsageHint(fmt.Sprintf("Error in %s parameter", valueNames[i]))
		}
		intVal, err := basicValueToInt(val)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "SPRITE_PARAM_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE UPDATE").
				WithUsageHint(fmt.Sprintf("%s must be numeric", valueNames[i]))
		}
		values[i] = intVal
	}

	// Validierung
	if values[0] < 0 || values[0] > MaxSpriteInstances {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_INSTANCE_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE UPDATE").
			WithUsageHint(fmt.Sprintf("Sprite instance ID must be between 0 and %d", MaxSpriteInstances))
	}
	if values[1] < 0 || values[1] > MaxSpriteID {
		return NewBASICError(ErrCategoryEvaluation, "SPRITE_DEFINITION_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE UPDATE").
			WithUsageHint(fmt.Sprintf("Sprite definition ID must be between 0 and %d", MaxSpriteID))
	}

	// Standard-Werte für optionale Parameter werden direkt in 'details' gesetzt.
	details := map[string]interface{}{
		"definitionId": values[1],
		"x":            values[2],
		"y":            values[3],
		"rotation":     0,    // Standardwert für Rotation
		"visible":      true, // Standardwert für Visible
	}

	if len(params) >= 5 {
		details["rotation"] = values[4]
	}
	if len(params) >= 6 {
		details["visible"] = (values[5] != 0)
	}
	// Registriere Sprite-Position für Kollisionserkennung
	registerSpritePosition(values[0], values[1], values[2], values[3], details["visible"].(bool))

	b.sendSpriteCommand("UPDATE_SPRITE", values[0], details)

	return nil
}

// cmdDefineVirtualSprite implementiert den SPRITE VIRTUAL-Befehl: SPRITE VIRTUAL id, layout, baseSpriteId1, baseSpriteId2, ...
// Definiert ein virtuelles Sprite, das aus mehreren Basis-Sprites besteht
func (b *TinyBASIC) cmdDefineVirtualSprite(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 3 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("SPRITE VIRTUAL requires at least 3 parameters: SPRITE VIRTUAL id, layout, baseSpriteId1, [baseSpriteId2, ...]")
	}

	// ID auswerten
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VIRTUAL_SPRITE_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("Virtual sprite ID parameter must be a valid numeric expression")
	}

	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VIRTUAL_SPRITE_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("Virtual sprite ID must be numeric")
	}

	if id < 0 || id > MaxSpriteID {
		return NewBASICError(ErrCategoryEvaluation, "VIRTUAL_SPRITE_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint(fmt.Sprintf("Virtual sprite ID must be between 0 and %d", MaxSpriteID))
	}

	// Layout auswerten
	layoutExpr := strings.TrimSpace(params[1])
	layoutVal, err := b.evalExpression(layoutExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "LAYOUT_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("Layout parameter must be a valid string expression")
	}

	if layoutVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "LAYOUT_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("Layout must be a string like \"2x2\" or \"4x4\"")
	}
	layout := strings.Trim(layoutVal.StrValue, "\"' \t")
	if layout != "2x2" && layout != "4x4" {
		return NewBASICError(ErrCategoryEvaluation, "LAYOUT_VALUE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint("Layout must be either \"2x2\" or \"4x4\"")
	}

	// Basis-Sprite-IDs auswerten
	baseSpriteIds := make([]int, len(params)-2)
	expectedCount := 0
	if layout == "2x2" {
		expectedCount = 4
	} else if layout == "4x4" {
		expectedCount = 16
	}
	if len(params)-2 != expectedCount {
		return NewBASICError(ErrCategorySyntax, "VIRTUAL_SPRITE_COUNT_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("SPRITE VIRTUAL").
			WithUsageHint(fmt.Sprintf("Layout %s requires exactly %d base sprite IDs", layout, expectedCount))
	}

	for i := 0; i < len(params)-2; i++ {
		expr := strings.TrimSpace(params[i+2])
		val, err := b.evalExpression(expr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "BASE_SPRITE_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE VIRTUAL").
				WithUsageHint(fmt.Sprintf("Error in baseSpriteId parameter %d", i+1))
		}

		baseId, err := basicValueToInt(val)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "BASE_SPRITE_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE VIRTUAL").
				WithUsageHint(fmt.Sprintf("BaseSpriteId %d must be numeric", i+1))
		}

		if baseId < 0 || baseId > MaxSpriteID {
			return NewBASICError(ErrCategoryEvaluation, "BASE_SPRITE_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("SPRITE VIRTUAL").
				WithUsageHint(fmt.Sprintf("BaseSpriteId %d must be between 0 and %d", i+1, MaxSpriteID))
		}

		baseSpriteIds[i] = baseId
	}

	// Virtuelles Sprite an das Frontend senden
	b.sendSpriteCommand("DEFINE_VIRTUAL_SPRITE", id, map[string]interface{}{
		"layout":        layout,
		"baseSpriteIds": baseSpriteIds,
	})

	return nil
}

// cmdSprite verarbeitet die verschiedenen Sprite-Befehle:
// - SPRITE id, pixelData (Definition eines Sprites)
// - SPRITE UPDATE id, definitionId, x, y, [rotation], [visible] (Platzierung/Update)
// - SPRITE VIRTUAL id, layout, baseSpriteId1, baseSpriteId2, ... (Virtuelles Sprite)
func (b *TinyBASIC) cmdSprite(args string) error {
	// DEBUG: Log that cmdSprite was called (disabled for performance)
	// fmt.Printf("[DEBUG-SPRITE] cmdSprite called with args: '%s'\n", args)

	// Überprüfen auf Unterbefehle (UPDATE, VIRTUAL)
	upperArgs := strings.ToUpper(args)
	if strings.HasPrefix(upperArgs, "UPDATE ") {
		// fmt.Printf("[DEBUG-SPRITE] SPRITE UPDATE command detected\n")
		return b.cmdUpdateSprite(strings.TrimSpace(args[6:])) // Nach "UPDATE " abschneiden
	}

	if strings.HasPrefix(upperArgs, "VIRTUAL ") {
		// fmt.Printf("[DEBUG-SPRITE] SPRITE VIRTUAL command detected\n")
		return b.cmdDefineVirtualSprite(strings.TrimSpace(args[7:])) // Nach "VIRTUAL " abschneiden
	}

	if strings.HasPrefix(upperArgs, "PHYSICS ") {
		// Handle SPRITE PHYSICS command
		return b.handleSpritePhysicsCommand(strings.TrimSpace(args[8:])) // Nach "PHYSICS " abschneiden
	}

	// Standard SPRITE-Befehl (Definition)
	// fmt.Printf("[DEBUG-SPRITE] SPRITE DEFINE command detected\n")
	return b.cmdDefineSprite(args)
}

// Hilfsfunktion für Umwandlung String in Int
func parseBasicInt(s string) (int, error) {
	return basicValueToInt(BASICValue{
		StrValue:  s,
		IsNumeric: false,
	})
}

// Sprite-Batching für Performance-Optimierung
func (b *TinyBASIC) initSpriteBatching() {
	b.spriteBatchMutex.Lock()
	defer b.spriteBatchMutex.Unlock()

	b.spriteBatch = make([]shared.Message, 0, SpriteBatchMaxSize)
	b.batchingEnabled = true
	b.spriteBatchTimer = nil
}

func (b *TinyBASIC) addToBatch(msg shared.Message) {
	b.spriteBatchMutex.Lock()
	defer b.spriteBatchMutex.Unlock()

	if !b.batchingEnabled {
		// Batching disabled, send immediately
		b.sendMessageObject(msg)
		return
	}

	b.spriteBatch = append(b.spriteBatch, msg)

	// Send batch if full
	if len(b.spriteBatch) >= SpriteBatchMaxSize {
		b.flushBatchUnsafe()
		return
	}

	// Set/reset timer for auto-flush
	if b.spriteBatchTimer != nil {
		b.spriteBatchTimer.Stop()
	}
	b.spriteBatchTimer = time.AfterFunc(SpriteBatchTimeout*time.Millisecond, func() {
		b.flushBatch()
	})
}

func (b *TinyBASIC) flushBatch() {
	b.spriteBatchMutex.Lock()
	defer b.spriteBatchMutex.Unlock()
	b.flushBatchUnsafe()
}

func (b *TinyBASIC) flushBatchUnsafe() {
	if len(b.spriteBatch) == 0 {
		return
	}

	// Send batch as individual messages (could be optimized further with batch message type)
	for _, msg := range b.spriteBatch {
		b.sendMessageObject(msg)
	}

	// Clear batch
	b.spriteBatch = b.spriteBatch[:0]

	// Stop timer
	if b.spriteBatchTimer != nil {
		b.spriteBatchTimer.Stop()
		b.spriteBatchTimer = nil
	}
}

// Kollisionserkennung-Strukturen
type SpritePosition struct {
	ID           int
	DefinitionID int
	X            int
	Y            int
	Visible      bool
	LastSeen     time.Time
	// IsStationary Feld entfernt - alle Sprites bleiben permanent aktiv
}

type SpriteCollisionInfo struct {
	SpriteID      int
	CollidingWith []int // Liste der Sprite-IDs mit denen kollidiert wird
	LastChecked   time.Time
}

// Globale Kollisionsdaten
var (
	activeSprites     = make(map[int]*SpritePosition)      // spriteID -> Position
	spriteCollisions  = make(map[int]*SpriteCollisionInfo) // spriteID -> CollisionInfo
	collisionCacheTTL = 16 * time.Millisecond              // ~60 FPS Cache
	spriteMutex       sync.RWMutex                         // Schützt Sprite-Daten
)

// Registriere ein Sprite für Kollisionserkennung
func registerSpritePosition(id, definitionID, x, y int, visible bool) {
	spriteMutex.Lock()
	defer spriteMutex.Unlock()

	now := time.Now()

	// Wenn Sprite bereits existiert, aktualisiere nur Position und Zeit
	if existing, exists := activeSprites[id]; exists {
		existing.X = x
		existing.Y = y
		existing.Visible = visible
		existing.LastSeen = now
		return
	}

	// Neues Sprite registrieren - bleibt permanent aktiv
	activeSprites[id] = &SpritePosition{
		ID:           id,
		DefinitionID: definitionID,
		X:            x,
		Y:            y,
		Visible:      visible,
		LastSeen:     now,
	}
}

// Kollisionserkennung zwischen zwei Sprites (zweistufig für Genauigkeit)
func checkSpriteCollision(sprite1, sprite2 *SpritePosition) bool {
	if sprite1 == nil || sprite2 == nil || !sprite1.Visible || !sprite2.Visible {
		return false
	}

	// Erste Stufe: Schnelle Bounding-Box-Kollision (32x32 Sprites)
	x1, y1 := float64(sprite1.X), float64(sprite1.Y)
	x2, y2 := float64(sprite2.X), float64(sprite2.Y)
	size := float64(SpriteSize)

	// Prüfe ob sich die 32x32 Bounding-Boxes überschneiden
	if !(x1 < x2+size && x1+size > x2 && y1 < y2+size && y1+size > y2) {
		return false // Keine Überschneidung der Bounding-Boxes
	}

	// Zweite Stufe: Pixel-genaue Kollisionsprüfung
	return checkPixelCollision(sprite1, sprite2)
}

// Pixel-genaue Kollisionsprüfung zwischen zwei Sprites
func checkPixelCollision(sprite1, sprite2 *SpritePosition) bool {
	// Berechne Überschneidungsbereich
	x1, y1 := sprite1.X, sprite1.Y
	x2, y2 := sprite2.X, sprite2.Y

	// Grenzen des Überschneidungsbereichs
	overlapLeft := max(x1, x2)
	overlapTop := max(y1, y2)
	overlapRight := min(x1+SpriteSize, x2+SpriteSize)
	overlapBottom := min(y1+SpriteSize, y2+SpriteSize)

	// Keine Überschneidung (sollte nicht passieren, da Bounding-Box bereits geprüft)
	if overlapLeft >= overlapRight || overlapTop >= overlapBottom {
		return false
	}

	// Hole Sprite-Pixel-Daten (falls verfügbar)
	pixels1 := getSpritePixelData(sprite1.DefinitionID)
	pixels2 := getSpritePixelData(sprite2.DefinitionID)

	// Falls keine Pixel-Daten verfügbar, verwende Bounding-Box-Kollision
	if pixels1 == nil || pixels2 == nil {
		return true // Bounding-Box-Kollision ist bereits bestätigt
	}

	// Prüfe jeden Pixel im Überschneidungsbereich
	for y := overlapTop; y < overlapBottom; y++ {
		for x := overlapLeft; x < overlapRight; x++ {
			// Lokale Koordinaten für beide Sprites
			local1X := x - x1
			local1Y := y - y1
			local2X := x - x2
			local2Y := y - y2

			// Prüfe ob beide Pixel nicht-transparent sind (> 0)
			if local1X >= 0 && local1X < SpriteSize && local1Y >= 0 && local1Y < SpriteSize &&
				local2X >= 0 && local2X < SpriteSize && local2Y >= 0 && local2Y < SpriteSize {

				pixel1Index := local1Y*SpriteSize + local1X
				pixel2Index := local2Y*SpriteSize + local2X

				if pixel1Index < len(pixels1) && pixel2Index < len(pixels2) &&
					pixels1[pixel1Index] > 0 && pixels2[pixel2Index] > 0 {
					return true // Kollision gefunden
				}
			}
		}
	}

	return false // Keine Pixel-Kollision
}

// Cache für Sprite-Pixel-Daten
var spritePixelCache = make(map[int][]int)
var spritePixelCacheMutex sync.RWMutex

// Hole Sprite-Pixel-Daten für eine Definition-ID
func getSpritePixelData(definitionID int) []int {
	spritePixelCacheMutex.RLock()
	pixels, exists := spritePixelCache[definitionID]
	spritePixelCacheMutex.RUnlock()

	if exists {
		return pixels
	}

	// Falls nicht im Cache, gib nil zurück (Fallback auf Bounding-Box)
	return nil
}

// Speichere Sprite-Pixel-Daten im Cache (wird beim SPRITE-Define aufgerufen)
func cacheSpritePixelData(definitionID int, pixels []int) {
	spritePixelCacheMutex.Lock()
	spritePixelCache[definitionID] = make([]int, len(pixels))
	copy(spritePixelCache[definitionID], pixels)
	spritePixelCacheMutex.Unlock()
}

// Hilfsfunktionen für min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Aktualisiere Kollisionsinformationen für alle Sprites
func (b *TinyBASIC) updateCollisionDetection() {
	spriteMutex.Lock()
	defer spriteMutex.Unlock()

	now := time.Now() // Alle Sprites bleiben permanent aktiv - keine automatische Entfernung
	// Dies vereinfacht das System erheblich und vermeidet Kollisionsprobleme

	// Sammle alle sichtbaren Sprites
	var visibleSprites []*SpritePosition
	for _, sprite := range activeSprites {
		if sprite.Visible {
			visibleSprites = append(visibleSprites, sprite)
		}
	}

	// Überprüfe Kollisionen für jeden Sprite
	for _, sprite1 := range visibleSprites {
		collisionInfo, exists := spriteCollisions[sprite1.ID]
		if !exists {
			collisionInfo = &SpriteCollisionInfo{
				SpriteID:      sprite1.ID,
				CollidingWith: make([]int, 0),
			}
			spriteCollisions[sprite1.ID] = collisionInfo
		}

		// Reduziere Cache-TTL für responsivere Kollisionserkennung
		if now.Sub(collisionInfo.LastChecked) < 8*time.Millisecond {
			continue
		}

		// Lösche alte Kollisionen
		collisionInfo.CollidingWith = collisionInfo.CollidingWith[:0]

		// Prüfe gegen alle anderen Sprites
		for _, sprite2 := range visibleSprites {
			if sprite1.ID != sprite2.ID && checkSpriteCollision(sprite1, sprite2) {
				collisionInfo.CollidingWith = append(collisionInfo.CollidingWith, sprite2.ID)
			}
		}

		collisionInfo.LastChecked = now
	}
}

// COLLISION Befehl - gibt Anzahl der Kollisionen zurück
func (b *TinyBASIC) cmdCollision(args string) (int, error) {
	b.updateCollisionDetection()

	parts := strings.Fields(strings.TrimSpace(args))
	if len(parts) < 1 {
		return 0, fmt.Errorf("COLLISION: Sprite-ID erforderlich")
	}

	spriteID, err := b.evalExpression(parts[0])
	if err != nil {
		return 0, fmt.Errorf("COLLISION: Ungültige Sprite-ID: %v", err)
	}

	spriteIDInt, err := basicValueToInt(spriteID)
	if err != nil {
		return 0, fmt.Errorf("COLLISION: Sprite-ID muss eine Zahl sein")
	}

	// Wenn zwei Parameter: COLLISION(spriteID, index) - gibt ID des kollidierenden Sprites zurück
	if len(parts) >= 2 {
		indexValue, err := b.evalExpression(parts[1])
		if err != nil {
			return 0, fmt.Errorf("COLLISION: Ungültiger Index: %v", err)
		}

		index, err := basicValueToInt(indexValue)
		if err != nil {
			return 0, fmt.Errorf("COLLISION: Index muss eine Zahl sein")
		}

		spriteMutex.RLock()
		collisionInfo, exists := spriteCollisions[spriteIDInt]
		spriteMutex.RUnlock()

		if !exists || index < 1 || index > len(collisionInfo.CollidingWith) {
			return 0, nil // Kein Sprite an diesem Index
		}

		return collisionInfo.CollidingWith[index-1], nil // 1-basierter Index
	}

	// Ein Parameter: COLLISION(spriteID) - gibt Anzahl der Kollisionen zurück
	spriteMutex.RLock()
	collisionInfo, exists := spriteCollisions[spriteIDInt]
	spriteMutex.RUnlock()

	if !exists {
		return 0, nil
	}

	return len(collisionInfo.CollidingWith), nil
}
