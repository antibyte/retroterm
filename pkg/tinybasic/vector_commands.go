package tinybasic

import (
	"fmt"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Vector-Graphics commands for TinyBASIC

// Constants for vector commands
const (
	MaxVectorID       = 255 // Maximum number of vector objects
	DefaultBrightness = 15  // Default brightness for vector objects
	MaxBrightness     = 15  // Maximum brightness for vector objects
	MaxGridSize       = 256 // Maximum grid size for VECFLOOR
)

// VectorShape represents the supported 3D shapes
type VectorShape string

// Supported 3D shapes
const (
	ShapeCube       VectorShape = "cube"
	ShapePyramid    VectorShape = "pyramid"
	ShapeSphere     VectorShape = "sphere"
	ShapeCylinder   VectorShape = "cylinder"
	ShapeCone       VectorShape = "cone"
	ShapeFloor      VectorShape = "floor"
)

// Helper function to send vector commands
func (b *TinyBASIC) sendVectorCommand(cmd shared.Message) { // Parameter auf shared.Message geändert
	b.sendMessageObject(cmd) // Send message to the frontend
}

// cmdVector implements the VECTOR command:
// VECTOR id, shape, x, y, z, rotX, rotY, rotZ, scale, [brightness]
// Creates or updates a 3D vector object
func (b *TinyBASIC) cmdVector(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 9 || len(params) > 10 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("VECTOR id, shape, x, y, z, rotX, rotY, rotZ, scale, [brightness]")
	}

	// Check id
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Error in vector ID parameter")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Vector ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint(fmt.Sprintf("Vector ID must be between 0 and %d", MaxVectorID))
	}
	// Check shape
	shapeExpr := strings.TrimSpace(params[1])
	shapeVal, err := b.evalExpression(shapeExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Error in shape parameter")
	}
	if shapeVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Shape must be a string: 'cube', 'pyramid', or 'sphere'")
	}
	shapeName := strings.ToLower(strings.Trim(shapeVal.StrValue, "\"' \t"))
	// Validate shape later or in the frontend

	// Check position X,Y,Z
	// pos := Position3D{} // Old structure
	posX, err := b.evalNumericParam(params[2], "x position")
	if err != nil {
		return err
	}
	posY, err := b.evalNumericParam(params[3], "y position")
	if err != nil {
		return err
	}
	posZ, err := b.evalNumericParam(params[4], "z position")
	if err != nil {
		return err
	}
	// pos.X = posX // Old structure
	// pos.Y = posY // Old structure
	// pos.Z = posZ // Old structure

	// Rotation X,Y,Z calculate in degrees
	// rot := Rotation3D{} // Old structure
	rotX, err := b.evalNumericParam(params[5], "x rotation")
	if err != nil {
		return err
	}
	rotY, err := b.evalNumericParam(params[6], "y rotation")
	if err != nil {
		return err
	}
	rotZ, err := b.evalNumericParam(params[7], "z rotation")
	if err != nil {
		return err
	}
	// transform rotation from degrees to radians
	// rot.X = degToRad(rotX) // Old structure
	// rot.Y = degToRad(rotY) // Old structure
	// rot.Z = degToRad(rotZ) // Old structure
	// Check scale
	scaleExpr := strings.TrimSpace(params[8])
	scaleVal, err := b.evalExpression(scaleExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Error in scale parameter")
	}
	if !scaleVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR").
			WithUsageHint("Scale must be numeric")
	}
	scale := scaleVal.NumValue
	// Optional brightness check
	brightness := DefaultBrightness
	if len(params) >= 10 {
		brightnessExpr := strings.TrimSpace(params[9])
		brightnessVal, err := b.evalExpression(brightnessExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECTOR").
				WithUsageHint("Brightness parameter must be a valid numeric expression")
		}
		if !brightnessVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECTOR").
				WithUsageHint("Brightness must be numeric between 0-15")
		}
		brightness = int(brightnessVal.NumValue)
		if brightness < 0 || brightness > MaxBrightness {
			brightness = MaxBrightness // Limit to valid range
		}
	}

	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Shape:   shapeName, // string
		Position: map[string]float64{
			"x": posX,
			"y": posY,
			"z": posZ,
		},
		VecRotation: map[string]float64{
			"x": degToRad(rotX),
			"y": degToRad(rotY),
			"z": degToRad(rotZ),
		},
		Scale:      scale, // scale is float64, which accepts interface{}
		Visible:    boolPtr(true),
		Brightness: brightness,
	})

	return nil
}

