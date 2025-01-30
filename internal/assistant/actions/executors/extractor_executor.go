package executors

import (
	"context"
	"log/slog"

	"github.com/y0ug/codehaven/internal/assistant/actions"
	"github.com/y0ug/codehaven/internal/assistant/extractors"
)

type ExtractorExecutor struct {
	logger     *slog.Logger
	extractors []extractors.Extractor
}

func NewExtractorExecutor(
	logger *slog.Logger,
	e []extractors.Extractor,
) *ExtractorExecutor {
	return &ExtractorExecutor{
		logger:     logger,
		extractors: e,
	}
}

func (s *ExtractorExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.ActionExtractor)
	return ok
}

func (s *ExtractorExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) (actionResult actions.Action, results []actions.Action, err error) {
	a := action.Payload.(actions.ActionExtractor)

	// logger := actions.GetLogger(ctx)

	for _, e := range s.extractors {
		// TODO: We should pass the CTX
		r, err2 := e.Extract(&a.Msg)
		if err2 != nil {
			s.logger.Error("Error extracting", "error", err)
			continue
		}

		// Set the parents ID and append them to the results
		for _, cur := range r {
			results = append(results, cur.WithParent(&action))
		}
	}
	//  total := 10
	// return []actions.Action{
	// 	actions.NewLogAction(&action, fmt.Sprintf("extract :\n%s", string(output))),
	// }, err
	//
	actionResult = action
	return
}
