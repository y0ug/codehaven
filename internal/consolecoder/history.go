package consolecoder

import (
	"bufio"
	"os"
)

func (c *Console) loadHistory() []string {
	var history []string
	file, err := os.OpenFile(c.historyFile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return history
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			history = append(history, line)
		}
	}
	return history
}

func (c *Console) appendHistory(input string) {
	if input == "" {
		return
	}

	file, err := os.OpenFile(c.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	file.WriteString(input + "\n")
}