// cmdVector3DScale implements the VECTOR.SCALE command:
// VECTOR.SCALE id, scaleX, scaleY, scaleZ, [brightness]
// Updates the scaling of a 3D vector object with separate X,Y,Z values
func (b *TinyBASIC) cmdVector3DScale(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 4 || len(params) > 5 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SCALE").
			WithUsageHint("VECTOR.SCALE requires 4 or 5 parameters: VECTOR.SCALE id, scaleX, scaleY, scaleZ, [brightness]")
	}

	// Check ID
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SCALE").
			WithUsageHint("Vector ID parameter must be a valid numeric expression")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SCALE").
			WithUsageHint("Vector ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SCALE").
			WithUsageHint(fmt.Sprintf("Vector ID must be between 0 and %d", MaxVectorID))
	}

	// Check scaling X,Y,Z
	scaleX, err := b.evalNumericParam(params[1], "x scale")
	if err != nil {
		return err
	}
	scaleY, err := b.evalNumericParam(params[2], "y scale")
	if err != nil {
		return err
	}
	scaleZ, err := b.evalNumericParam(params[3], "z scale")
	if err != nil {
		return err
	}
	// Optional brightness check
	brightness := DefaultBrightness
	if len(params) >= 5 {
		brightnessExpr := strings.TrimSpace(params[4])
		brightnessVal, err := b.evalExpression(brightnessExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECTOR.SCALE").
				WithUsageHint("Brightness parameter must be a valid numeric expression")
		}
		if !brightnessVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECTOR.SCALE").
				WithUsageHint("Brightness must be numeric between 0-15")
		}
		brightness = int(brightnessVal.NumValue)
		if brightness < 0 || brightness > MaxBrightness {
			brightness = MaxBrightness // Limit to valid range
		}
	}

	// Create a 3D scaling map for shared.Message
	scaleMap := map[string]float64{
		"x": scaleX,
		"y": scaleY,
		"z": scaleZ,
	}

	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		// Shape, Position, VecRotation are not changed, so do not send or use default values
		Scale:      scaleMap, // Corrected: use scaleMap
		Visible:    boolPtr(true),
		Brightness: brightness,
	})

	return nil
}

// cmdVectorHide implements the VECTOR.HIDE command: VECTOR.HIDE id
// Hides a 3D vector object
func (b *TinyBASIC) cmdVectorHide(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 1 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.HIDE").
			WithUsageHint("VECTOR.HIDE requires exactly 1 parameter: VECTOR.HIDE id")
	}

	// Check ID
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.HIDE").
			WithUsageHint("Vector ID parameter must be a valid numeric expression")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.HIDE").WithUsageHint("Vector ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.HIDE").
			WithUsageHint(fmt.Sprintf("Vector ID must be between 0 and %d", MaxVectorID))
	}

	// Create and send the VECTOR command
	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Visible: boolPtr(false), // Hide object
		// Other fields are not changed
	})

	return nil
}

// cmdVectorShow implements the VECTOR.SHOW command: VECTOR.SHOW id
// Shows a 3D vector object
func (b *TinyBASIC) cmdVectorShow(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 1 {
		return NewBASICError(ErrCategorySyntax, "SYNTAX_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SHOW").
			WithUsageHint("VECTOR.SHOW requires exactly 1 parameter: VECTOR.SHOW id")
	}

	// Check ID
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SHOW").
			WithUsageHint("Vector ID parameter must be a valid numeric expression")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_TYPE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SHOW").
			WithUsageHint("Vector ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "VECTOR_ID_RANGE_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("VECTOR.SHOW").
			WithUsageHint(fmt.Sprintf("Vector ID must be between 0 and %d", MaxVectorID))
	}

	// Create and send the VECTOR command
	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Visible: boolPtr(true), // Show object
		// Other fields are not changed
	})

	return nil
}

