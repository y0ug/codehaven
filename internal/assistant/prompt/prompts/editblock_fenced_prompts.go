package prompts

func NewEditBlockFencedPrompts() *EditBlockFencedPrompts {
	p := &EditBlockFencedPrompts{
		EditBlockPrompts: *NewEditBlockPrompts(),
	}
	p.Name = "EditBlockFenced"
	p.EditFormat = "block-diff-fenced"
	p.ExampleMessages = getEditBlockFencedExampleMessages()
	return p
}

func getEditBlockFencedExampleMessages() []Message {
	return []Message{
		{Role: "user", Content: "Change get_factorial() to use math.factorial"},
		{
			Role: "assistant",
			Content: `To make this change we need to modify ` + Fence + `mathweb/flask/app.py` + Fence + ` to:

1. Import the math package.
2. Remove the existing factorial() function.
3. Update get_factorial() to call math.factorial instead.

Here are the *SEARCH/REPLACE* blocks:

mathweb/flask/app.py
{{ .Fence0 }}python
<<<<<<< SEARCH
from flask import Flask
=======
import math
from flask import Flask
>>>>>>> REPLACE
{{.Fence1}}

mathweb/flask/app.py
{{.Fence0}}python
<<<<<<< SEARCH
def factorial(n):
    "compute factorial"

    if n == 0:
        return 1
    else:
        return n * factorial(n-1)

=======
>>>>>>> REPLACE
{{ .Fence1 }}

mathweb/flask/app.py
{{ .Fence0 }}python
<<<<<<< SEARCH
    return str(factorial(n))
=======
    return str(math.factorial(n))
>>>>>>> REPLACE
{{ .Fence1 }}
`,
		},
		{Role: "user", Content: "Refactor hello() into its own file."},
		{
			Role: "assistant",
			Content: `To make this change we need to modify ` + Fence + `main.py` + Fence + ` and make a new file ` + Fence + `hello.py` + Fence + `

1. Make a new hello.py file with hello() in it.
2. Remove hello() from main.py and replace it with an import.

Here are the *SEARCH/REPLACE* blocks:

hello.py
{{ .Fence0 }}python
<<<<<<< SEARCH
=======
def hello():
    "print a greeting"

    print("hello")
>>>>>>> REPLACE
{{ .Fence1 }}

main.py
{{ .Fence0 }}python
<<<<<<< SEARCH
def hello():
    "print a greeting"

    print("hello")
=======
from hello import hello
>>>>>>> REPLACE
{{ .Fence1 }}
`,
		},
	}
}
