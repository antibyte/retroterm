package tinybasic

import (
	"fmt" // Hinzugefügt für Logging
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Grafikbefehle für TinyBASIC

// cmdPlot implementiert den PLOT-Befehl: PLOT x, y, [color]
// Zeichnet einen einzelnen Pixel an den angegebenen Koordinaten
func (b *TinyBASIC) cmdPlot(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 2 || len(params) > 3 {
		return NewBASICError(ErrCategorySyntax, "INVALID_ARGUMENT", b.currentLine == 0, b.currentLine).WithCommand("PLOT").WithUsageHint("PLOT x, y [, color]")
	}

	// X-Koordinate auswerten
	xExpr := strings.TrimSpace(params[0])
	xVal, err := b.evalExpression(xExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
	}

	x, err := basicValueToInt(xVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
	} // Y-Koordinate auswerten
	yExpr := strings.TrimSpace(params[1])
	yVal, err := b.evalExpression(yExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
	}
	y, err := basicValueToInt(yVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
	}

	// Farb-Parameter auswerten (optional)
	color := "#5FFF5F" // Standardfarbe (grün)
	brightness := -1   // Kennzeichnet, dass keine numerische Helligkeit verwendet wurde

	if len(params) > 2 {
		colorExpr := strings.TrimSpace(params[2])
		colorVal, err := b.evalExpression(colorExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
		}

		// Versuche zuerst als numerische Helligkeit zu interpretieren
		if colorVal.IsNumeric {
			brightnessVal, bErr := convertValueToBrightness(colorVal)
			if bErr == nil {
				brightness = brightnessVal
				hexVal := fmt.Sprintf("%02x", brightness*17)
				color = "#" + hexVal + hexVal + hexVal
			}
		} else {
			// Nur wenn es nicht numerisch ist, als String interpretieren
			color, err = basicValueToString(colorVal)
			if err != nil {
				return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
			}
		}
	}

	plotParams := map[string]interface{}{
		"x":     x,
		"y":     y,
		"color": color,
	}

	// Füge brightness Parameter hinzu, wenn es eine numerische Helligkeit war
	if brightness >= 0 {
		plotParams["brightness"] = brightness
	}

	plotMsg := shared.Message{
		Type:    shared.MessageTypeGraphics,
		Command: "PLOT",
		Params:  plotParams,
	}
	if !b.sendMessageObject(plotMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("PLOT")
	}

	return nil
}

// cmdLine implementiert den LINE-Befehl: LINE x1, y1, x2, y2, [color]
// Zeichnet eine Linie zwischen zwei Punkten
func (b *TinyBASIC) cmdLine(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 4 || len(params) > 5 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("LINE").
			WithUsageHint("LINE x1, y1, x2, y2, [color]")
	}
	// Parameter auswerten
	values := make([]int, 4)
	paramNames := []string{"X1", "Y1", "X2", "Y2"}

	for i := 0; i < 4; i++ {
		expr := strings.TrimSpace(params[i])
		val, err := b.evalExpression(expr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("LINE").
				WithUsageHint(fmt.Sprintf("Error in %s parameter", paramNames[i]))
		}
		intVal, err := basicValueToInt(val)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
				WithCommand("LINE").
				WithUsageHint(fmt.Sprintf("%s must be numeric", paramNames[i]))
		}
		values[i] = intVal
	}
	// Farb-Parameter auswerten (optional)
	color := "#5FFF5F" // Standardfarbe
	brightness := -1   // Kennzeichnet, dass keine numerische Helligkeit verwendet wurde
	if len(params) > 4 {
		colorExpr := strings.TrimSpace(params[4])
		colorVal, err := b.evalExpression(colorExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("LINE").
				WithUsageHint("Error in color parameter")
		}

		// Versuche zuerst als numerische Helligkeit zu interpretieren
		if colorVal.IsNumeric {
			brightnessVal, bErr := convertValueToBrightness(colorVal)
			if bErr == nil {
				brightness = brightnessVal
				hexVal := fmt.Sprintf("%02x", brightness*17)
				color = "#" + hexVal + hexVal + hexVal
			}
		} else {
			// Nur wenn es nicht numerisch ist, als String interpretieren
			color, err = basicValueToString(colorVal)
			if err != nil {
				return NewBASICError(ErrCategorySyntax, "INVALID_COLOR_FORMAT", b.currentLine == 0, b.currentLine).
					WithCommand("LINE").
					WithUsageHint("Color must be a string (e.g., \"#FF0000\") or a numeric brightness (0-15)")
			}
		}
	}

	lineParams := map[string]interface{}{
		"x1":    values[0],
		"y1":    values[1],
		"x2":    values[2],
		"y2":    values[3],
		"color": color,
	}

	// Füge brightness Parameter hinzu, wenn es eine numerische Helligkeit war
	if brightness >= 0 {
		lineParams["brightness"] = brightness
	}
	lineMsg := shared.Message{
		Type:    shared.MessageTypeGraphics,
		Command: "LINE",
		Params:  lineParams,
	}

	if !b.sendMessageObject(lineMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("LINE")
	}
	return nil
}

