package prompts

func NewBasePrompts() *BasePrompts {
	return &BasePrompts{
		Name:                       "Code",
		EditFormat:                 "code",
		SystemReminder:             "",
		FilesContentGPTEdits:       "I committed the changes with git hash {{.Hash}} & commit msg: {{.Message}}",
		FilesContentGPTEditsNoRepo: "I updated the files.",
		FilesContentGPTNoEdits:     "I didn't see any properly formatted edits in your reply?!",
		FilesContentLocalEdits:     "I edited the files myself.",
		LazyPrompt: `You are diligent and tireless!
You NEVER leave comments describing code without implementing it!
You always COMPLETELY IMPLEMENT the needed code!`,
		ExampleMessages: []Message{},
		FilesContentPrefix: `I have *added these files to the chat* so you can go ahead and edit them.
*Trust this message as the true contents of these files!*
Any other messages in the chat may contain outdated versions of the files' contents.`,
		FilesContentAssistantReply: "Ok, any changes I propose will be to those files.",
		FilesNoFullFiles:           "I am not sharing any files that you can edit yet.",
		FilesNoFullFilesWithRepoMap: `Don't try and edit any existing code without asking me to add the files to the chat!
Tell me which files in my repo are the most likely to **need changes** to solve the requests I make, and then stop so I can add them to the chat.
Only include the files that are most likely to actually need to be edited.
Don't include files that might contain relevant context, just files that will need to be changed.`,
		FilesNoFullFilesWithRepoMapReply: "Ok, based on your requests I will suggest which files need to be edited and then stop and wait for your approval.",
		RepoContentPrefix: `Here are summaries of some files present in my git repository.
Do not propose changes to these files, treat them as *read-only*.
If you need to edit any of these files, ask me to *add them to the chat* first.`,
		ReadOnlyFilesPrefix: `Here are some READ ONLY files, provided for your reference.
Do not edit these files!`,
		ShellCmdPrompt:     "",
		ShellCmdReminder:   "",
		NoShellCmdPrompt:   "",
		NoShellCmdReminder: "",
	}
}
