package llm

import "github.com/y0ug/llmhaven/chat"

type Params struct {
	ChatParams      *chat.ChatParams
	StreamProcessor *StreamProcessor
	Stream          bool
}

func (p *Params) Update(opts ...func(*Params)) {
	for _, opt := range opts {
		opt(p)
	}
}

func NewParams(
	opts ...func(*Params),
) *Params {
	params := &Params{}
	params.Update(opts...)
	return params
}

func WithChatParams(opts ...func(*chat.ChatParams)) func(*Params) {
	return func(p *Params) {
		if p.ChatParams == nil {
			p.ChatParams = &chat.ChatParams{}
		}
		p.ChatParams.Update(opts...)
	}
}

func WithStreamProcessor(streamProcessor *StreamProcessor) func(*Params) {
	return func(p *Params) {
		p.StreamProcessor = streamProcessor
	}
}

func WithStream(stream bool) func(*Params) {
	return func(p *Params) {
		p.Stream = stream
	}
}