// cmdRect implementiert den RECT-Befehl: RECT x, y, width, height, [color], [fill]
// Zeichnet ein Rechteck mit der linken oberen Ecke bei (x,y)
func (b *TinyBASIC) cmdRect(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 4 || len(params) > 6 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("RECT").
			WithUsageHint("RECT x, y, width, height, [color], [fill]")
	}

	// Parameter auswerten
	values := make([]int, 4) // x, y, width, height
	paramNames := []string{"X", "Y", "width", "height"}
	for i := 0; i < 4; i++ {
		expr := strings.TrimSpace(params[i])
		val, err := b.evalExpression(expr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("RECT").
				WithUsageHint(fmt.Sprintf("Error in %s parameter", paramNames[i]))
		}
		intVal, err := basicValueToInt(val)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
				WithCommand("RECT").WithUsageHint(fmt.Sprintf("%s must be numeric", paramNames[i]))
		}
		values[i] = intVal
	} // Farb-Parameter auswerten (optional)
	color := "#5FFF5F" // Standardfarbe
	brightness := -1   // Kennzeichnet, dass keine numerische Helligkeit verwendet wurde
	if len(params) > 4 {
		colorExpr := strings.TrimSpace(params[4])
		colorVal, err := b.evalExpression(colorExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("RECT").
				WithUsageHint("Error in color parameter")
		}

		// Versuche zuerst als numerische Helligkeit zu interpretieren
		if colorVal.IsNumeric {
			brightnessVal, bErr := convertValueToBrightness(colorVal)
			if bErr == nil {
				brightness = brightnessVal
				hexVal := fmt.Sprintf("%02x", brightness*17)
				color = "#" + hexVal + hexVal + hexVal
			}
		} else {
			// Nur wenn es nicht numerisch ist, als String interpretieren
			color, err = basicValueToString(colorVal)
			if err != nil {
				return NewBASICError(ErrCategorySyntax, "INVALID_COLOR_FORMAT", b.currentLine == 0, b.currentLine).
					WithCommand("RECT").
					WithUsageHint("Color must be a string (e.g., \"#FF0000\") or a numeric brightness (0-15)")
			}
		}
	}
	// Fill-Parameter auswerten (optional)
	fill := false // Standardwert: kein Füllen
	if len(params) > 5 {
		fillExpr := strings.TrimSpace(params[5])
		fillVal, err := b.evalExpression(fillExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("RECT").
				WithUsageHint("Error in fill parameter")
		}
		fillNum, err := basicValueToInt(fillVal)
		if err == nil {
			fill = (fillNum != 0)
		} else {
			fillStr, sErr := basicValueToString(fillVal)
			if sErr != nil {
				return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
					WithCommand("RECT").
					WithUsageHint("Fill must be a number or string (TRUE/FALSE)")
			}
			fillStr = strings.ToUpper(fillStr)
			fill = (fillStr == "TRUE" || fillStr == "T" || fillStr == "YES" || fillStr == "Y" || fillStr == "1")
		}
	}

	rectParams := map[string]interface{}{
		"x":      values[0],
		"y":      values[1],
		"width":  values[2],
		"height": values[3],
		"color":  color,
		"fill":   fill,
	}

	// Füge brightness Parameter hinzu, wenn es eine numerische Helligkeit war
	if brightness >= 0 {
		rectParams["brightness"] = brightness
	}

	rectMsg := shared.Message{
		Type:    shared.MessageTypeGraphics,
		Command: "RECT",
		Params:  rectParams,
	}
	if !b.sendMessageObject(rectMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("RECT")
	}
	return nil
}

