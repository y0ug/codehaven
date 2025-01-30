package settings

import (
	"github.com/y0ug/codehaven/internal/assistant/extractors"
	"github.com/y0ug/codehaven/internal/assistant/llm/models"
)

type CoderSettings struct {
	mainModel            *models.Model
	chatLanguage         string
	autoLint             bool
	autoTest             bool
	testCmd              string
	maxOutputToken       int
	lintCommands         map[string]string
	suggestShellCommands bool
	verbose              bool

	// Generated field
	lastCommitHash string
	platformInfo   string
	isGit          bool
	fence          extractors.Fence
	rootPath       string
	stream         bool
}

func NewCoderSettings(mainModel *models.Model) *CoderSettings {
	return &CoderSettings{
		mainModel:            mainModel,
		autoLint:             true,
		autoTest:             true,
		testCmd:              "go test",
		maxOutputToken:       4096,
		lintCommands:         map[string]string{},
		suggestShellCommands: true,
		stream:               true,
	}
}

func (s *CoderSettings) SetMaxOutputToken(val int) {
	s.maxOutputToken = val
	if val, ok := s.mainModel.ExtraParams["max_tokens"]; ok {
		switch val := val.(type) {
		case int:
			if s.maxOutputToken > val || s.maxOutputToken == 0 {
				s.maxOutputToken = val
			}
		}
	}
}

func (s *CoderSettings) Stream() bool {
	return s.stream
}

func (s *CoderSettings) MainModel() *models.Model {
	return s.mainModel
}

func (s *CoderSettings) GetMaxOutputToken() int {
	return s.maxOutputToken
}

func (s *CoderSettings) GetModelName() string {
	return s.mainModel.Name
}

func (s *CoderSettings) getPlatformInfo() string {
	if s.platformInfo != "" {
		s.platformInfo = s.genPlatformInfo()
	}
	return s.platformInfo
}

func (s *CoderSettings) IsLazyModel() bool {
	return s.mainModel.Lazy
}

func (c *CoderSettings) getLanguage() string {
	if c.chatLanguage != "" {
		return c.chatLanguage
	}
	return "the same language they are using"
}

func (c *CoderSettings) GetTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Language": c.getLanguage(),
		"Platform": c.getPlatformInfo(),
		"Fence0":   c.fence[0],
		"Fence1":   c.fence[1],
	}
}

func (s *CoderSettings) Update(isGit bool, root string, fence [2]string) {
	s.platformInfo = s.getPlatformInfo()
}
