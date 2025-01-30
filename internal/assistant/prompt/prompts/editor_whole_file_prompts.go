package prompts

func NewEditorWholeFilePrompts() *EditorWholeFilePrompts {
	p := &EditorWholeFilePrompts{WholeFilePrompts: *NewWholeFilePrompts()}
	p.Name = "EditorWholeFile"
	p.EditFormat = "editor-whole"
	p.MainSystem = `Act as an expert software developer and make changes to source code.
{{.LazyPrompt}}
Output a copy of each file that needs changes.`
	return p
}
