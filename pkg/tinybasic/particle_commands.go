package tinybasic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Particle system constants
const (
	MaxParticleEmitters = 16
	MaxParticlesPerSecond = 255
	MaxGravity = 255
)

// ParticleType represents different particle emission patterns
type ParticleType string

const (
	ParticleTypePoint  ParticleType = "point"
	ParticleTypeStar   ParticleType = "star"
	ParticleTypeCircle ParticleType = "circle"
	ParticleTypeRect   ParticleType = "rect"
)

// ParticleEmitter represents a particle emitter
type ParticleEmitter struct {
	ID           int          `json:"id"`
	Type         ParticleType `json:"type"`
	X            float64      `json:"x"`
	Y            float64      `json:"y"`
	PPS          int          `json:"pps"`          // Particles per second
	Speed        float64      `json:"speed"`        // Initial velocity of particles
	Lifetime     float64      `json:"lifetime"`     // Lifetime in seconds
	Visible      bool         `json:"visible"`      // Whether emitter is active
	Positioned   bool         `json:"positioned"`   // Whether emitter has been positioned
}

// Default values for particle emitters
const (
	DefaultPPS      = 20
	DefaultSpeed    = 50.0
	DefaultLifetime = 3.0
)

// handleParticleCommand processes PARTICLE commands
func (b *TinyBASIC) handleParticleCommand(args string) error {
	// Parse the args string into individual arguments
	argList := strings.Fields(args)
	
	if len(argList) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_MISSING_SUBCOMMAND", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE").
			WithUsageHint("PARTICLE CREATE id,type,[pps],[speed],[lifetime] | PARTICLE MOVE id,x,y | PARTICLE SHOW id | PARTICLE HIDE id | PARTICLE GRAVITY value")
	}

	subcommand := strings.ToUpper(argList[0])

	switch subcommand {
	case "CREATE":
		return b.handleParticleCreate(argList[1:])
	case "MOVE":
		return b.handleParticleMove(argList[1:])
	case "SHOW":
		return b.handleParticleShow(argList[1:])
	case "HIDE":
		return b.handleParticleHide(argList[1:])
	case "GRAVITY":
		return b.handleParticleGravity(argList[1:])
	default:
		return NewBASICError(ErrCategoryCommand, "PARTICLE_UNKNOWN_SUBCOMMAND", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE").
			WithUsageHint("Valid subcommands: CREATE, MOVE, SHOW, HIDE, GRAVITY")
	}
}

// handleParticleCreate creates a new particle emitter
func (b *TinyBASIC) handleParticleCreate(args []string) error {
	if len(args) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_CREATE_MISSING_ARGS", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE CREATE").
			WithUsageHint("PARTICLE CREATE id,type,[pps],[speed],[lifetime]")
	}

	// Parse comma-separated arguments
	argStr := strings.Join(args, " ")
	parts := strings.Split(argStr, ",")

	if len(parts) < 2 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_CREATE_INSUFFICIENT_ARGS", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE CREATE").
			WithUsageHint("PARTICLE CREATE id,type,[pps],[speed],[lifetime]")
	}

	// Parse emitter ID using expression evaluator to handle variables
	idVal, err := b.evalExpression(strings.TrimSpace(parts[0]))
	if err != nil || !idVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE CREATE").
			WithUsageHint("Emitter ID must be a valid expression")
	}
	
	emitterID := int(idVal.NumValue)
	if emitterID < 1 || emitterID > MaxParticleEmitters {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE CREATE").
			WithUsageHint(fmt.Sprintf("Emitter ID must be 1-%d", MaxParticleEmitters))
	}

	// Parse particle type
	particleType := ParticleType(strings.ToLower(strings.TrimSpace(parts[1])))
	if !isValidParticleType(particleType) {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_TYPE", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE CREATE").
			WithUsageHint("Valid types: point, star, circle, rect")
	}

	// Parse optional parameters with defaults
	pps := DefaultPPS
	speed := DefaultSpeed
	lifetime := DefaultLifetime

	if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
		if p, err := strconv.Atoi(strings.TrimSpace(parts[2])); err == nil {
			if p < 1 || p > MaxParticlesPerSecond {
				return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_PPS", b.currentLine == 0, b.currentLine).
					WithCommand("PARTICLE CREATE").
					WithUsageHint(fmt.Sprintf("PPS must be 1-%d", MaxParticlesPerSecond))
			}
			pps = p
		}
	}

	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		if s, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64); err == nil {
			if s < 0 || s > 1000 {
				return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_SPEED", b.currentLine == 0, b.currentLine).
					WithCommand("PARTICLE CREATE").
					WithUsageHint("Speed must be 0-1000")
			}
			speed = s
		}
	}

	if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
		if l, err := strconv.ParseFloat(strings.TrimSpace(parts[4]), 64); err == nil {
			if l < 0.1 || l > 30.0 {
				return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_LIFETIME", b.currentLine == 0, b.currentLine).
					WithCommand("PARTICLE CREATE").
					WithUsageHint("Lifetime must be 0.1-30.0 seconds")
			}
			lifetime = l
		}
	}

	// Create emitter
	emitter := ParticleEmitter{
		ID:         emitterID,
		Type:       particleType,
		X:          0,
		Y:          0,
		PPS:        pps,
		Speed:      speed,
		Lifetime:   lifetime,
		Visible:    false,     // Not visible until positioned
		Positioned: false,     // Not positioned yet
	}

	// Send to frontend
	return b.sendParticleCommand("CREATE_EMITTER", emitter)
}