// Helper function to create a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// Helper function to evaluate a numeric parameter
func (b *TinyBASIC) evalNumericParam(paramExpr string, paramName string) (float64, error) {
	expr := strings.TrimSpace(paramExpr)
	val, err := b.evalExpression(expr)
	if err != nil {
		return 0, NewBASICError(ErrCategoryEvaluation, "NUMERIC_PARAM_ERROR", b.currentLine == 0, b.currentLine).
			WithUsageHint(fmt.Sprintf("Error in %s parameter: %v", paramName, err))
	}
	if !val.IsNumeric {
		return 0, NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithUsageHint(fmt.Sprintf("%s must be numeric", paramName))
	}
	return val.NumValue, nil
}

// Helper function to convert degrees to radians
func degToRad(deg float64) float64 {
	return deg * 0.017453292519943295 // deg * (π / 180)
}

// cmdPyramid implements the PYRAMID command:
// PYRAMID id, base, x, y, z, rotX, rotY, rotZ, scale, height, [brightness]
// Creates a pyramid with different base shapes: "square", "triangle", "pentagon", "hexagon"
func (b *TinyBASIC) cmdPyramid(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 10 || len(params) > 11 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("PYRAMID id, base, x, y, z, rotX, rotY, rotZ, scale, height, [brightness]")
	}

	// Check id
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("Error in pyramid ID parameter")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("Pyramid ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint(fmt.Sprintf("Pyramid ID must be between 0 and %d", MaxVectorID))
	}

	// Check base shape
	baseExpr := strings.TrimSpace(params[1])
	baseVal, err := b.evalExpression(baseExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("Error in base shape parameter")
	}
	if baseVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("Base shape must be a string: 'square', 'triangle', 'pentagon', 'hexagon'")
	}
	baseShape := strings.ToLower(strings.Trim(baseVal.StrValue, "\"' \t"))

	// Validate base shape
	validBases := map[string]bool{
		"square":   true,
		"triangle": true,
		"pentagon": true,
		"hexagon":  true,
	}
	if !validBases[baseShape] {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("PYRAMID").
			WithUsageHint("Base shape must be 'square', 'triangle', 'pentagon', or 'hexagon'")
	}

	// Position X,Y,Z
	posX, err := b.evalNumericParam(params[2], "x position")
	if err != nil {
		return err
	}
	posY, err := b.evalNumericParam(params[3], "y position")
	if err != nil {
		return err
	}
	posZ, err := b.evalNumericParam(params[4], "z position")
	if err != nil {
		return err
	}

	// Rotation X,Y,Z
	rotX, err := b.evalNumericParam(params[5], "x rotation")
	if err != nil {
		return err
	}
	rotY, err := b.evalNumericParam(params[6], "y rotation")
	if err != nil {
		return err
	}
	rotZ, err := b.evalNumericParam(params[7], "z rotation")
	if err != nil {
		return err
	}

	// Scale
	scale, err := b.evalNumericParam(params[8], "scale")
	if err != nil {
		return err
	}

	// Height
	height, err := b.evalNumericParam(params[9], "height")
	if err != nil {
		return err
	}

	// Optional brightness check
	brightness := DefaultBrightness
	if len(params) >= 11 {
		brightnessExpr := strings.TrimSpace(params[10])
		brightnessVal, err := b.evalExpression(brightnessExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("PYRAMID").
				WithUsageHint("Brightness parameter must be a valid numeric expression")
		}
		if !brightnessVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("PYRAMID").
				WithUsageHint("Brightness must be numeric between 0-15")
		}
		brightness = int(brightnessVal.NumValue)
		if brightness < 0 || brightness > MaxBrightness {
			brightness = MaxBrightness // Limit to valid range
		}
	}

	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Shape:   "pyramid", // string
		Position: map[string]float64{
			"x": posX,
			"y": posY,
			"z": posZ,
		},
		VecRotation: map[string]float64{
			"x": degToRad(rotX),
			"y": degToRad(rotY),
			"z": degToRad(rotZ),
		},
		Scale:      scale,
		Visible:    boolPtr(true),
		Brightness: brightness,
		// Store base shape and height as custom parameters
		CustomData: map[string]interface{}{
			"baseShape": baseShape,
			"height":    height,
		},
	})

	return nil
}

