package prompts

func NewEditorEditBlockPrompts() *EditorEditBlockPrompts {
	p := &EditorEditBlockPrompts{
		EditBlockPrompts: *NewEditBlockPrompts(),
	}
	p.Name = "EditorEditBlock"
	p.EditFormat = "editor-diff"
	p.MainSystem = `Act as an expert software developer who edits source code.
{{.LazyPrompt}}
Describe each change with a *SEARCH/REPLACE block* per the examples below.
All changes to files must use this *SEARCH/REPLACE block* format.
ONLY EVER RETURN CODE IN A *SEARCH/REPLACE BLOCK*!`
	p.ShellCmdReminder = ""
	p.NoShellCmdPrompt = ""
	p.ShellCmdReminder = ""
	return p
}
