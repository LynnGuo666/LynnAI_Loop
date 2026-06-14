package models

import "time"

const (
	ProtocolAnthropicMessages     = "anthropic_messages"
	ProtocolOpenAIChatCompletions = "openai_chat_completions"
	ProtocolOpenAIResponses       = "openai_responses"
	ProtocolGeminiGenerateContent = "gemini_generate_content"
)

type Channel struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	BaseURL     string    `json:"base_url"`
	Protocol    string    `json:"protocol"`
	Description string    `json:"description"`
	ProbeModel  string    `json:"probe_model"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