// cmdCylinder implements the CYLINDER command:
// CYLINDER id, x, y, z, rotX, rotY, rotZ, radius, height, [brightness]
// Creates a cylinder with 8 connecting lines between top and bottom circles
func (b *TinyBASIC) cmdCylinder(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 9 || len(params) > 10 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("CYLINDER").
			WithUsageHint("CYLINDER id, x, y, z, rotX, rotY, rotZ, radius, height, [brightness]")
	}

	// Check id
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("CYLINDER").
			WithUsageHint("Error in cylinder ID parameter")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("CYLINDER").
			WithUsageHint("Cylinder ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("CYLINDER").
			WithUsageHint(fmt.Sprintf("Cylinder ID must be between 0 and %d", MaxVectorID))
	}

	// Position X,Y,Z
	posX, err := b.evalNumericParam(params[1], "x position")
	if err != nil {
		return err
	}
	posY, err := b.evalNumericParam(params[2], "y position")
	if err != nil {
		return err
	}
	posZ, err := b.evalNumericParam(params[3], "z position")
	if err != nil {
		return err
	}

	// Rotation X,Y,Z
	rotX, err := b.evalNumericParam(params[4], "x rotation")
	if err != nil {
		return err
	}
	rotY, err := b.evalNumericParam(params[5], "y rotation")
	if err != nil {
		return err
	}
	rotZ, err := b.evalNumericParam(params[6], "z rotation")
	if err != nil {
		return err
	}

	// Radius
	radius, err := b.evalNumericParam(params[7], "radius")
	if err != nil {
		return err
	}

	// Height
	height, err := b.evalNumericParam(params[8], "height")
	if err != nil {
		return err
	}

	// Optional brightness check
	brightness := DefaultBrightness
	if len(params) >= 10 {
		brightnessExpr := strings.TrimSpace(params[9])
		brightnessVal, err := b.evalExpression(brightnessExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("CYLINDER").
				WithUsageHint("Brightness parameter must be a valid numeric expression")
		}
		if !brightnessVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("CYLINDER").
				WithUsageHint("Brightness must be numeric between 0-15")
		}
		brightness = int(brightnessVal.NumValue)
		if brightness < 0 || brightness > MaxBrightness {
			brightness = MaxBrightness // Limit to valid range
		}
	}

	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Shape:   "cylinder", // string
		Position: map[string]float64{
			"x": posX,
			"y": posY,
			"z": posZ,
		},
		VecRotation: map[string]float64{
			"x": degToRad(rotX),
			"y": degToRad(rotY),
			"z": degToRad(rotZ),
		},
		Scale:      1.0, // Not used for cylinder, use radius/height instead
		Visible:    boolPtr(true),
		Brightness: brightness,
		// Store radius and height as custom parameters
		CustomData: map[string]interface{}{
			"radius":     radius,
			"height":     height,
			"lineCount":  8, // 8 connecting lines
		},
	})

	return nil
}