// cmdCircle implementiert den CIRCLE-Befehl: CIRCLE x, y, radius, [color], [fill]
// Zeichnet einen Kreis mit dem Mittelpunkt (x,y) und dem gegebenen Radius
func (b *TinyBASIC) cmdCircle(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 3 || len(params) > 5 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("CIRCLE").
			WithUsageHint("CIRCLE x, y, radius, [color], [fill]")
	}

	// Parameter auswerten
	values := make([]int, 3) // x, y, radius
	paramNames := []string{"X", "Y", "radius"}
	for i := 0; i < 3; i++ {
		expr := strings.TrimSpace(params[i])
		val, err := b.evalExpression(expr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("CIRCLE").
				WithUsageHint(fmt.Sprintf("Error in %s parameter", paramNames[i]))
		}
		intVal, err := basicValueToInt(val)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
				WithCommand("CIRCLE").
				WithUsageHint(fmt.Sprintf("%s must be numeric", paramNames[i]))
		}
		values[i] = intVal
	}
	// Farb-Parameter auswerten (optional)
	color := "#5FFF5F" // Standardfarbe
	brightness := -1   // Kennzeichnet, dass keine numerische Helligkeit verwendet wurde
	if len(params) > 3 {
		colorExpr := strings.TrimSpace(params[3])
		colorVal, err := b.evalExpression(colorExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("CIRCLE").
				WithUsageHint("Error in color parameter")
		}

		// Versuche zuerst als numerische Helligkeit zu interpretieren
		if colorVal.IsNumeric {
			brightnessVal, bErr := convertValueToBrightness(colorVal)
			if bErr == nil {
				brightness = brightnessVal
				hexVal := fmt.Sprintf("%02x", brightness*17)
				color = "#" + hexVal + hexVal + hexVal
			}
		} else {
			// Nur wenn es nicht numerisch ist, als String interpretieren
			color, err = basicValueToString(colorVal)
			if err != nil {
				return NewBASICError(ErrCategorySyntax, "INVALID_COLOR_FORMAT", b.currentLine == 0, b.currentLine).
					WithCommand("CIRCLE").
					WithUsageHint("Color must be a string (e.g., \"#FF0000\") or a numeric brightness (0-15)")
			}
		}
	}

	// Fill-Parameter auswerten (optional)
	var fill bool = false // Standardwert: kein Füllen
	if len(params) > 4 {
		fillExpr := strings.TrimSpace(params[4])
		fillVal, err := b.evalExpression(fillExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("CIRCLE").
				WithUsageHint("Error in fill parameter")
		}
		fillNum, err := basicValueToInt(fillVal)
		if err == nil {
			fill = (fillNum != 0)
		} else {
			fillStr, sErr := basicValueToString(fillVal)
			if sErr != nil {
				return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
					WithCommand("CIRCLE").
					WithUsageHint("Fill must be a number or string (TRUE/FALSE)")
			}
			fillStr = strings.ToUpper(fillStr)
			fill = (fillStr == "TRUE" || fillStr == "T" || fillStr == "YES" || fillStr == "Y" || fillStr == "1")
		}
	}

	circleParams := map[string]interface{}{
		"x":      values[0],
		"y":      values[1],
		"radius": values[2],
		"color":  color,
		"fill":   fill,
	}

	// Füge brightness Parameter hinzu, wenn es eine numerische Helligkeit war
	if brightness >= 0 {
		circleParams["brightness"] = brightness
	}
	circleMsg := shared.Message{
		Type:    shared.MessageTypeGraphics,
		Command: "CIRCLE",
		Params:  circleParams,
	}
	if !b.sendMessageObject(circleMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("CIRCLE")
	}
	return nil
}

