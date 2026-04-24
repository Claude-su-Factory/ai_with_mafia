package ai

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"go.uber.org/zap"

	"ai-playground/config"
	"ai-playground/internal/domain/entity"
)

type Agent struct {
	PlayerID    string
	Persona     Persona
	Role        entity.Role
	Allies      []string // 마피아 공범 ID 목록
	systemPrompt string
	history     []anthropic.MessageParam
	eventCh     chan entity.GameEvent
	outCh       chan AgentOutput
	cfg         *config.AIConfig
	client      *anthropic.Client
	logger      *zap.Logger
}

type AgentOutput struct {
	PlayerID   string
	PlayerName string
	Message    string
	MafiaOnly  bool
	// ActionType is set for game actions ("vote", "kill", "investigate").
	// Empty means the output is a chat message.
	ActionType string
	// TargetID is the player ID targeted by the action.
	TargetID string
}

func NewAgent(
	playerID string,
	persona Persona,
	role entity.Role,
	allies []string,
	cfg *config.AIConfig,
	client *anthropic.Client,
	logger *zap.Logger,
) *Agent {
	a := &Agent{
		PlayerID: playerID,
		Persona:  persona,
		Role:     role,
		Allies:   allies,
		eventCh:  make(chan entity.GameEvent, 32),
		outCh:    make(chan AgentOutput, 32),
		cfg:      cfg,
		client:   client,
		logger:   logger,
	}
	a.systemPrompt = buildSystemPrompt(persona, role, allies)
	return a
}

func (a *Agent) Output() <-chan AgentOutput {
	return a.outCh
}

func (a *Agent) Send(event entity.GameEvent) {
	select {
	case a.eventCh <- event:
	default:
		a.logger.Warn("agent event channel full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("player_id", a.PlayerID))
	}
}

func (a *Agent) Run(ctx context.Context) {
	for {
		select {
		case event := <-a.eventCh:
			a.handleEvent(ctx, event)
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) handleEvent(ctx context.Context, event entity.GameEvent) {
	switch event.Type {
	case entity.EventChat:
		a.onChat(ctx, event)
	case entity.EventPhaseChange:
		a.onPhaseChange(ctx, event)
	case entity.EventMafiaChat:
		if a.Role == entity.RoleMafia {
			a.onMafiaChat(ctx, event)
		}
	case entity.EventMafiaChannelOpen:
		msg, _ := event.Payload["message"].(string)
		if msg != "" {
			a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(
				"[시스템]: " + msg,
			)))
		}
	}
}

func (a *Agent) onChat(ctx context.Context, event entity.GameEvent) {
	senderName, _ := event.Payload["sender_name"].(string)
	message, _ := event.Payload["message"].(string)

	// 자신의 발언은 무시
	if event.PlayerID == a.PlayerID {
		return
	}

	userMsg := fmt.Sprintf("[%s]: %s", senderName, message)
	a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)))

	// LLM이 응답 여부 자율 판단 (응답 또는 [PASS])
	reply := a.callLLM(ctx, a.cfg.ModelDefault,
		"위 대화를 보고 지금 발언해야 한다면 발언 내용을, 발언할 필요가 없다면 정확히 [PASS]라고만 답하세요.")
	if reply == "" || strings.TrimSpace(reply) == "[PASS]" {
		return
	}

	a.addHistory(anthropic.NewAssistantMessage(anthropic.NewTextBlock(reply)))
	a.delayedOutput(ctx, AgentOutput{
		PlayerID:   a.PlayerID,
		PlayerName: a.Persona.Name,
		Message:    reply,
		MafiaOnly:  false,
	})
}

