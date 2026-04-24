package ai

import (
	"testing"

	"ai-playground/config"
)

// TestMaxTokensSplit_DecisionUsesDecisionLimit proves that when a decision-type
// call is made, the outgoing MaxTokens is the decision limit, not the chat limit.
// Compile-driven RED: maxTokensFor does not exist yet.
func TestMaxTokensSplit_DecisionUsesDecisionLimit(t *testing.T) {
	a := &Agent{
		cfg: &config.AIConfig{
			MaxTokensChat:     160,
			MaxTokensDecision: 20,
		},
	}
	got := a.maxTokensFor("decision")
	if got != 20 {
		t.Errorf("decision max_tokens = %d, want 20", got)
	}
}

func TestMaxTokensSplit_ChatUsesChatLimit(t *testing.T) {
	a := &Agent{
		cfg: &config.AIConfig{
			MaxTokensChat:     160,
			MaxTokensDecision: 20,
		},
	}
	got := a.maxTokensFor("chat")
	if got != 160 {
		t.Errorf("chat max_tokens = %d, want 160", got)
	}
}
