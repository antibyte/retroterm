package tinybasic

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Maximum image dimensions
const (
	MaxImageWidth  = 2048
	MaxImageHeight = 2048
	MaxImageHandle = 8
)

// Green color palette (16 shades) - Brighter palette for better visibility
var greenPalette = []color.RGBA{
	{0, 32, 0, 255},      // 0: Dark green (not black)
	{0, 48, 0, 255},      // 1: Dark green
	{0, 64, 0, 255},      // 2: Dark green
	{0, 80, 0, 255},      // 3: Medium dark green
	{0, 96, 0, 255},      // 4: Medium dark green
	{0, 112, 0, 255},     // 5: Medium dark green
	{0, 128, 0, 255},     // 6: Medium green
	{0, 144, 0, 255},     // 7: Medium green
	{0, 160, 0, 255},     // 8: Medium bright green
	{0, 176, 0, 255},     // 9: Medium bright green
	{0, 192, 0, 255},     // 10: Bright green
	{0, 208, 0, 255},     // 11: Bright green
	{0, 224, 0, 255},     // 12: Very bright green
	{0, 240, 0, 255},     // 13: Very bright green
	{0, 255, 0, 255},     // 14: Pure bright green
	{0, 255, 0, 255},     // 15: Pure green
}

// convertToGreenShades converts an image to 16 green shades with improved quality
func convertToGreenShades(img image.Image) (*image.RGBA, error) {
	bounds := img.Bounds()
	converted := image.NewRGBA(bounds)
	width := bounds.Dx()
	height := bounds.Dy()

	// Create error diffusion matrix for Floyd-Steinberg dithering
	errorMatrix := make([][]float64, height)
	for i := range errorMatrix {
		errorMatrix[i] = make([]float64, width)
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			// Convert to grayscale using improved luminance formula
			gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0

			// Apply contrast enhancement (S-curve)
			gray = enhanceContrast(gray)

			// Add dithering error from previous pixels
			relX := x - bounds.Min.X
			relY := y - bounds.Min.Y
			if relX < width && relY < height {
				gray += errorMatrix[relY][relX]
			}

			// Clamp to [0, 1]
			if gray < 0 {
				gray = 0
			}
			if gray > 1 {
				gray = 1
			}

			// Map to palette index with better distribution
			paletteIndex := int(gray * 15.0 + 0.5) // Round instead of floor
			if paletteIndex > 15 {
				paletteIndex = 15
			}
			if paletteIndex < 0 {
				paletteIndex = 0
			}

			// Calculate quantization error for dithering
			actualGray := float64(paletteIndex) / 15.0
			quantError := gray - actualGray

			// Distribute error using Floyd-Steinberg dithering
			if relX < width-1 && relY < height {
				errorMatrix[relY][relX+1] += quantError * 7.0 / 16.0
			}
			if relX > 0 && relY < height-1 {
				errorMatrix[relY+1][relX-1] += quantError * 3.0 / 16.0
			}
			if relY < height-1 {
				errorMatrix[relY+1][relX] += quantError * 5.0 / 16.0
			}
			if relX < width-1 && relY < height-1 {
				errorMatrix[relY+1][relX+1] += quantError * 1.0 / 16.0
			}

			// Apply original alpha
			greenColor := greenPalette[paletteIndex]
			greenColor.A = uint8(a / 256)

			converted.Set(x, y, greenColor)
		}
	}
	return converted, nil
}

// enhanceContrast applies an S-curve to improve contrast
func enhanceContrast(gray float64) float64 {
	// S-curve formula: y = 3x² - 2x³ (for x in [0,1])
	// This enhances contrast by making dark areas darker and bright areas brighter
	return 3.0*gray*gray - 2.0*gray*gray*gray
}

// encodeImageToBase64 converts image to base64 PNG
func encodeImageToBase64(img image.Image) (string, error) {
	// Create a buffer to write PNG data
	var buf strings.Builder
	encoder := base64.NewEncoder(base64.StdEncoding, &buf)

	// Encode as PNG
	err := png.Encode(encoder, img)
	if err != nil {
		return "", err
	}
	encoder.Close()

	return buf.String(), nil
}

