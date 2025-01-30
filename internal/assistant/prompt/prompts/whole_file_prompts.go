package prompts

func NewWholeFilePrompts() *WholeFilePrompts {
	prompts := &WholeFilePrompts{BasePrompts: *NewBasePrompts()}
	prompts.Name = "WholeFile"
	prompts.EditFormat = "block-whole"
	prompts.MainSystem = `Act as an expert software developer.
Take requests for changes to the supplied code.
If the request is ambiguous, ask questions.

Always reply to the user in {{.Language}}.

{{.LazyPrompt}}
Once you understand the request you MUST:
1. Determine if any code changes are needed.
2. Explain any needed changes.
3. If changes are needed, output a copy of each file that needs changes.`
	prompts.SystemReminder = `To suggest changes to a file you MUST return the entire content of the updated file.
You MUST use this *file listing* format:

path/to/filename.js
{{.Fence0}}
// entire file content ...
// ... goes in between
{{.Fence1}}`
	prompts.RedactedEditMessage = "No changes are needed."
	prompts.ExampleMessages = getWholeFileExampleMessages()
	return prompts
}

func getWholeFileExampleMessages() []Message {
	return []Message{
		{Role: "user", Content: "Change the greeting to be more casual"},
		{Role: "assistant", Content: `Ok, I will:

        1. Switch the greeting text from "Hello" to "Hey".

        show_greeting.py
        ` + BlockFence + `
        import sys

        def greeting(name):
            print(f"Hey {name}")

        if __name__ == '__main__':
            greeting(sys.argv[1])
        ` + BlockFence},
	}
}
