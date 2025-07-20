package tinybasic

import (
	"fmt"
	"github.com/antibyte/retroterm/pkg/shared"
)

// Cache monitoring and debugging commands for TinyBASIC

// cmdCacheStats displays cache performance statistics
func (b *TinyBASIC) cmdCacheStats() []shared.Message {
	stats := b.exprTokenCache.GetStats()
	
	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "=== Expression Token Cache Statistics ==="},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Cache Hits:       %d", stats.Hits)},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Cache Misses:     %d", stats.Misses)},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Hit Ratio:        %.2f%%", stats.HitRatio*100)},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Current Size:     %d/%d entries", stats.Size, stats.MaxSize)},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Evictions:        %d", stats.Evictions)},
		{Type: shared.MessageTypeText, Content: ""},
	}
	
	// Calculate cache effectiveness
	totalRequests := stats.Hits + stats.Misses
	if totalRequests > 0 {
		effectiveness := "Poor"
		if stats.HitRatio > 0.8 {
			effectiveness = "Excellent"
		} else if stats.HitRatio > 0.6 {
			effectiveness = "Good"
		} else if stats.HitRatio > 0.4 {
			effectiveness = "Fair"
		}
		
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Cache Effectiveness: %s", effectiveness),
		})
	}
	
	return messages
}

// cmdCacheTop displays the most frequently cached expressions
func (b *TinyBASIC) cmdCacheTop() []shared.Message {
	topExpressions := b.exprTokenCache.GetTopExpressions(10)
	
	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "=== Top 10 Cached Expressions ==="},
		{Type: shared.MessageTypeText, Content: "Rank | Hit Count | Tokens | Hash"},
		{Type: shared.MessageTypeText, Content: "-----+-----------+--------+------------------"},
	}
	
	for i, expr := range topExpressions {
		line := fmt.Sprintf("%4d | %9d | %6d | %16x", 
			i+1, expr.HitCount, expr.TokenCount, expr.Hash)
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: line,
		})
	}
	
	if len(topExpressions) == 0 {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "No cached expressions found.",
		})
	}
	
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "",
	})
	
	return messages
}

// cmdCacheClear clears the expression token cache
func (b *TinyBASIC) cmdCacheClear() []shared.Message {
	oldStats := b.exprTokenCache.GetStats()
	b.exprTokenCache.Clear()
	
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Expression token cache cleared."},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Removed %d cached expressions.", oldStats.Size)},
		{Type: shared.MessageTypeText, Content: ""},
	}
}

// cmdCacheConfig displays and allows modification of cache configuration
func (b *TinyBASIC) cmdCacheConfig(args []string) []shared.Message {
	if len(args) == 0 {
		// Display current configuration
		stats := b.exprTokenCache.GetStats()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeText, Content: "=== Cache Configuration ==="},
			{Type: shared.MessageTypeText, Content: fmt.Sprintf("Max Size:         %d entries", stats.MaxSize)},
			{Type: shared.MessageTypeText, Content: fmt.Sprintf("Current Size:     %d entries", stats.Size)},
			{Type: shared.MessageTypeText, Content: ""},
			{Type: shared.MessageTypeText, Content: "Usage: CACHECONFIG MAXSIZE <number>"},
			{Type: shared.MessageTypeText, Content: ""},
		}
	}
	
	if len(args) >= 2 && args[0] == "MAXSIZE" {
		// Parse new max size
		if newSize, err := parseInteger(args[1]); err == nil && newSize > 0 && newSize <= 10000 {
			b.exprTokenCache.SetMaxSize(int(newSize))
			return []shared.Message{
				{Type: shared.MessageTypeText, Content: ""},
				{Type: shared.MessageTypeText, Content: fmt.Sprintf("Cache max size set to %d entries.", newSize)},
				{Type: shared.MessageTypeText, Content: ""},
			}
		} else {
			return []shared.Message{
				{Type: shared.MessageTypeText, Content: "ERROR: Invalid max size. Must be 1-10000."},
			}
		}
	}
	
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "ERROR: Invalid cache configuration command."},
		{Type: shared.MessageTypeText, Content: "Usage: CACHECONFIG MAXSIZE <number>"},
	}
}

// Helper function to parse integers from strings
func parseInteger(s string) (int64, error) {
	var result int64 = 0
	var sign int64 = 1
	i := 0
	
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}
	
	if s[0] == '-' {
		sign = -1
		i = 1
	} else if s[0] == '+' {
		i = 1
	}
	
	for i < len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, fmt.Errorf("invalid character")
		}
		result = result*10 + int64(s[i]-'0')
		i++
	}
	
	return result * sign, nil
}

// Integration with existing command system
// These functions should be called from the main command dispatcher

// GetCacheMonitoringCommands returns a map of cache monitoring commands
func GetCacheMonitoringCommands() map[string]func(*TinyBASIC) []shared.Message {
	return map[string]func(*TinyBASIC) []shared.Message{
		"CACHESTATS": func(b *TinyBASIC) []shared.Message { return b.cmdCacheStats() },
		"CACHETOP":   func(b *TinyBASIC) []shared.Message { return b.cmdCacheTop() },
		"CACHECLEAR": func(b *TinyBASIC) []shared.Message { return b.cmdCacheClear() },
	}
}

// GetCacheConfigCommands returns commands that take arguments
func GetCacheConfigCommands() map[string]func(*TinyBASIC, []string) []shared.Message {
	return map[string]func(*TinyBASIC, []string) []shared.Message{
		"CACHECONFIG": func(b *TinyBASIC, args []string) []shared.Message { return b.cmdCacheConfig(args) },
	}
}