func (a *Agent) onMafiaChat(ctx context.Context, event entity.GameEvent) {
	senderName, _ := event.Payload["sender_name"].(string)
	message, _ := event.Payload["message"].(string)
	if event.PlayerID == a.PlayerID {
		return
	}

	userMsg := fmt.Sprintf("[마피아 채널] [%s]: %s", senderName, message)
	a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)))

	reply := a.callLLM(ctx, a.cfg.ModelDefault,
		"마피아 팀 채널입니다. 발언할 내용이 있으면 하고, 없으면 [PASS]라고만 답하세요.")
	if reply == "" || strings.TrimSpace(reply) == "[PASS]" {
		return
	}

	a.addHistory(anthropic.NewAssistantMessage(anthropic.NewTextBlock(reply)))
	a.delayedOutput(ctx, AgentOutput{
		PlayerID:   a.PlayerID,
		PlayerName: a.Persona.Name,
		Message:    reply,
		MafiaOnly:  true,
	})
}

func (a *Agent) onPhaseChange(ctx context.Context, event entity.GameEvent) {
	phase, _ := event.Payload["phase"].(string)
	switch entity.Phase(phase) {
	case entity.PhaseDayDiscussion:
		a.openDiscussion(ctx, event)
	case entity.PhaseDayVote:
		a.decideVote(ctx, event)
	case entity.PhaseNight:
		if a.Role == entity.RolePolice {
			a.decideInvestigate(ctx, event)
		}
		if a.Role == entity.RoleMafia {
			a.decideKill(ctx, event)
		}
	}
}

func (a *Agent) openDiscussion(ctx context.Context, event entity.GameEvent) {
	round, _ := event.Payload["round"].(int)
	var prompt string
	if round <= 1 {
		prompt = "낮 토론이 시작됐습니다. 게임 첫 라운드입니다. 자연스럽게 첫 인사나 의견을 한 문장으로 말하세요."
	} else {
		prompt = fmt.Sprintf("낮 토론 %d라운드가 시작됐습니다. 지금까지의 대화를 바탕으로 의심되는 점이나 관찰을 한 문장으로 말하세요.", round)
	}

	reply := a.callLLM(ctx, a.cfg.ModelDefault, prompt)
	if reply == "" || strings.TrimSpace(reply) == "[PASS]" {
		return
	}

	a.addHistory(anthropic.NewAssistantMessage(anthropic.NewTextBlock(reply)))
	a.delayedOutput(ctx, AgentOutput{
		PlayerID:   a.PlayerID,
		PlayerName: a.Persona.Name,
		Message:    reply,
		MafiaOnly:  false,
	})
}

func (a *Agent) decideVote(ctx context.Context, event entity.GameEvent) {
	alivePlayers, _ := event.Payload["alive_players"].([]string)
	prompt := fmt.Sprintf(
		"투표 시간입니다. 생존 플레이어: %s\n마피아로 가장 의심되는 사람 한 명의 ID만 답하세요.",
		strings.Join(alivePlayers, ", "),
	)
	a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))
	targetID := strings.TrimSpace(a.callLLM(ctx, a.cfg.ModelReasoning, ""))
	if targetID != "" && containsID(alivePlayers, targetID) {
		select {
		case a.outCh <- AgentOutput{
			PlayerID:   a.PlayerID,
			ActionType: "vote",
			TargetID:   targetID,
		}:
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) decideKill(ctx context.Context, event entity.GameEvent) {
	alivePlayers, _ := event.Payload["alive_players"].([]string)
	prompt := fmt.Sprintf(
		"밤입니다. 마피아로서 처치할 플레이어 한 명의 ID만 답하세요. 생존자: %s",
		strings.Join(alivePlayers, ", "),
	)
	a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))
	targetID := strings.TrimSpace(a.callLLM(ctx, a.cfg.ModelReasoning, ""))
	if targetID != "" && containsID(alivePlayers, targetID) {
		select {
		case a.outCh <- AgentOutput{
			PlayerID:   a.PlayerID,
			ActionType: "kill",
			TargetID:   targetID,
		}:
		case <-ctx.Done():
			return
		}
	}
}