// cmdImageOpen implements the IMAGE OPEN command:
// IMAGE OPEN "filename", handle
func (b *TinyBASIC) cmdImageOpen(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 2 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("IMAGE OPEN \"filename\", handle")
	}

	// Parse filename
	filenameExpr := strings.TrimSpace(params[0])
	filenameVal, err := b.evalExpression(filenameExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Error in filename parameter")
	}
	if filenameVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Filename must be a string")
	}
	filename := strings.Trim(filenameVal.StrValue, "\"' \t")

	// Parse handle
	handleExpr := strings.TrimSpace(params[1])
	handleVal, err := b.evalExpression(handleExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Error in handle parameter")
	}
	handle, err := basicValueToInt(handleVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Handle must be numeric")
	}
	if handle < 1 || handle > MaxImageHandle {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint(fmt.Sprintf("Handle must be between 1 and %d", MaxImageHandle))
	}

	// Check if file exists and is PNG
	if !strings.HasSuffix(strings.ToLower(filename), ".png") {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_FILE_FORMAT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Only PNG files are supported")
	}

	// Try different paths for the file
	var imagePath string
	possiblePaths := []string{
		filename,                                    // Direct path
		filepath.Join("images", filename),           // images/ directory
		filepath.Join("examples", filename),         // examples/ directory
		filepath.Join("dyson", filename),           // dyson/ directory
	}

	var file *os.File
	for _, path := range possiblePaths {
		if f, err := os.Open(path); err == nil {
			file = f
			imagePath = path
			break
		}
	}

	if file == nil {
		return NewBASICError(ErrCategoryEvaluation, "FILE_NOT_FOUND", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint(fmt.Sprintf("File not found: %s", filename))
	}
	defer file.Close()

	// Decode PNG
	img, err := png.Decode(file)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_IMAGE_FORMAT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Failed to decode PNG file")
	}

	// Check image dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width > MaxImageWidth || height > MaxImageHeight {
		return NewBASICError(ErrCategoryEvaluation, "IMAGE_TOO_LARGE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint(fmt.Sprintf("Image too large. Maximum size: %dx%d", MaxImageWidth, MaxImageHeight))
	}

	// Convert to green shades
	convertedImg, err := convertToGreenShades(img)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "IMAGE_CONVERSION_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Failed to convert image to green shades")
	}

	// Encode to base64
	base64Data, err := encodeImageToBase64(convertedImg)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "IMAGE_ENCODING_ERROR", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE OPEN").
			WithUsageHint("Failed to encode image")
	}

	// Send to frontend
	b.sendImageCommand(shared.Message{
		Type:    shared.MessageTypeImage,
		Command: "LOAD_IMAGE",
		ID:      handle,
		CustomData: map[string]interface{}{
			"imageData": base64Data,
			"width":     width,
			"height":    height,
			"filename":  filename,
			"path":      imagePath,
		},
	})

	return nil
}

// cmdImageShow implements the IMAGE SHOW command:
// IMAGE SHOW handle, x, y, [scale]
func (b *TinyBASIC) cmdImageShow(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 3 || len(params) > 4 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE SHOW").
			WithUsageHint("IMAGE SHOW handle, x, y, [scale]")
	}

	// Parse handle
	handle, err := b.evalNumericParam(params[0], "handle")
	if err != nil {
		return err
	}
	if int(handle) < 1 || int(handle) > MaxImageHandle {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE SHOW").
			WithUsageHint(fmt.Sprintf("Handle must be between 1 and %d", MaxImageHandle))
	}

	// Parse position
	x, err := b.evalNumericParam(params[1], "x position")
	if err != nil {
		return err
	}
	y, err := b.evalNumericParam(params[2], "y position")
	if err != nil {
		return err
	}

	// Parse optional scale
	scale := 0.0 // Default: original size
	if len(params) >= 4 {
		scale, err = b.evalNumericParam(params[3], "scale")
		if err != nil {
			return err
		}
		if scale < -2.0 || scale > 2.0 {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
				WithCommand("IMAGE SHOW").
				WithUsageHint("Scale must be between -2.0 and 2.0")
		}
	}

	// Send to frontend
	b.sendImageCommand(shared.Message{
		Type:    shared.MessageTypeImage,
		Command: "SHOW_IMAGE",
		ID:      int(handle),
		Position: map[string]float64{
			"x": x,
			"y": y,
		},
		Scale: scale,
		Visible: boolPtr(true),
	})

	return nil
}

