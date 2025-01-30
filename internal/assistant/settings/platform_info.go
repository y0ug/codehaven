package settings

import (
	"fmt"
	"os"
	"runtime"
)

func (c *CoderSettings) genPlatformInfo() string {
	var info string

	// Platform info
	info += fmt.Sprintf("- Platform: %s\n", runtime.GOOS)

	// Shell info
	shell := "SHELL"
	if runtime.GOOS == "windows" {
		shell = "COMSPEC"
	}
	info += fmt.Sprintf("- Shell: %s=%s\n", shell, os.Getenv(shell))

	// Language info
	if lang := c.chatLanguage; lang != "" {
		info += fmt.Sprintf("- Language: %s\n", lang)
	}

	// Git repo info
	if c.isGit {
		info += "- The user is operating inside a git repository\n"
	}

	// Lint commands
	if len(c.lintCommands) > 0 {
		if c.autoLint {
			info += "- The user's pre-commit runs these lint commands, don't suggest running them:\n"
		} else {
			info += "- The user prefers these lint commands:\n"
		}
		for lang, cmd := range c.lintCommands {
			if lang == "" {
				info += fmt.Sprintf("  - %s\n", cmd)
			} else {
				info += fmt.Sprintf("  - %s: %s\n", lang, cmd)
			}
		}
	}

	// Test command
	if c.testCmd != "" {
		if c.autoTest {
			info += fmt.Sprintf(
				"- The user's pre-commit runs this test command, don't suggest running them: %s\n",
				c.testCmd,
			)
		} else {
			info += fmt.Sprintf("- The user prefers this test command: %s\n", c.testCmd)
		}
	}

	return info
}
