package protoc

import (
	"fmt"
	"io/ioutil"
	"testing"
)

func TestParser(t *testing.T) {
	body, _ := ioutil.ReadFile("../data/test.proto")
	program := string(body)

	p := NewParser(program)

	fmt.Printf("parse result: %T, lastErr: %s\n", p.Program(), p.lastError)
}
