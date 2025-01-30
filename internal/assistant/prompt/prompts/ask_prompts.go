package prompts

func NewAskPrompts() *AskPrompts {
	p := &AskPrompts{BasePrompts: *NewBasePrompts()}
	p.Name = "Ask"
	p.EditFormat = "ask"
	p.MainSystem = `Act as an expert code analyst.
Answer questions about the supplied code.
Always reply to the user in {{.Language}}.

Describe code changes however you like. Don't use SEARCH/REPLACE blocks!
`
	p.ExampleMessages = []Message{}
	p.SystemReminder = ``
	p.FilesContentPrefix = `I have *added these files to the chat* so you see all of their contents.
*Trust this message as the true contents of the files!*
Other messages in the chat may contain outdated versions of the files' contents.
`
	p.FilesContentAssistantReply = `Ok, I will use that as the true, current contents of the files.`
	p.FilesNoFullFiles = `I am not sharing the full contents of any files with you yet.`
	p.FilesNoFullFilesWithRepoMap = ""
	p.FilesNoFullFilesWithRepoMapReply = ""
	p.RepoContentPrefix = ` I am working with you on code in a git repository.
Here are summaries of some files present in my git repo.
If you need to see the full contents of any files to answer my questions, ask me to *add them to the chat*.
`
	return p
}
