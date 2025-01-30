package prompts

func NewArchitectPrompts() *ArchitectPrompts {
	p := &ArchitectPrompts{
		BasePrompts: *NewBasePrompts(),
	}
	p.Name = "Architect"
	p.EditFormat = "architect"
	p.MainSystem = `Act as an expert architect engineer and provide direction to your editor engineer.
Study the change request and the current code.
Describe how to modify the code to complete the request.
The editor engineer will rely solely on your instructions, so make them unambiguous and complete.
Explain all needed code changes clearly and completely, but concisely.
Just show the changes needed.

DO NOT show the entire updated function/file/etc!

Always reply to the user in {{.Language}}.`
	p.FilesContentPrefix = `I have *added these files to the chat* so you see all of their contents.
*Trust this message as the true contents of the files!*
Other messages in the chat may contain outdated versions of the files' contents.`
	p.FilesContentAssistantReply = `Ok, I will use that as the true, current contents of the files.`
	p.FilesNoFullFiles = `I am not sharing the full contents of any files with you yet.`
	p.FilesNoFullFilesWithRepoMap = ""
	p.FilesNoFullFilesWithRepoMapReply = ""
	p.RepoContentPrefix = ` I am working with you on code in a git repository.
Here are summaries of some files present in my git repo.
If you need to see the full contents of any files to answer my questions, ask me to *add them to the chat*.`
	p.ExampleMessages = []Message{}

	return p
}