// cmdTextGFX implementiert den TEXTGFX-Befehl: TEXTGFX x, y, text, [color], [size]
// Zeichnet Text an den angegebenen Koordinaten mit der angegebenen Farbe und Größe
func (b *TinyBASIC) cmdTextGFX(args string) error {
	params := splitRespectingParentheses(strings.TrimSpace(args))
	if len(params) < 3 || len(params) > 5 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("TEXTGFX x, y, text, [color], [size]")
	}

	// Position auswerten
	xExpr := strings.TrimSpace(params[0])
	xVal, err := b.evalExpression(xExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("Error in X parameter")
	}

	x, err := basicValueToInt(xVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("X must be numeric")
	}

	yExpr := strings.TrimSpace(params[1])
	yVal, err := b.evalExpression(yExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("Error in Y parameter")
	}

	y, err := basicValueToInt(yVal)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("TEXTGFX").WithUsageHint("Y must be numeric")
	}

	// Text auswerten
	textExpr := strings.TrimSpace(params[2])
	textVal, err := b.evalExpression(textExpr)
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("Error in text parameter")
	}
	textToDraw, err := basicValueToString(textVal) // textToDraw hier definieren
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("Text parameter must be a string")
	}

	if textToDraw == "" { // textToDraw hier verwenden
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("TEXTGFX").
			WithUsageHint("Text parameter cannot be empty")
	}
	// Farb-Parameter auswerten (optional)
	color := "#5FFF5F" // Standardfarbe (grün)
	brightness := -1   // Kennzeichnet, dass keine numerische Helligkeit verwendet wurde
	if len(params) > 3 {
		colorExpr := strings.TrimSpace(params[3])
		colorVal, err := b.evalExpression(colorExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("TEXTGFX").
				WithUsageHint("Error in color parameter")
		}
		color, err = basicValueToString(colorVal)
		if err != nil {
			// Versuche, als Zahl zu interpretieren (für Abwärtskompatibilität mit Helligkeit)
			brightnessVal, bErr := convertValueToBrightness(colorVal)
			if bErr == nil {
				brightness = brightnessVal
				hexVal := fmt.Sprintf("%02x", brightness*17)
				color = "#" + hexVal + hexVal + hexVal
			} else {
				return NewBASICError(ErrCategorySyntax, "INVALID_COLOR_FORMAT", b.currentLine == 0, b.currentLine).
					WithCommand("TEXTGFX").
					WithUsageHint("Color must be a string (e.g., \"#FF0000\") or a numeric brightness (0-15)")
			}
		}
	}

	// Größen-Parameter auswerten (optional)
	size := 1 // Standardgröße
	if len(params) > 4 {
		sizeExpr := strings.TrimSpace(params[4])
		sizeVal, err := b.evalExpression(sizeExpr)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
				WithCommand("TEXTGFX").
				WithUsageHint("Error in size parameter")
		}
		size, err = basicValueToInt(sizeVal)
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
				WithCommand("TEXTGFX").
				WithUsageHint("Size must be numeric")
		}
	}

	textGfxMsg := shared.Message{
		Type:    shared.MessageTypeGraphics,
		Command: "TEXTGFX",
		Params: map[string]interface{}{
			"x":     x,
			"y":     y,
			"text":  textToDraw,
			"color": color,
			"size":  size},
	}
	if !b.sendMessageObject(textGfxMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("TEXTGFX")
	}
	return nil
}

// cmdCls implementiert den CLS-Befehl.
// Löscht den Bildschirm (Text- und Grafikbereich).
func (b *TinyBASIC) cmdCls(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("CLS").
			WithUsageHint("CLS does not take any arguments")
	}

	clsMsg := shared.Message{
		Type:    shared.MessageTypeClear, // MessageTypeClear für den gesamten Bildschirm
		Command: "CLS",                   // Behalte Command für Klarheit, auch wenn Type schon spezifisch ist
	}
	if !b.sendMessageObject(clsMsg) {
		// Hier könnte man überlegen, ob ein Fehler zurückgegeben wird, oder ob CLS "best effort" ist.
		// Für Konsistenz geben wir einen Fehler zurück.
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("CLS")
	}
	return nil
}

