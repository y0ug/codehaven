package prompts

func NewSingleWholeFileFunctionPrompts() *SingleWholeFileFunctionPrompts {
	p := &SingleWholeFileFunctionPrompts{BasePrompts: *NewBasePrompts()}
	p.Name = "SingleWholeFileFunction"
	p.EditFormat = "func-whole"
	p.MainSystem = `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'write_file' function to update the file to make the changes.
`
	p.SystemReminder = `ONLY return code using the 'write_file' function.
NEVER return code outside the 'write_file' function.
`
	p.RedactedEditMessage = "No changes are needed."
	p.RepoContentPrefix = ""
	return p
}
