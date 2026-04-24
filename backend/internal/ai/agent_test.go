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

// TestCallLLM_TruncationEmitsHook verifies that when the agent surfaces a
// truncation event, the onUsage hook receives Truncated=true so downstream
// metrics can increment the truncated_turns counter.
func TestCallLLM_TruncationEmitsHook(t *testing.T) {
	var got AIUsage
	a := &Agent{
		onUsage: func(u AIUsage) { got = u },
	}
	a.recordUsage(AIUsage{Truncated: true, TokensIn: 10, TokensOut: 3})
	if !got.Truncated || got.TokensIn != 10 || got.TokensOut != 3 {
		t.Errorf("usage = %+v, want Truncated=true TokensIn=10 TokensOut=3", got)
	}
}