func (a *Agent) decideInvestigate(ctx context.Context, event entity.GameEvent) {
	alivePlayers, _ := event.Payload["alive_players"].([]string)
	prompt := fmt.Sprintf(
		"밤입니다. 경찰로서 조사할 플레이어 한 명의 ID만 답하세요. 생존자: %s",
		strings.Join(alivePlayers, ", "),
	)
	a.addHistory(anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))
	targetID := strings.TrimSpace(a.callLLM(ctx, a.cfg.ModelReasoning, ""))
	if targetID != "" && containsID(alivePlayers, targetID) {
		select {
		case a.outCh <- AgentOutput{
			PlayerID:   a.PlayerID,
			ActionType: "investigate",
			TargetID:   targetID,
		}:
		case <-ctx.Done():
			return
		}
	}
}

// containsID reports whether target is in the ids slice.
func containsID(ids []string, target string) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// maxTokensFor returns the token limit appropriate for the given use-case.
// "chat" = free-form discussion; "decision" = short ID-only response
// (vote/kill/investigate). See phase-A design §3-A.
func (a *Agent) maxTokensFor(kind string) int {
	if kind == "decision" {
		return a.cfg.MaxTokensDecision
	}
	return a.cfg.MaxTokensChat
}

func (a *Agent) callLLM(ctx context.Context, model, extraInstruction string) string {
	messages := make([]anthropic.MessageParam, len(a.history))
	copy(messages, a.history)

	systemText := a.systemPrompt
	if extraInstruction != "" {
		systemText += "\n\n" + extraInstruction
	}

	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 300,
		System:    []anthropic.TextBlockParam{{Text: systemText}},
		Messages:  messages,
	})
	if err != nil {
		a.logger.Warn("claude api error", zap.String("agent", a.PlayerID), zap.Error(err))
		return ""
	}
	if len(resp.Content) == 0 {
		return ""
	}
	// ContentBlockUnion is a struct; access Text directly
	if resp.Content[0].Type == "text" {
		return resp.Content[0].Text
	}
	return ""
}

func (a *Agent) addHistory(msg anthropic.MessageParam) {
	a.history = append(a.history, msg)
	max := a.cfg.HistoryMax
	if max <= 0 {
		max = 40
	}
	if len(a.history) > max {
		a.history = a.history[len(a.history)-max:]
	}
}

func (a *Agent) delayedOutput(ctx context.Context, out AgentOutput) {
	min := a.cfg.ResponseDelayMin
	max := a.cfg.ResponseDelayMax
	if max <= min {
		max = min + 4
	}
	delay := time.Duration(min+rand.Intn(max-min)) * time.Second

	go func() {
		select {
		case <-time.After(delay):
			select {
			case a.outCh <- out:
			case <-ctx.Done():
			}
		case <-ctx.Done():
		}
	}()
}

func buildSystemPrompt(persona Persona, role entity.Role, allies []string) string {
	base := fmt.Sprintf(
		"당신의 이름은 %s입니다. 성격: %s\n"+
			"당신은 마피아 게임에 참여 중인 플레이어입니다. 절대로 AI임을 밝혀서는 안 됩니다. "+
			"자연스러운 한국어로 대화하세요.",
		persona.Name, persona.Personality,
	)

	switch role {
	case entity.RoleMafia:
		allyNames := strings.Join(allies, ", ")
		return base + fmt.Sprintf(
			"\n\n역할: 마피아. 공범: [%s]. "+
				"시민인 척하면서 의심을 피하세요. 공범과 협력하여 시민을 처치하세요. "+
				"절대 자신이 마피아임을 시민에게 드러내지 마세요.",
			allyNames,
		)
	case entity.RolePolice:
		return base + "\n\n역할: 경찰. 매 밤 한 명을 조사하여 마피아 여부를 확인할 수 있습니다. " +
			"조사 결과를 활용하여 마피아를 찾아내세요. 자신이 경찰임을 함부로 공개하지 마세요."
	default:
		return base + "\n\n역할: 시민. 대화와 논리로 마피아를 찾아내세요."
	}
}
