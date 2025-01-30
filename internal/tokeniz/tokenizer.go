package tokeniz

import (
	"fmt"
	"log"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
)

type Counter func(input string) int

func TokenizerCounter(tk *tokenizer.Tokenizer) Counter {
	return func(input string) int {
		en, err := tk.EncodeSingle(input)
		if err != nil {
			return 0
		}
		return len(en.Tokens)
	}
}

func CountAntropic() {
	// configFile, err := tokenizer.CachedPath("bert-base-uncased", "tokenizer.json")
	// if err != nil {
	// 	panic(err)
	// }

	configFile := "anthropic_tokenizer.json"
	tk, err := pretrained.FromFile(configFile)
	if err != nil {
		panic(err)
	}
	sentence := `The Gophers craft code using [MASK] language.`
	en, err := tk.EncodeSingle(sentence)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tokens: %q\n", en.Tokens)
}