// handleParticleMove positions and shows an emitter
func (b *TinyBASIC) handleParticleMove(args []string) error {
	if len(args) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_MOVE_MISSING_ARGS", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint("PARTICLE MOVE id,x,y")
	}

	// Parse comma-separated arguments
	argStr := strings.Join(args, " ")
	parts := strings.Split(argStr, ",")

	if len(parts) < 3 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_MOVE_INSUFFICIENT_ARGS", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint("PARTICLE MOVE id,x,y")
	}

	// Parse emitter ID using expression evaluator to handle variables
	idVal, err := b.evalExpression(strings.TrimSpace(parts[0]))
	if err != nil || !idVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint("Emitter ID must be a valid expression")
	}
	
	emitterID := int(idVal.NumValue)
	if emitterID < 1 || emitterID > MaxParticleEmitters {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint(fmt.Sprintf("Emitter ID must be 1-%d", MaxParticleEmitters))
	}

	// Parse position using expression evaluator to handle variables
	xVal, err := b.evalExpression(strings.TrimSpace(parts[1]))
	if err != nil || !xVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_X", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint("X coordinate must be a valid expression")
	}
	x := xVal.NumValue

	yVal, err := b.evalExpression(strings.TrimSpace(parts[2]))
	if err != nil || !yVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_Y", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE MOVE").
			WithUsageHint("Y coordinate must be a valid expression")
	}
	y := yVal.NumValue

	// Create move command
	moveData := map[string]interface{}{
		"id": emitterID,
		"x":  x,
		"y":  y,
	}

	return b.sendParticleCommandWithData("MOVE_EMITTER", moveData)
}

// handleParticleShow shows an emitter
func (b *TinyBASIC) handleParticleShow(args []string) error {
	if len(args) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_SHOW_MISSING_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE SHOW").
			WithUsageHint("PARTICLE SHOW id")
	}

	// Parse emitter ID using expression evaluator to handle variables
	idVal, err := b.evalExpression(strings.TrimSpace(args[0]))
	if err != nil || !idVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE SHOW").
			WithUsageHint("Emitter ID must be a valid expression")
	}
	
	emitterID := int(idVal.NumValue)
	if emitterID < 1 || emitterID > MaxParticleEmitters {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE SHOW").
			WithUsageHint(fmt.Sprintf("Emitter ID must be 1-%d", MaxParticleEmitters))
	}

	showData := map[string]interface{}{
		"id": emitterID,
	}

	return b.sendParticleCommandWithData("SHOW_EMITTER", showData)
}

// handleParticleHide hides an emitter
func (b *TinyBASIC) handleParticleHide(args []string) error {
	if len(args) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_HIDE_MISSING_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE HIDE").
			WithUsageHint("PARTICLE HIDE id")
	}

	// Parse emitter ID using expression evaluator to handle variables
	idVal, err := b.evalExpression(strings.TrimSpace(args[0]))
	if err != nil || !idVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE HIDE").
			WithUsageHint("Emitter ID must be a valid expression")
	}
	
	emitterID := int(idVal.NumValue)
	if emitterID < 1 || emitterID > MaxParticleEmitters {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_ID", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE HIDE").
			WithUsageHint(fmt.Sprintf("Emitter ID must be 1-%d", MaxParticleEmitters))
	}

	hideData := map[string]interface{}{
		"id": emitterID,
	}

	return b.sendParticleCommandWithData("HIDE_EMITTER", hideData)
}

// handleParticleGravity sets global gravity
func (b *TinyBASIC) handleParticleGravity(args []string) error {
	if len(args) == 0 {
		return NewBASICError(ErrCategoryCommand, "PARTICLE_GRAVITY_MISSING_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE GRAVITY").
			WithUsageHint("PARTICLE GRAVITY value (0-255)")
	}

	// Parse gravity using expression evaluator to handle variables
	gravityVal, err := b.evalExpression(strings.TrimSpace(args[0]))
	if err != nil || !gravityVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_GRAVITY", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE GRAVITY").
			WithUsageHint("Gravity must be a valid expression (0-255)")
	}
	
	gravity := int(gravityVal.NumValue)
	if gravity < 0 || gravity > MaxGravity {
		return NewBASICError(ErrCategoryEvaluation, "PARTICLE_INVALID_GRAVITY", b.currentLine == 0, b.currentLine).
			WithCommand("PARTICLE GRAVITY").
			WithUsageHint(fmt.Sprintf("Gravity must be 0-%d", MaxGravity))
	}

	gravityData := map[string]interface{}{
		"gravity": gravity,
	}

	return b.sendParticleCommandWithData("SET_GRAVITY", gravityData)
}

// isValidParticleType checks if a particle type is valid
func isValidParticleType(pType ParticleType) bool {
	return pType == ParticleTypePoint || 
		   pType == ParticleTypeStar || 
		   pType == ParticleTypeCircle || 
		   pType == ParticleTypeRect
}

// sendParticleCommand sends a particle command to the frontend
func (b *TinyBASIC) sendParticleCommand(command string, emitter ParticleEmitter) error {
	msg := shared.Message{
		Type:    shared.MessageTypeParticle,
		Command: command,
		ID:      emitter.ID,
		CustomData: map[string]interface{}{
			"type":       string(emitter.Type),
			"x":          emitter.X,
			"y":          emitter.Y,
			"pps":        emitter.PPS,
			"speed":      emitter.Speed,
			"lifetime":   emitter.Lifetime,
			"visible":    emitter.Visible,
			"positioned": emitter.Positioned,
		},
	}

	b.sendMessageObject(msg)
	return nil
}

// sendParticleCommandWithData sends a particle command with custom data
func (b *TinyBASIC) sendParticleCommandWithData(command string, data map[string]interface{}) error {
	msg := shared.Message{
		Type:       shared.MessageTypeParticle,
		Command:    command,
		CustomData: data,
	}

	b.sendMessageObject(msg)
	return nil
}