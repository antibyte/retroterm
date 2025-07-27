package tinybasic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// handlePhysicsCommand processes PHYSICS commands
func (b *TinyBASIC) handlePhysicsCommand(args string) error {
	if args == "" {
		return fmt.Errorf("PHYSICS command requires arguments")
	}

	// Split into subcommand and arguments
	parts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	if len(parts) == 0 {
		return fmt.Errorf("PHYSICS command requires subcommand")
	}

	subcommand := strings.ToUpper(parts[0])
	var subArgsStr string
	if len(parts) > 1 {
		subArgsStr = strings.TrimSpace(parts[1])
	}

	// Parse comma-separated arguments for the subcommand
	var subArgs []string
	if subArgsStr != "" {
		subArgs = strings.Split(subArgsStr, ",")
		// Trim whitespace from each argument
		for i := range subArgs {
			subArgs[i] = strings.TrimSpace(subArgs[i])
		}
	}

	switch subcommand {
	case "WORLD":
		return b.handlePhysicsWorld(subArgs)
	case "SCALE":
		return b.handlePhysicsScale(subArgs)
	case "FLOOR":
		return b.handlePhysicsFloor(subArgs)
	case "WALL":
		return b.handlePhysicsWall(subArgs)
	case "LINE":
		return b.handlePhysicsLine(subArgs)
	case "RECT":
		return b.handlePhysicsRect(subArgs)
	case "CIRCLE":
		return b.handlePhysicsCircle(subArgs)
	case "STEP":
		return b.handlePhysicsStep()
	case "AUTO":
		return b.handlePhysicsAuto(subArgs)
	case "VELOCITY":
		return b.handlePhysicsVelocity(subArgs)
	case "FORCE":
		return b.handlePhysicsForce(subArgs)
	case "FRICTION":
		return b.handlePhysicsFriction(subArgs)
	case "BOUNCE":
		return b.handlePhysicsBounce(subArgs)
	case "DENSITY":
		return b.handlePhysicsDensity(subArgs)
	case "GROUP":
		return b.handlePhysicsGroup(subArgs)
	case "COLLIDE":
		return b.handlePhysicsCollide(subArgs)
	case "COLLISION":
		return b.handlePhysicsCollision(subArgs)
	case "LINK":
		return b.handlePhysicsLink(subArgs)
	default:
		return fmt.Errorf("unknown PHYSICS subcommand: %s", subcommand)
	}
}

// handlePhysicsWorld sets up the physics world with gravity
func (b *TinyBASIC) handlePhysicsWorld(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS WORLD requires gravity X and Y values")
	}

	gravityXVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid gravity X: %v", err)
	}
	gravityX, err := basicValueToInt(gravityXVal)
	if err != nil {
		return fmt.Errorf("gravity X must be numeric: %v", err)
	}

	gravityYVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid gravity Y: %v", err)
	}
	gravityY, err := basicValueToInt(gravityYVal)
	if err != nil {
		return fmt.Errorf("gravity Y must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("WORLD", map[string]interface{}{
		"gravityX": float64(gravityX),
		"gravityY": float64(gravityY),
	})
}

// handlePhysicsScale sets the pixel to meter scale
func (b *TinyBASIC) handlePhysicsScale(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("PHYSICS SCALE requires scale value")
	}

	scaleVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid scale: %v", err)
	}
	scale, err := basicValueToInt(scaleVal)
	if err != nil {
		return fmt.Errorf("scale must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("SCALE", map[string]interface{}{
		"scale": float64(scale),
	})
}

// handlePhysicsFloor creates a floor line
func (b *TinyBASIC) handlePhysicsFloor(args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("PHYSICS FLOOR requires x1, y1, x2, y2")
	}

	x1Val, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid x1: %v", err)
	}
	x1, err := basicValueToInt(x1Val)
	if err != nil {
		return fmt.Errorf("x1 must be numeric: %v", err)
	}

	y1Val, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid y1: %v", err)
	}
	y1, err := basicValueToInt(y1Val)
	if err != nil {
		return fmt.Errorf("y1 must be numeric: %v", err)
	}

	x2Val, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid x2: %v", err)
	}
	x2, err := basicValueToInt(x2Val)
	if err != nil {
		return fmt.Errorf("x2 must be numeric: %v", err)
	}

	y2Val, err := b.evalExpression(args[3])
	if err != nil {
		return fmt.Errorf("invalid y2: %v", err)
	}
	y2, err := basicValueToInt(y2Val)
	if err != nil {
		return fmt.Errorf("y2 must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("FLOOR", map[string]interface{}{
		"x1": float64(x1),
		"y1": float64(y1),
		"x2": float64(x2),
		"y2": float64(y2),
	})
}

