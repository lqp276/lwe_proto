package protoc

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestLexer(t *testing.T) {
	body, _ := ioutil.ReadFile("../data/test.proto")
	program := string(body)

	lex := newLexer(program)
	for {
		token := lex.getNextToken()
		if token == nil {
			fmt.Printf("parse error: %v", lex.getLastError())
			break
		}

		fmt.Printf("found token: %v\n", token)

		if token.type_ == EOF {
			break
		}
	}
}
