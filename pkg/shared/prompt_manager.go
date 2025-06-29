package shared

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// PromptManager handles loading and managing system prompts with template support
type PromptManager struct {
	mcpCreateTemplate *template.Template
	mcpEditTemplate   *template.Template
	mcpChatTemplate   *template.Template
}

// TemplateData contains all variables that can be used in prompt templates
type TemplateData struct {
	Username    string
	Description string
	Instruction string
	FileContent string
	Timestamp   string
}

// NewPromptManager creates a new prompt manager and loads all prompt templates
func NewPromptManager() (*PromptManager, error) {
	pm := &PromptManager{}

	// Use current working directory to find prompts
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	promptsDir := filepath.Join(cwd, "prompts")

	// Load MCP CREATE prompt template
	createPath := filepath.Join(promptsDir, "mcp_create.txt")
	content, err := os.ReadFile(createPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP CREATE prompt from %s: %w", createPath, err)
	}
	pm.mcpCreateTemplate, err = template.New("mcp_create").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP CREATE template: %w", err)
	}

	// Load MCP EDIT prompt template
	editPath := filepath.Join(promptsDir, "mcp_edit.txt")
	content, err = os.ReadFile(editPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP EDIT prompt from %s: %w", editPath, err)
	}
	pm.mcpEditTemplate, err = template.New("mcp_edit").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP EDIT template: %w", err)
	}

	// Load MCP CHAT prompt template
	chatPath := filepath.Join(promptsDir, "mcp_chat.txt")
	content, err = os.ReadFile(chatPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load MCP CHAT prompt from %s: %w", chatPath, err)
	}
	pm.mcpChatTemplate, err = template.New("mcp_chat").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse MCP CHAT template: %w", err)
	}

	fmt.Printf("[PromptManager] Loaded all prompt templates successfully from %s\n", promptsDir)
	return pm, nil
}

// GetMCPCreatePrompt returns the MCP CREATE prompt with template variables filled
func (pm *PromptManager) GetMCPCreatePrompt(data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := pm.mcpCreateTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute MCP CREATE template: %w", err)
	}
	return buf.String(), nil
}

// GetMCPEditPrompt returns the MCP EDIT prompt with template variables filled
func (pm *PromptManager) GetMCPEditPrompt(data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := pm.mcpEditTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute MCP EDIT template: %w", err)
	}
	return buf.String(), nil
}

// GetMCPChatPrompt returns the MCP CHAT prompt with template variables filled
func (pm *PromptManager) GetMCPChatPrompt(data TemplateData) (string, error) {
	var buf bytes.Buffer
	if err := pm.mcpChatTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute MCP CHAT template: %w", err)
	}
	return buf.String(), nil
}