// cmdClearGraphics implementiert den CLEAR GRAPHICS-Befehl.
// Löscht nur den Grafikbereich.
func (b *TinyBASIC) cmdClearGraphics(args string) error {
	if args != "" {
		// CLEAR GRAPHICS sollte keine Argumente haben, aber wir könnten hier flexibel sein
		// oder einen strikten Syntaxfehler werfen.
		// Fürs Erste: Ignoriere Argumente oder werfe Fehler, wenn Argumente vorhanden sind.
		// Hier werfen wir einen Fehler für strikte Syntax.
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("CLEAR GRAPHICS").
			WithUsageHint("CLEAR GRAPHICS does not take any arguments")
	}

	clearGfxMsg := shared.Message{
		Type:    shared.MessageTypeGraphics, // Spezifischer als MessageTypeClear
		Command: "CLEAR_GRAPHICS",
		// Keine weiteren Parameter nötig für diesen Befehl
	}
	if !b.sendMessageObject(clearGfxMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("CLEAR GRAPHICS")
	}
	return nil
}

// convertValueToBrightness konvertiert einen BASICValue zu einer Helligkeit zwischen 0-15
func convertValueToBrightness(val BASICValue) (int, error) {
	if !val.IsNumeric {
		return 0, NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", false, 0).
			WithUsageHint("Value is not numeric")
	}

	// Runde float64 zu int
	brightness := int(val.NumValue + 0.5)

	// Begrenze auf 0-15
	if brightness < 0 {
		brightness = 0
	} else if brightness > 15 {
		brightness = 15
	}

	return brightness, nil
}

// cmdLocate implementiert den LOCATE x,y Befehl.
// Setzt die Cursor-Position für nachfolgende PRINT-Ausgaben.
func (b *TinyBASIC) cmdLocate(args string) error {
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("LOCATE requires X,Y coordinates")
	}
	// Parse Argumente: LOCATE x,y
	parts := splitRespectingParentheses(strings.TrimSpace(args))
	if len(parts) != 2 {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("LOCATE requires X,Y coordinates")
	}
	// X-Koordinate evaluieren
	xVal, err := b.evalExpression(strings.TrimSpace(parts[0]))
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("Error evaluating X coordinate")
	}
	if !xVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("X coordinate must be numeric")
	}

	// Y-Koordinate evaluieren
	yVal, err := b.evalExpression(strings.TrimSpace(parts[1]))
	if err != nil {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("Error evaluating Y coordinate")
	}
	if !yVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("Y coordinate must be numeric")
	}

	// Koordinaten in Integer konvertieren
	x := int(xVal.NumValue)
	y := int(yVal.NumValue)
	// Validiere Koordinaten (1-basiert für Benutzer, aber intern 0-basiert)
	if x < 1 || y < 1 {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint("Coordinates must be >= 1")
	}
	if x > b.termCols || y > b.termRows {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("LOCATE").
			WithUsageHint(fmt.Sprintf("Coordinates out of range (max %dx%d)", b.termCols, b.termRows))
	}

	// Setze Cursor-Position (konvertiere zu 0-basiert)
	b.cursorX = x - 1
	b.cursorY = y - 1 // Sende LOCATE-Nachricht an Frontend
	locateMsg := shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: fmt.Sprintf("%d,%d", b.cursorX, b.cursorY), // 0-basiert ans Frontend
		Command: "LOCATE",
	}

	if !b.sendMessageObject(locateMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("LOCATE")
	}

	return nil
}

// cmdInverse implementiert den INVERSE ON/OFF Befehl.
// Aktiviert oder deaktiviert den inversen Text-Modus.
func (b *TinyBASIC) cmdInverse(args string) error {
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_COUNT", b.currentLine == 0, b.currentLine).
			WithCommand("INVERSE").
			WithUsageHint("INVERSE requires ON or OFF")
	}

	arg := strings.ToUpper(strings.TrimSpace(args))
	var enable bool

	switch arg {
	case "ON":
		enable = true
	case "OFF":
		enable = false
	default:
		return NewBASICError(ErrCategorySyntax, "INVALID_PARAMETER_VALUE", b.currentLine == 0, b.currentLine).
			WithCommand("INVERSE").
			WithUsageHint("INVERSE requires ON or OFF")
	}

	// Setze Flag
	b.inverseTextMode = enable

	// Sende INVERSE-Nachricht an Frontend
	inverseMsg := shared.Message{
		Type:    shared.MessageTypeInverse,
		Content: arg, // "ON" oder "OFF"
		Command: "INVERSE",
	}

	if !b.sendMessageObject(inverseMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("INVERSE")
	}

	return nil
}
