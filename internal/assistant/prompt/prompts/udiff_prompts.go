package prompts

func NewUdiffPrompts() *UdiffPrompts {
	prompts := &UdiffPrompts{BasePrompts: *NewBasePrompts()}
	prompts.Name = "Udiff"
	prompts.EditFormat = "udiff"
	prompts.MainSystem = `# File editing rules:

Return edits similar to unified diffs that ` + Fence + `diff -U0` + Fence + ` would produce.

Make sure you include the first 2 lines with the file paths.
Don't include timestamps with the file paths.

Start each hunk of changes with a ` + Fence + `@@ ... @@` + Fence + ` line.
Don't include line numbers like ` + Fence + `diff -U0` + Fence + ` does.
The user's patch tool doesn't need them.

The user's patch tool needs CORRECT patches that apply cleanly against the current contents of the file!
Think carefully and make sure you include and mark all lines that need to be removed or changed as ` + Fence + `-` + Fence + ` lines.
Make sure you mark all new or modified lines with ` + Fence + `+` + Fence + `.
Don't leave out any lines or the diff patch won't apply correctly.

Indentation matters in the diffs!

Start a new hunk for each section of the file that needs changes.

Only output hunks that specify changes with ` + Fence + `+` + Fence + ` or ` + Fence + `-` + Fence + ` lines.
Skip any hunks that are entirely unchanging ` + Fence + ` ` + Fence + ` lines.

Output hunks in whatever order makes the most sense.
Hunks don't need to be in any particular order.

When editing a function, method, loop, etc use a hunk to replace the *entire* code block.
Delete the entire existing version with ` + Fence + `-` + Fence + ` lines and then add a new, updated version with ` + Fence + `+` + Fence + ` lines.
This will help you generate correct code and correct diffs.

To move code within a file, use 2 hunks: 1 to delete it from its current location, 1 to insert it in the new location.

To make a new file, show a diff from ` + Fence + `--- /dev/null` + Fence + ` to ` + Fence + `+++ path/to/new/file.ext` + Fence + `.

{lazy_prompt}`
	prompts.SystemReminder = `To suggest changes to a file you MUST return the entire content of the updated file.
You MUST use this *file listing* format:

path/to/filename.js
{{.Fence0}}
// entire file content ...
// ... goes in between
{{.Fence1}}`
	prompts.RedactedEditMessage = "No changes are needed."
	prompts.ExampleMessages = getUdiffExampleMessages()
	return prompts
}

func getUdiffExampleMessages() []Message {
	return []Message{
		{Role: "user", Content: "Replace is_prime with a call to sympy"},
		{Role: "assistant", Content: `Ok, I will:

1. Add an imports of sympy.
2. Remove the is_prime() function.
3. Replace the existing call to is_prime() with a call to sympy.isprime().

Here are the diffs for those changes:

` + BlockFence + `diff
--- mathweb/flask/app.py
+++ mathweb/flask/app.py
@@ ... @@
-class MathWeb:
+import sympy
+
+class MathWeb:
@@ ... @@
-def is_prime(x):
-    if x < 2:
-        return False
-    for i in range(2, int(math.sqrt(x)) + 1):
-        if x % i == 0:
-            return False
-    return True
@@ ... @@
-@app.route('/prime/<int:n>')
-def nth_prime(n):
-    count = 0
-    num = 1
-    while count < n:
-        num += 1
-        if is_prime(num):
-            count += 1
-    return str(num)
+@app.route('/prime/<int:n>')
+def nth_prime(n):
+    count = 0
+    num = 1
+    while count < n:
+        num += 1
+        if sympy.isprime(num):
+            count += 1
+    return str(num)
        ` + BlockFence},
	}
}