// cmdImageHide implements the IMAGE HIDE command:
// IMAGE HIDE handle
func (b *TinyBASIC) cmdImageHide(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 1 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE HIDE").
			WithUsageHint("IMAGE HIDE handle")
	}

	// Parse handle
	handle, err := b.evalNumericParam(params[0], "handle")
	if err != nil {
		return err
	}
	if int(handle) < 1 || int(handle) > MaxImageHandle {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE HIDE").
			WithUsageHint(fmt.Sprintf("Handle must be between 1 and %d", MaxImageHandle))
	}

	// Send to frontend
	b.sendImageCommand(shared.Message{
		Type:    shared.MessageTypeImage,
		Command: "HIDE_IMAGE",
		ID:      int(handle),
		Visible: boolPtr(false),
	})

	return nil
}

// cmdImageRotate implements the IMAGE ROTATE command:
// IMAGE ROTATE handle, rotation
func (b *TinyBASIC) cmdImageRotate(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) != 2 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE ROTATE").
			WithUsageHint("IMAGE ROTATE handle, rotation")
	}

	// Parse handle
	handle, err := b.evalNumericParam(params[0], "handle")
	if err != nil {
		return err
	}
	if int(handle) < 1 || int(handle) > MaxImageHandle {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE ROTATE").
			WithUsageHint(fmt.Sprintf("Handle must be between 1 and %d", MaxImageHandle))
	}

	// Parse rotation
	rotation, err := b.evalNumericParam(params[1], "rotation")
	if err != nil {
		return err
	}
	if rotation < -360 || rotation > 360 {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE ROTATE").
			WithUsageHint("Rotation must be between -360 and 360 degrees")
	}

	// Send to frontend
	b.sendImageCommand(shared.Message{
		Type:    shared.MessageTypeImage,
		Command: "ROTATE_IMAGE",
		ID:      int(handle),
		VecRotation: map[string]float64{
			"z": rotation * 3.14159 / 180, // Convert to radians
		},
	})

	return nil
}

// sendImageCommand sends an image command to the frontend
func (b *TinyBASIC) sendImageCommand(msg shared.Message) {
	b.sendMessageObject(msg)
}

// handleImageCommand dispatches IMAGE subcommands
func (b *TinyBASIC) handleImageCommand(args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_SUBCOMMAND", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE").
			WithUsageHint("Available: IMAGE OPEN, IMAGE SHOW, IMAGE HIDE, IMAGE ROTATE")
	}

	// Find the first space to separate subcommand from args
	parts := strings.SplitN(args, " ", 2)
	subcommand := strings.ToUpper(strings.TrimSpace(parts[0]))
	subargs := ""
	if len(parts) > 1 {
		subargs = strings.TrimSpace(parts[1])
	}

	switch subcommand {
	case "OPEN":
		return b.cmdImageOpen(subargs)
	case "SHOW":
		return b.cmdImageShow(subargs)
	case "HIDE":
		return b.cmdImageHide(subargs)
	case "ROTATE":
		return b.cmdImageRotate(subargs)
	default:
		return NewBASICError(ErrCategorySyntax, "UNKNOWN_SUBCOMMAND", b.currentLine == 0, b.currentLine).
			WithCommand("IMAGE " + subcommand).
			WithUsageHint("Available: IMAGE OPEN, IMAGE SHOW, IMAGE HIDE, IMAGE ROTATE")
	}
}