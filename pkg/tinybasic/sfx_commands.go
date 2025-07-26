package tinybasic

import (
	"fmt"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// SFX effect constants
const (
	MaxSFXVariants = 8 // Maximum variants per effect type
)

// SFXEffectType represents different sound effects
type SFXEffectType string

const (
	SFXExplosion SFXEffectType = "explosion"
	SFXShoot     SFXEffectType = "shoot"
	SFXLaser     SFXEffectType = "laser"
	SFXCoin      SFXEffectType = "coin"
	SFXPowerup   SFXEffectType = "powerup"
	SFXHit       SFXEffectType = "hit"
	SFXJump      SFXEffectType = "jump"
	SFXSynth     SFXEffectType = "synth"
	SFXSpecial   SFXEffectType = "special"
	SFXRandom    SFXEffectType = "random"
)

// handlePlaySFXCommand processes PLAYSFX commands
func (b *TinyBASIC) handlePlaySFXCommand(args string) error {
	if strings.TrimSpace(args) == "" {
		return NewBASICError(ErrCategoryCommand, "PLAYSFX_MISSING_EFFECT", b.currentLine == 0, b.currentLine).
			WithCommand("PLAYSFX").
			WithUsageHint("PLAYSFX effect[,variant] - effect: explosion, shoot, laser, coin, powerup, hit, jump, synth, special, random")
	}

	// Parse arguments
	params := strings.Split(strings.TrimSpace(args), ",")
	if len(params) < 1 || len(params) > 2 {
		return NewBASICError(ErrCategoryCommand, "PLAYSFX_INVALID_ARGS", b.currentLine == 0, b.currentLine).
			WithCommand("PLAYSFX").
			WithUsageHint("PLAYSFX effect[,variant]")
	}

	// Parse effect type
	effectExpr := strings.TrimSpace(params[0])
	effectVal, err := b.evalExpression(effectExpr)
	if err != nil || effectVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "PLAYSFX_INVALID_EFFECT", b.currentLine == 0, b.currentLine).
			WithCommand("PLAYSFX").
			WithUsageHint("Effect must be a string: explosion, shoot, laser, coin, powerup, hit, jump, synth, special, random")
	}

	effect := strings.ToLower(strings.Trim(effectVal.StrValue, "\""))
	if !isValidSFXEffect(SFXEffectType(effect)) {
		return NewBASICError(ErrCategoryEvaluation, "PLAYSFX_UNKNOWN_EFFECT", b.currentLine == 0, b.currentLine).
			WithCommand("PLAYSFX").
			WithUsageHint("Valid effects: explosion, shoot, laser, coin, powerup, hit, jump, synth, special, random")
	}

	// Parse variant (optional)
	var variant int = -1 // -1 means use library default
	if len(params) == 2 {
		variantExpr := strings.TrimSpace(params[1])
		variantVal, err := b.evalExpression(variantExpr)
		if err != nil || !variantVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "PLAYSFX_INVALID_VARIANT", b.currentLine == 0, b.currentLine).
				WithCommand("PLAYSFX").
				WithUsageHint("Variant must be a number (1-8)")
		}
		
		variant = int(variantVal.NumValue)
		if variant < 1 || variant > MaxSFXVariants {
			return NewBASICError(ErrCategoryEvaluation, "PLAYSFX_VARIANT_OUT_OF_RANGE", b.currentLine == 0, b.currentLine).
				WithCommand("PLAYSFX").
				WithUsageHint(fmt.Sprintf("Variant must be between 1 and %d", MaxSFXVariants))
		}
	}

	// Send SFX command to frontend
	sfxData := map[string]interface{}{
		"effect":  effect,
		"variant": variant,
	}

	return b.sendSFXCommandWithData("PLAY_SFX", sfxData)
}

// isValidSFXEffect checks if the effect type is valid
func isValidSFXEffect(effect SFXEffectType) bool {
	switch effect {
	case SFXExplosion, SFXShoot, SFXLaser, SFXCoin, SFXPowerup, SFXHit, SFXJump, SFXSynth, SFXSpecial, SFXRandom:
		return true
	default:
		return false
	}
}

// sendSFXCommandWithData sends SFX command with data to the frontend
func (b *TinyBASIC) sendSFXCommandWithData(command string, data map[string]interface{}) error {
	message := shared.Message{
		Type:    shared.MessageTypeSFX,
		Command: command,
		Params:  data,
	}

	if !b.sendMessageObject(message) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).
			WithCommand("PLAYSFX")
	}

	return nil
}