// handlePhysicsWall creates a wall line  
func (b *TinyBASIC) handlePhysicsWall(args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("PHYSICS WALL requires x1, y1, x2, y2")
	}

	x1Val, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid x1: %v", err)
	}
	x1, err := basicValueToInt(x1Val)
	if err != nil {
		return fmt.Errorf("x1 must be numeric: %v", err)
	}

	y1Val, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid y1: %v", err)
	}
	y1, err := basicValueToInt(y1Val)
	if err != nil {
		return fmt.Errorf("y1 must be numeric: %v", err)
	}

	x2Val, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid x2: %v", err)
	}
	x2, err := basicValueToInt(x2Val)
	if err != nil {
		return fmt.Errorf("x2 must be numeric: %v", err)
	}

	y2Val, err := b.evalExpression(args[3])
	if err != nil {
		return fmt.Errorf("invalid y2: %v", err)
	}
	y2, err := basicValueToInt(y2Val)
	if err != nil {
		return fmt.Errorf("y2 must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("WALL", map[string]interface{}{
		"x1": float64(x1),
		"y1": float64(y1),
		"x2": float64(x2),
		"y2": float64(y2),
	})
}

// handlePhysicsLine creates a generic line
func (b *TinyBASIC) handlePhysicsLine(args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("PHYSICS LINE requires x1, y1, x2, y2")
	}

	x1Val, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid x1: %v", err)
	}
	x1, err := basicValueToInt(x1Val)
	if err != nil {
		return fmt.Errorf("x1 must be numeric: %v", err)
	}

	y1Val, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid y1: %v", err)
	}
	y1, err := basicValueToInt(y1Val)
	if err != nil {
		return fmt.Errorf("y1 must be numeric: %v", err)
	}

	x2Val, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid x2: %v", err)
	}
	x2, err := basicValueToInt(x2Val)
	if err != nil {
		return fmt.Errorf("x2 must be numeric: %v", err)
	}

	y2Val, err := b.evalExpression(args[3])
	if err != nil {
		return fmt.Errorf("invalid y2: %v", err)
	}
	y2, err := basicValueToInt(y2Val)
	if err != nil {
		return fmt.Errorf("y2 must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("LINE", map[string]interface{}{
		"x1": float64(x1),
		"y1": float64(y1),
		"x2": float64(x2),
		"y2": float64(y2),
	})
}

// handlePhysicsRect creates a rectangle collider
func (b *TinyBASIC) handlePhysicsRect(args []string) error {
	if len(args) != 4 {
		return fmt.Errorf("PHYSICS RECT requires x, y, width, height")
	}

	xVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid x: %v", err)
	}
	x, err := basicValueToInt(xVal)
	if err != nil {
		return fmt.Errorf("x must be numeric: %v", err)
	}

	yVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid y: %v", err)
	}
	y, err := basicValueToInt(yVal)
	if err != nil {
		return fmt.Errorf("y must be numeric: %v", err)
	}

	widthVal, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid width: %v", err)
	}
	width, err := basicValueToInt(widthVal)
	if err != nil {
		return fmt.Errorf("width must be numeric: %v", err)
	}

	heightVal, err := b.evalExpression(args[3])
	if err != nil {
		return fmt.Errorf("invalid height: %v", err)
	}
	height, err := basicValueToInt(heightVal)
	if err != nil {
		return fmt.Errorf("height must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("RECT", map[string]interface{}{
		"x":      float64(x),
		"y":      float64(y),
		"width":  float64(width),
		"height": float64(height),
	})
}

// handlePhysicsCircle creates a circle collider
func (b *TinyBASIC) handlePhysicsCircle(args []string) error {
	if len(args) < 3 || len(args) > 4 {
		return fmt.Errorf("PHYSICS CIRCLE requires x, y, radius, [id]")
	}

	xVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid x: %v", err)
	}
	x, err := basicValueToInt(xVal)
	if err != nil {
		return fmt.Errorf("x must be numeric: %v", err)
	}

	yVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid y: %v", err)
	}
	y, err := basicValueToInt(yVal)
	if err != nil {
		return fmt.Errorf("y must be numeric: %v", err)
	}

	radiusVal, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid radius: %v", err)
	}
	radius, err := basicValueToInt(radiusVal)
	if err != nil {
		return fmt.Errorf("radius must be numeric: %v", err)
	}

	// Optional ID parameter (defaults to 1 for backwards compatibility)
	id := 1
	if len(args) == 4 {
		idVal, err := b.evalExpression(args[3])
		if err != nil {
			return fmt.Errorf("invalid ID: %v", err)
		}
		id, err = basicValueToInt(idVal)
		if err != nil {
			return fmt.Errorf("ID must be numeric: %v", err)
		}
	}

	// Send physics command with visual properties
	err = b.sendPhysicsCommand("CIRCLE", map[string]interface{}{
		"x":      float64(x),
		"y":      float64(y),
		"radius": float64(radius),
		"id":     id,
	})
	if err != nil {
		return err
	}

	// Also send visual properties for rendering
	return b.sendPhysicsCommand("SET_VISUAL", map[string]interface{}{
		"id":    1, // Auto-assign ID 1 for the circle
		"shape": "circle",
		"color": "red",
		"size":  float64(radius),
	})
}

