package executors

import (
	"context"
	"fmt"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	conversation "github.com/y0ug/codehaven/internal/assistant/conversation"
)

type LLMResponseExecutor struct {
	conversation *conversation.ConversationManager
}

func NewLLMResponseExecutor(conversation *conversation.ConversationManager) *LLMResponseExecutor {
	return &LLMResponseExecutor{
		conversation: conversation,
	}
}

func (e *LLMResponseExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.LLMResponseAction)
	return ok
}

func (e *LLMResponseExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actionResult actions.Action, results []actions.Action, err error) {
	logger := actions.GetLogger(ctx)

	val := action.Payload.(actions.LLMResponseAction)
	resp := val.Resp

	if len(resp.Choice) == 0 {
		return action, results, fmt.Errorf("no choice returned from LLM")
	}

	choice := resp.Choice[0]
	msg := resp.ToMessageParams()

	if msg.Role != "assistant" {
		return action, results, fmt.Errorf("last should be from assistant")
	}

	// Create an LLMResponseAction, child of parentAction
	// var rawText string
	// if len(msg.Content) > 0 {
	// 	_ = msg.Content[0].String()
	// }

	// Add to curernt chat history
	e.conversation.AddMessage(*msg)

	// We are pushing to the event action to extract action
	// from the LLM response. We alsa pass the parent action
	results = append(results, actions.NewActionExtractor(*msg).WithParent(&action))

	// The idea was if LLM stop reason is end_turn, we move the current messages to done
	// So any new llm request will start rebuilding the prompt states.
	// After the refactoring we don't know at this point
	// if we shouild reset the history context or not.
	// But we should maybe keep track of the current StopReason to reused it later?
	// maybe in message history
	if choice.StopReason == "end_turn" {
		e.conversation.EndTurn()
		// We do not set an messsages here, this should be done by
		// an action at the end for example if commit succeeded
		// e.history.MoveCurrentToDone("")
	} else {
		e.conversation.SetTurn(choice.StopReason)
	}

	logger.Debug(
		"End Processing response",
		"stop",
		choice.StopReason,
	)
	return action, results, nil
}
