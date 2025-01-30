package prompts

func NewEditBlockFunctionPrompts() *EditBlockFunctionPrompts {
	p := &EditBlockFunctionPrompts{BasePrompts: *NewBasePrompts()}
	p.Name = "EditBlockFunction"
	p.MainSystem = `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Once you understand the request you MUST use the 'replace_lines' function to edit the files to make the needed changes.`
	p.SystemReminder = `
ONLY return code using the ` + Fence + `replace_lines` + Fence + ` function.
NEVER return code outside the ` + Fence + `replace_lines` + Fence + ` function.
`
	p.FilesContentPrefix = "Here is the current content of the files:\n"
	p.FilesNoFullFiles = "I am not sharing any files yet."

	p.RedactedEditMessage = "No changes are needed."
	p.RepoContentPrefix = `Below here are summaries of other files! Do not propose changes to these *read-only*
files without asking me first.
  `
	return p
}