// cmdVecFloor implements the VECFLOOR command:
// VECFLOOR id, x, y, z, rotX, rotY, rotZ, gridWidth, gridDepth, spacing, [brightness]
// Creates a grid floor with adjustable size and positioning
func (b *TinyBASIC) cmdVecFloor(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 10 || len(params) > 11 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint("VECFLOOR id, x, y, z, rotX, rotY, rotZ, gridWidth, gridDepth, spacing, [brightness]")
	}

	// Check id
	idExpr := strings.TrimSpace(params[0])
	idVal, err := b.evalExpression(idExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint("Error in floor ID parameter")
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint("Floor ID must be numeric")
	}
	if id < 0 || id > MaxVectorID {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint(fmt.Sprintf("Floor ID must be between 0 and %d", MaxVectorID))
	}

	// Position X,Y,Z
	posX, err := b.evalNumericParam(params[1], "x position")
	if err != nil {
		return err
	}
	posY, err := b.evalNumericParam(params[2], "y position")
	if err != nil {
		return err
	}
	posZ, err := b.evalNumericParam(params[3], "z position")
	if err != nil {
		return err
	}

	// Rotation X,Y,Z
	rotX, err := b.evalNumericParam(params[4], "x rotation")
	if err != nil {
		return err
	}
	rotY, err := b.evalNumericParam(params[5], "y rotation")
	if err != nil {
		return err
	}
	rotZ, err := b.evalNumericParam(params[6], "z rotation")
	if err != nil {
		return err
	}

	// Grid dimensions
	gridWidth, err := b.evalNumericParam(params[7], "grid width")
	if err != nil {
		return err
	}
	if gridWidth < 1 || gridWidth > MaxGridSize {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint(fmt.Sprintf("Grid width must be between 1 and %d", MaxGridSize))
	}

	gridDepth, err := b.evalNumericParam(params[8], "grid depth")
	if err != nil {
		return err
	}
	if gridDepth < 1 || gridDepth > MaxGridSize {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECFLOOR").
			WithUsageHint(fmt.Sprintf("Grid depth must be between 1 and %d", MaxGridSize))
	}

	// Spacing between grid points
	spacing, err := b.evalNumericParam(params[9], "spacing")
	if err != nil {
		return err
	}

	// Optional brightness check
	brightness := DefaultBrightness
	if len(params) >= 11 {
		brightnessExpr := strings.TrimSpace(params[10])
		brightnessVal, err := b.evalExpression(brightnessExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_PARAM_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECFLOOR").
				WithUsageHint("Brightness parameter must be a valid numeric expression")
		}
		if !brightnessVal.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "VECTOR_BRIGHTNESS_TYPE_ERROR", b.currentLine == 0, b.currentLine).
				WithCommand("VECFLOOR").
				WithUsageHint("Brightness must be numeric between 0-15")
		}
		brightness = int(brightnessVal.NumValue)
		if brightness < 0 || brightness > MaxBrightness {
			brightness = MaxBrightness // Limit to valid range
		}
	}

	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_VECTOR",
		ID:      id,
		Shape:   "floor", // string
		Position: map[string]float64{
			"x": posX,
			"y": posY,
			"z": posZ,
		},
		VecRotation: map[string]float64{
			"x": degToRad(rotX),
			"y": degToRad(rotY),
			"z": degToRad(rotZ),
		},
		Scale:      1.0, // Not used for floor, use grid dimensions instead
		Visible:    boolPtr(true),
		Brightness: brightness,
		// Store grid parameters
		CustomData: map[string]interface{}{
			"gridWidth": int(gridWidth),
			"gridDepth": int(gridDepth),
			"spacing":   spacing,
		},
	})

	return nil
}

// cmdVecNode implements the VECNODE command:
// VECNODE gridX, gridZ, height
// Modifies the height of a specific grid point in the current floor
func (b *TinyBASIC) cmdVecNode(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 3 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("VECNODE").
			WithUsageHint("VECNODE gridX, gridZ, height")
	}

	// Grid coordinates
	gridX, err := b.evalNumericParam(params[0], "grid X")
	if err != nil {
		return err
	}
	if gridX < 0 || gridX >= MaxGridSize {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECNODE").
			WithUsageHint(fmt.Sprintf("Grid X must be between 0 and %d", MaxGridSize-1))
	}

	gridZ, err := b.evalNumericParam(params[1], "grid Z")
	if err != nil {
		return err
	}
	if gridZ < 0 || gridZ >= MaxGridSize {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("VECNODE").
			WithUsageHint(fmt.Sprintf("Grid Z must be between 0 and %d", MaxGridSize-1))
	}

	// Height modification
	height, err := b.evalNumericParam(params[2], "height")
	if err != nil {
		return err
	}

	// Send node modification command
	b.sendVectorCommand(shared.Message{
		Type:    shared.MessageTypeVector,
		Command: "UPDATE_NODE",
		CustomData: map[string]interface{}{
			"gridX":  int(gridX),
			"gridZ":  int(gridZ),
			"height": height,
		},
	})

	return nil
}

// Helper function to handle VECTOR.* commands
func (b *TinyBASIC) handleVectorCommands(cmd string, args string) error {
	switch {
	case cmd == "VECTOR":
		return b.cmdVector(args)
	case cmd == "VECTOR.SCALE":
		return b.cmdVector3DScale(args)
	case cmd == "VECTOR.HIDE":
		return b.cmdVectorHide(args)
	case cmd == "VECTOR.SHOW":
		return b.cmdVectorShow(args)
	case cmd == "PYRAMID":
		return b.cmdPyramid(args)
	case cmd == "CYLINDER":
		return b.cmdCylinder(args)
	default:
		return NewBASICError(ErrCategorySyntax, "UNKNOWN_COMMAND", b.currentLine == 0, b.currentLine).
			WithCommand(cmd).
			WithUsageHint("Unknown vector command. Valid commands: VECTOR, VECTOR.SCALE, VECTOR.HIDE, VECTOR.SHOW, PYRAMID, CYLINDER")
	}
}