// handlePhysicsStep performs one physics step
func (b *TinyBASIC) handlePhysicsStep() error {
	return b.sendPhysicsCommand("STEP", map[string]interface{}{})
}

// handlePhysicsAuto enables/disables automatic physics updates
func (b *TinyBASIC) handlePhysicsAuto(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("PHYSICS AUTO requires ON or OFF")
	}

	state := strings.ToUpper(args[0])
	if state != "ON" && state != "OFF" {
		return fmt.Errorf("PHYSICS AUTO requires ON or OFF")
	}

	return b.sendPhysicsCommand("AUTO", map[string]interface{}{
		"enabled": state == "ON",
	})
}

// handlePhysicsVelocity sets object velocity
func (b *TinyBASIC) handlePhysicsVelocity(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("PHYSICS VELOCITY requires id, vx, vy")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	vxVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid vx: %v", err)
	}
	vx, err := basicValueToInt(vxVal)
	if err != nil {
		return fmt.Errorf("vx must be numeric: %v", err)
	}

	vyVal, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid vy: %v", err)
	}
	vy, err := basicValueToInt(vyVal)
	if err != nil {
		return fmt.Errorf("vy must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("VELOCITY", map[string]interface{}{
		"id": id,
		"vx": float64(vx),
		"vy": float64(vy),
	})
}

// handlePhysicsForce applies force to object
func (b *TinyBASIC) handlePhysicsForce(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("PHYSICS FORCE requires id, fx, fy")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	fxVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid fx: %v", err)
	}
	fx, err := basicValueToInt(fxVal)
	if err != nil {
		return fmt.Errorf("fx must be numeric: %v", err)
	}

	fyVal, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid fy: %v", err)
	}
	fy, err := basicValueToInt(fyVal)
	if err != nil {
		return fmt.Errorf("fy must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("FORCE", map[string]interface{}{
		"id": id,
		"fx": float64(fx),
		"fy": float64(fy),
	})
}

// handlePhysicsFriction sets object friction
func (b *TinyBASIC) handlePhysicsFriction(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS FRICTION requires id, friction")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	frictionVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid friction: %v", err)
	}
	friction, err := basicValueToInt(frictionVal)
	if err != nil {
		return fmt.Errorf("friction must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("FRICTION", map[string]interface{}{
		"id":       id,
		"friction": float64(friction),
	})
}

// handlePhysicsBounce sets object restitution (bounce)
func (b *TinyBASIC) handlePhysicsBounce(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS BOUNCE requires id, bounce")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	bounceVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid bounce: %v", err)
	}
	bounce, err := basicValueToInt(bounceVal)
	if err != nil {
		return fmt.Errorf("bounce must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("BOUNCE", map[string]interface{}{
		"id":     id,
		"bounce": float64(bounce),
	})
}

// handlePhysicsDensity sets object density
func (b *TinyBASIC) handlePhysicsDensity(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS DENSITY requires id, density")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	densityVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid density: %v", err)
	}
	density, err := basicValueToInt(densityVal)
	if err != nil {
		return fmt.Errorf("density must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("DENSITY", map[string]interface{}{
		"id":      id,
		"density": float64(density),
	})
}

// handlePhysicsGroup sets object collision group
func (b *TinyBASIC) handlePhysicsGroup(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS GROUP requires id, group")
	}

	idVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("id must be numeric: %v", err)
	}

	group := strings.Trim(args[1], "\"")

	return b.sendPhysicsCommand("GROUP", map[string]interface{}{
		"id":    id,
		"group": group,
	})
}

// handlePhysicsCollide enables/disables collision between groups
func (b *TinyBASIC) handlePhysicsCollide(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("PHYSICS COLLIDE requires group1, group2, enabled")
	}

	group1 := strings.Trim(args[0], "\"")
	group2 := strings.Trim(args[1], "\"")

	enabledVal, err := b.evalExpression(args[2])
	if err != nil {
		return fmt.Errorf("invalid enabled: %v", err)
	}
	enabled, err := basicValueToInt(enabledVal)
	if err != nil {
		return fmt.Errorf("enabled must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("COLLIDE", map[string]interface{}{
		"group1":  group1,
		"group2":  group2,
		"enabled": enabled != 0,
	})
}

