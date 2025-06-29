package tinybasic

// isTruthy returns true if the BASICValue is considered logically true.
func isTruthy(val BASICValue) bool {
	if val.IsNumeric {
		return val.NumValue != 0
	}
	return val.StrValue != "" && val.StrValue != "0"
}