// handlePhysicsCollision sets up collision detection callback
func (b *TinyBASIC) handlePhysicsCollision(args []string) error {
	if len(args) < 4 || strings.ToUpper(args[2]) != "THEN" {
		return fmt.Errorf("PHYSICS COLLISION requires id1, id2, THEN, line_number")
	}

	id1Val, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid id1: %v", err)
	}
	id1, err := basicValueToInt(id1Val)
	if err != nil {
		return fmt.Errorf("id1 must be numeric: %v", err)
	}

	id2Val, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid id2: %v", err)
	}
	id2, err := basicValueToInt(id2Val)
	if err != nil {
		return fmt.Errorf("id2 must be numeric: %v", err)
	}

	lineNumber, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Errorf("invalid line number: %v", err)
	}

	return b.sendPhysicsCommand("COLLISION", map[string]interface{}{
		"id1":        id1,
		"id2":        id2,
		"lineNumber": lineNumber,
	})
}

// sendPhysicsCommand sends a physics command to the frontend
func (b *TinyBASIC) sendPhysicsCommand(command string, params map[string]interface{}) error {
	message := shared.Message{
		Type:    shared.MessageTypePhysics,
		Command: command,
		Params:  params,
	}

	b.sendMessageObject(message)
	return nil
}

// handleSpritePhysicsCommand processes SPRITE PHYSICS commands
func (b *TinyBASIC) handleSpritePhysicsCommand(args string) error {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		return fmt.Errorf("SPRITE PHYSICS requires id, type, shape")
	}

	idVal, err := b.evalExpression(parts[0])
	if err != nil {
		return fmt.Errorf("invalid sprite id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("sprite id must be numeric: %v", err)
	}

	bodyType := strings.ToLower(strings.Trim(parts[1], "\""))
	shape := strings.ToLower(strings.Trim(parts[2], "\""))

	// Optional density parameter
	density := 1.0
	if len(parts) > 3 {
		densityVal, err := b.evalExpression(parts[3])
		if err != nil {
			return fmt.Errorf("invalid density: %v", err)
		}
		densityFloat, err := basicValueToInt(densityVal)
		if err != nil {
			return fmt.Errorf("density must be numeric: %v", err)
		}
		density = float64(densityFloat)
	}

	return b.sendPhysicsCommand("SPRITE", map[string]interface{}{
		"id":       id,
		"type":     bodyType,
		"shape":    shape,
		"density":  density,
	})
}

// handleVectorPhysicsCommand processes VECTOR PHYSICS commands
func (b *TinyBASIC) handleVectorPhysicsCommand(args string) error {
	parts := strings.Fields(args)
	if len(parts) < 3 {
		return fmt.Errorf("VECTOR PHYSICS requires id, type, shape")
	}

	idVal, err := b.evalExpression(parts[0])
	if err != nil {
		return fmt.Errorf("invalid vector id: %v", err)
	}
	id, err := basicValueToInt(idVal)
	if err != nil {
		return fmt.Errorf("vector id must be numeric: %v", err)
	}

	bodyType := strings.ToLower(strings.Trim(parts[1], "\""))
	shape := strings.ToLower(strings.Trim(parts[2], "\""))

	// Optional density parameter
	density := 1.0
	if len(parts) > 3 {
		densityVal, err := b.evalExpression(parts[3])
		if err != nil {
			return fmt.Errorf("invalid density: %v", err)
		}
		densityFloat, err := basicValueToInt(densityVal)
		if err != nil {
			return fmt.Errorf("density must be numeric: %v", err)
		}
		density = float64(densityFloat)
	}

	return b.sendPhysicsCommand("VECTOR", map[string]interface{}{
		"id":       id,
		"type":     bodyType,
		"shape":    shape,
		"density":  density,
	})
}

// handlePhysicsLink links a physics body to a VECTOR/SPRITE object
func (b *TinyBASIC) handlePhysicsLink(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("PHYSICS LINK requires physics_id and vector_id")
	}

	physicsIdVal, err := b.evalExpression(args[0])
	if err != nil {
		return fmt.Errorf("invalid physics ID: %v", err)
	}
	physicsId, err := basicValueToInt(physicsIdVal)
	if err != nil {
		return fmt.Errorf("physics ID must be numeric: %v", err)
	}

	vectorIdVal, err := b.evalExpression(args[1])
	if err != nil {
		return fmt.Errorf("invalid vector ID: %v", err)
	}
	vectorId, err := basicValueToInt(vectorIdVal)
	if err != nil {
		return fmt.Errorf("vector ID must be numeric: %v", err)
	}

	return b.sendPhysicsCommand("LINK", map[string]interface{}{
		"physics_id": physicsId,
		"vector_id":  vectorId,
	})
}